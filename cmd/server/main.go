package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"google.golang.org/grpc"
	_ "google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/reflection"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"

	server "github.com/AntonBezemskiy/go-musthave-metrics/internal/grpc/api/server/impl"
	rpcHasher "github.com/AntonBezemskiy/go-musthave-metrics/internal/grpc/api/server/interceptors/hasher"
	rpcIPfilter "github.com/AntonBezemskiy/go-musthave-metrics/internal/grpc/api/server/interceptors/ipfilter"
	rpcLogger "github.com/AntonBezemskiy/go-musthave-metrics/internal/grpc/api/server/interceptors/logger"
	pb "github.com/AntonBezemskiy/go-musthave-metrics/internal/grpc/protoc"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/repositories"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/compress"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/encrypt"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/handlers"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/hasher"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/ipfilter"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/logger"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/pg"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/saver"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/storage"
)

const shutdownWaitPeriod = 20 * time.Second // для установки в контекст для реализаации graceful shutdown

func main() {
	// вывод глобальной информации о сборке
	printGlobalInfo(os.Stdout)

	saveMode := parseFlags()

	// Подключение к базе данных
	db, err := sql.Open("pgx", flagDatabaseDsn)
	if err != nil {
		log.Fatalf("Error connection to database: %v by address %s", err, flagDatabaseDsn)
	}
	defer db.Close()

	// Создаю разные хранилища в зависимости от типа запуска сервера
	var stor repositories.IStorage
	if saveMode == SAVEINDATABASE {
		// создаём соединение с СУБД PostgreSQL с помощью аргумента командной строки
		conn, err := sql.Open("pgx", flagDatabaseDsn)
		if err != nil {
			log.Fatalf("Error create database connection for saving metrics : %v\n", err)
		}
		// Проверка соединения с БД
		ctx := context.Background()
		err = db.PingContext(ctx)
		if err != nil {
			log.Fatalf("Error checking connection with database: %v\n", err)
		}
		// создаем экземпляр хранилища pg
		stor = pg.NewStore(conn)
		err = stor.Bootstrap(ctx)
		if err != nil {
			log.Fatalf("Error prepare database to work: %v\n", err)
		}
	} else {
		stor = storage.NewDefaultMemStorage()
	}

	// В случае запуска сервера в режиме сохранения метрик в файл
	var saverVar saver.FileWriter
	if saveMode == SAVEINFILE {
		// Для загрузки метрик из файла на сервер
		var err error
		saverVar, err = saver.NewWriter(saver.GetFilestoragePath())
		if err != nil {
			log.Fatalf("Error create writer for saving metrics : %v\n", err)
		}
	}

	if err := run(stor, saverVar, db, saveMode); err != nil {
		log.Fatalf("Error starting server: %v\n", err)
	}

	if saveMode == SAVEINFILE {
		// При штатном завершении работы сервера накопленные данные сохраняются в файл
		if err := saverVar.WriteMetrics(stor); err != nil {
			logger.ServerLog.Error("flushing metrics error", zap.String("error", error.Error(err)))
		}
	}
	log.Println("Shutdown the server gracefully")
}

