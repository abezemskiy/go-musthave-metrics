package main

import (
	"log"
	"net/http"
	"time"

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

	stor := storage.NewDefaultMemStorage()

	saver, err := saver.NewWriter(saver.GetFilestoragePath())
	if err != nil {
		log.Fatalf("Error create writer for saving metrics : %v\n", err)
	}

	if err := run(stor, saver); err != nil {
		log.Fatalf("Error starting server: %v\n", err)
	}
	// При штатном завершении работы сервера накопленные данные сохраняются
	if err := saver.FlushMetrics(); err != nil {
		logger.ServerLog.Error("flushing metrics error", zap.String("error", error.Error(err)))
	}
	log.Println("Stop server")
}

// функция run будет полезна при инициализации зависимостей сервера перед запуском
func run(stor repositories.ServerRepo, saverVar saver.WriterInterface) error {
	if err := logger.Initialize(flagLogLevel); err != nil {
		return err
	}
	reader, err := saver.NewReader(saver.GetFilestoragePath())
	if err != nil {
		log.Fatalf("Error create writer for saving metrics : %v\n", err)
	}
	// Загружаю на сервер метрики, сохраненные в предыдущих запусках
	saver.AddMetricsFromFile(stor, reader)
	go FlashMetricsToFile(saverVar)

	logger.ServerLog.Info("Running server", zap.String("address", flagNetAddr))
	return http.ListenAndServe(flagNetAddr, MetricRouter(stor, saverVar))
}

func MetricRouter(stor repositories.ServerRepo, saver saver.WriterInterface) chi.Router {

	r := chi.NewRouter()

	r.Route("/", func(r chi.Router) {
		r.Get("/", logger.RequestLogger(compress.GzipMiddleware(handlers.GetGlobalHandler(stor))))

		r.Route("/update", func(r chi.Router) {
			r.Post("/", logger.RequestLogger(compress.GzipMiddleware(handlers.UpdateMetricsJSONHandler(stor, saver))))
			r.Post("/{metricType}/{metricName}/{metricValue}", logger.RequestLogger(compress.GzipMiddleware(handlers.UpdateMetricsHandler(stor, saver))))
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

func FlashMetricsToFile(saverVar saver.WriterInterface) {
	logger.ServerLog.Debug("starting flush metrics to file")

	time.Sleep(100 * time.Millisecond)
	for {
		err := saverVar.FlushMetrics()
		if err != nil {
			logger.ServerLog.Error("flushing metrics error", zap.String("error", error.Error(err)))
		}
		time.Sleep(saver.GetStoreInterval() * time.Second)
	}
}
