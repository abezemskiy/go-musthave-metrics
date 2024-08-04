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
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/logger"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/saver"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/storage"
)

func main() {
	parseFlags()

	// Подключение к базе данных
	db, err := sql.Open("pgx", flagDatabaseDsn)
	if err != nil {
		log.Fatalf("Error connection to database: %v by address %s", err, flagDatabaseDsn)
	}
	defer db.Close()

	// Создаю родительский контекст
	ctx := context.Background()

	stor := storage.NewDefaultMemStorage()

	saver, err := saver.NewWriter(saver.GetFilestoragePath())
	if err != nil {
		log.Fatalf("Error create writer for saving metrics : %v\n", err)
	}

	if err := run(ctx, stor, saver, db); err != nil {
		log.Fatalf("Error starting server: %v\n", err)
	}
	// При штатном завершении работы сервера накопленные данные сохраняются
	if err := saver.WriteMetrics(stor); err != nil {
		logger.ServerLog.Error("flushing metrics error", zap.String("error", error.Error(err)))
	}
	log.Println("Stop server")
}

// функция run будет полезна при инициализации зависимостей сервера перед запуском
func run(ctx context.Context, stor repositories.ServerRepo, saverVar saver.WriterInterface, db *sql.DB) error {
	if err := logger.Initialize(flagLogLevel); err != nil {
		return err
	}
	reader, err := saver.NewReader(saver.GetFilestoragePath())
	if err != nil {
		log.Fatalf("Error create writer for saving metrics : %v\n", err)
	}
	// Загружаю на сервер метрики, сохраненные в предыдущих запусках
	saver.AddMetricsFromFile(stor, reader)
	go FlushMetricsToFile(stor, saverVar)

	logger.ServerLog.Info("Running server", zap.String("address", flagNetAddr))
	return http.ListenAndServe(flagNetAddr, MetricRouter(ctx, stor, db))
}

func MetricRouter(ctx context.Context, stor repositories.ServerRepo, db *sql.DB) chi.Router {

	r := chi.NewRouter()

	r.Route("/", func(r chi.Router) {
		r.Get("/", logger.RequestLogger(compress.GzipMiddleware(handlers.GetGlobalHandler(stor))))
		r.Get("/ping", logger.RequestLogger(compress.GzipMiddleware(handlers.PingDatabaseHandler(ctx, db))))

		r.Route("/update", func(r chi.Router) {
			r.Post("/", logger.RequestLogger(compress.GzipMiddleware(handlers.UpdateMetricsJSONHandler(stor))))
			r.Post("/{metricType}/{metricName}/{metricValue}", logger.RequestLogger(compress.GzipMiddleware(handlers.UpdateMetricsHandler(stor))))
		})

		r.Route("/value", func(r chi.Router) {
			r.Post("/", logger.RequestLogger(compress.GzipMiddleware(handlers.GetMetricJSONHandler(stor))))
			r.Get("/{metricType}/{metricName}", logger.RequestLogger(compress.GzipMiddleware(handlers.GetMetricHandler(stor))))
		})
	})

	// Определяем маршрут по умолчанию для некорректных запросов
	r.NotFound(logger.RequestLogger(compress.GzipMiddleware(handlers.OtherRequestHandler())))

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
