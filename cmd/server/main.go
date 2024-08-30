package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/repositories"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/compress"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/handlers"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/hasher"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/logger"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/pg"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/saver"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/storage"
)

func main() {
	saveMode := parseFlags()

	// Подключение к базе данных
	db, err := sql.Open("pgx", flagDatabaseDsn)
	if err != nil {
		log.Fatalf("Error connection to database: %v by address %s", err, flagDatabaseDsn)
	}
	defer db.Close()

	// Создаю разные хранилища в зависимости от типа запуска сервера
	var stor repositories.ServerRepo
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
	var saverVar saver.WriterInterface
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
	log.Println("Stop server")
}

// функция run будет полезна при инициализации зависимостей сервера перед запуском
func run(stor repositories.ServerRepo, saverVar saver.WriterInterface, db *sql.DB, saveMode int) error {
	if err := logger.Initialize(flagLogLevel); err != nil {
		return err
	}

	var reader saver.ReadInterface
	var err error
	if saveMode == SAVEINFILE {
		reader, err = saver.NewReader(saver.GetFilestoragePath())
		if err != nil {
			logger.ServerLog.Fatal("create writer for saving metrics error", zap.String("error", error.Error(err)))
		}
	}

	// Работаю с файлом для сохранения метрик, только в случае соответствующей конфигурации запуска сервера
	if saveMode == SAVEINFILE {
		// Загружаю на сервер метрики из файла, сохраненные в предыдущих запусках
		err := saver.AddMetricsFromFile(stor, reader)
		if err != nil {
			logger.ServerLog.Fatal("add metrics from file error", zap.String("error", error.Error(err)))
		}
		go FlushMetricsToFile(stor, saverVar)
	}

	logger.ServerLog.Info("Running server", zap.String("address", flagNetAddr))
	return http.ListenAndServe(flagNetAddr, MetricRouter(stor, db))
}

func MetricRouter(stor repositories.ServerRepo, db *sql.DB) chi.Router {

	r := chi.NewRouter()

	r.Route("/", func(r chi.Router) {
		r.Get("/", logger.RequestLogger(compress.GzipMiddleware(handlers.GetGlobalHandler(stor))))
		r.Get("/ping", logger.RequestLogger(compress.GzipMiddleware(handlers.PingDatabaseHandler(db))))

		r.Post("/updates/", logger.RequestLogger(compress.GzipMiddleware(hasher.HashMiddleware(handlers.UpdateMetricsBatchHandler(stor)))))
		r.Route("/update", func(r chi.Router) {
			r.Post("/", logger.RequestLogger(compress.GzipMiddleware(hasher.HashMiddleware(handlers.UpdateMetricsJSONHandler(stor)))))
			r.Post("/{metricType}/{metricName}/{metricValue}", logger.RequestLogger(compress.GzipMiddleware(hasher.HashMiddleware(handlers.UpdateMetricsHandler(stor)))))
		})

		r.Route("/value", func(r chi.Router) {
			r.Post("/", logger.RequestLogger(compress.GzipMiddleware(hasher.HashMiddleware(handlers.GetMetricJSONHandler(stor)))))
			r.Get("/{metricType}/{metricName}", logger.RequestLogger(compress.GzipMiddleware(handlers.GetMetricHandler(stor))))
		})
	})

	// Определяем маршрут по умолчанию для некорректных запросов
	r.NotFound(logger.RequestLogger(compress.GzipMiddleware(hasher.HashMiddleware(handlers.OtherRequestHandler()))))

	return r
}

func FlushMetricsToFile(stor repositories.ServerRepo, saverVar saver.WriterInterface) {
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
