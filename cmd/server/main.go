package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

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

	// Горутина для запуска сервера
	go func() {
		logger.ServerLog.Info("Running server", zap.String("address", flagNetAddr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error starting server: %v", err)
		}
	}()

	// Блокирование до тех пор, пока не поступит сигнал о прерывании
	<-quit
	log.Println("Shutting down server...")

	// Create a context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), shutdownWaitPeriod)
	defer cancel()

	// останавливаю сервер, чтобы он перестал принимать новые запросы
	if err := srv.Shutdown(ctx); err != nil {
		logger.ServerLog.Error("Stopping server error: %v", zap.String("error", error.Error(err)))
		return err
	}
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