// run полезна при инициализации зависимостей сервера перед запуском.
func run(stor repositories.IStorage, saverVar saver.FileWriter, db *sql.DB, saveMode int) error {
	if err := logger.Initialize(flagLogLevel); err != nil {
		return err
	}

	var reader saver.FileReader
	var err error
	if saveMode == SAVEINFILE {
		reader, err = saver.NewReader(saver.GetFilestoragePath())
		if err != nil {
			logger.ServerLog.Error("create writer for saving metrics error", zap.String("error", error.Error(err)))
			return err
		}
	}

	// Работаю с файлом для сохранения метрик, только в случае соответствующей конфигурации запуска сервера
	if saveMode == SAVEINFILE {
		// Загружаю на сервер метрики из файла, сохраненные в предыдущих запусках
		err := saver.AddMetricsFromFile(stor, reader)
		if err != nil {
			logger.ServerLog.Error("add metrics from file error", zap.String("error", error.Error(err)))
			return err
		}
		go FlushMetricsToFile(stor, saverVar)
	}

	// запускаю сам сервис с проверкой отмены контекста для реализации graceful shutdown--------------
	srv := &http.Server{
		Addr:    flagNetAddr,
		Handler: MetricRouter(stor, db),
	}
	// Канал для получения сигнала прерывания
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	// Горутина для запуска http сервера-----------------------------------------------
	go func() {
		logger.ServerLog.Info("Running http server", zap.String("address", flagNetAddr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error starting http server: %v", err)
		}
	}()

	// Подготовка grpc сервера----------------------------------------------------------
	lis, err := net.Listen("tcp", flagGRPCNetAddr)
	if err != nil {
		return fmt.Errorf("error starting gRPC server: %v", err)
	}
	opts := []logging.Option{
		// Логирование конца вызова
		logging.WithLogOnEvents(logging.FinishCall),
	}
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			logging.UnaryServerInterceptor(rpcLogger.Logger(logger.ServerGRPCLog), opts...),
			rpcHasher.UnaryServerInterceptor,
			rpcIPfilter.UnaryServerInterceptor,
			// Add any other interceptor.
		),
	)
	pb.RegisterServiceServer(grpcServer, server.NewServer(stor))
	reflection.Register(grpcServer)

	// Горутина для запуска grpc сервера-----------------------------------------------
	go func() {
		logger.ServerLog.Info("Running grpc server", zap.String("address", flagGRPCNetAddr))
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Error starting grpc server: %v", err)
		}
	}()

	// Блокирование до тех пор, пока не поступит сигнал о прерывании
	<-quit
	log.Println("Shutting down server...")

	// Create a context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), shutdownWaitPeriod)
	defer cancel()

	// группа синхронизации для ожидания мягкого завершения серверов
	var wg sync.WaitGroup

	// горутина для остановки http сервера, чтобы он перестал принимать новые запросы
	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		if err := srv.Shutdown(ctx); err != nil {
			logger.ServerLog.Error("Stopping server error: %v", zap.String("error", error.Error(err)))
		}
		defer wg.Done()
	}(&wg)

	// останавливаю grpc сервер, чтобы он перестал принимать новые запросы
	wg.Add(1)
	go func(ctx context.Context, wg *sync.WaitGroup) {
		// Создаю канал для уведомления о завершении GracefulStop
		done := make(chan struct{})

		// Горутина, которая отлеживает завершение контекста и в случае его отмены принудительно уменьшает
		// счетчик WaitGroup, что позволяет завершить работу программы, если grpcServer.GracefulStop() долго не
		// возвращает управление.
		go func(ctx context.Context, wg *sync.WaitGroup, done chan struct{}) {
			select {
			case <-done: // GracefulStop завершился
				wg.Done()
			case <-ctx.Done(): // GracefulStop ещё не завершился, но уже отменился контекст
				logger.ServerLog.Error("failed to stop grpc server gracefully, context is exceeded")
				wg.Done()
			}
		}(ctx, wg, done)

		grpcServer.GracefulStop()
		close(done)
	}(ctx, &wg)

	// Ожидание завершения работы серверов.
	wg.Wait()
	return nil
}

// MetricRouter - дирежирует обработку http запросов к серверу.
func MetricRouter(stor repositories.IStorage, db *sql.DB) chi.Router {
	r := chi.NewRouter()

	r.Route("/", func(r chi.Router) {
		r.Get("/", logger.RequestLogger(compress.GzipMiddleware(handlers.GetGlobalHandler(stor))))
		r.Get("/ping", logger.RequestLogger(compress.GzipMiddleware(handlers.PingDatabaseHandler(db))))

		r.Post("/updates/", logger.RequestLogger(ipfilter.Middleware(encrypt.Middleware(compress.GzipMiddleware(
			hasher.HashMiddleware(handlers.UpdateMetricsBatchHandler(stor)))))))
		r.Route("/update", func(r chi.Router) {
			r.Post("/", logger.RequestLogger(ipfilter.Middleware(encrypt.Middleware(compress.GzipMiddleware(
				hasher.HashMiddleware(handlers.UpdateMetricsJSONHandler(stor)))))))
			r.Post("/{metricType}/{metricName}/{metricValue}", logger.RequestLogger(
				ipfilter.Middleware(encrypt.Middleware(compress.GzipMiddleware(hasher.HashMiddleware(handlers.UpdateMetricsHandler(stor)))))))
		})

		r.Route("/value", func(r chi.Router) {
			r.Post("/", logger.RequestLogger(ipfilter.Middleware(encrypt.Middleware(compress.GzipMiddleware(
				hasher.HashMiddleware(handlers.GetMetricJSONHandler(stor)))))))
			r.Get("/{metricType}/{metricName}", logger.RequestLogger(compress.GzipMiddleware(handlers.GetMetricHandler(stor))))
		})
	})

	// Определяем маршрут по умолчанию для некорректных запросов
	r.NotFound(logger.RequestLogger(compress.GzipMiddleware(hasher.HashMiddleware(handlers.OtherRequestHandler()))))

	return r
}

// FlushMetricsToFile - сохраняет метрики в файл.
func FlushMetricsToFile(stor repositories.MetricsReader, saverVar saver.FileWriter) {
	logger.ServerLog.Debug("starting flush metrics to file")

	sleepInterval := saver.GetStoreInterval() * time.Second
	for {
		err := saverVar.WriteMetrics(stor)
		if err != nil {
			logger.ServerLog.Error("flushing metrics error", zap.String("error", error.Error(err)))
		}
		time.Sleep(sleepInterval)
	}
}
