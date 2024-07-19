package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/repositories"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/handlers"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/logger"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/storage"
)

func main() {
	parseFlags()

	stor := storage.NewDefaultMemStorage()

	err := run(stor)

	if err != nil {
		log.Printf("Error starting server: %v\n", err)
	}
}

// функция run будет полезна при инициализации зависимостей сервера перед запуском
func run(stor repositories.ServerRepo) error {
	if err := logger.Initialize(flagLogLevel); err != nil {
		return err
	}

	logger.ServerLog.Info("Running server", zap.String("address", flagNetAddr))
	return http.ListenAndServe(flagNetAddr, MetricRouter(stor))
}

func MetricRouter(stor repositories.ServerRepo) chi.Router {

	r := chi.NewRouter()

	r.Route("/", func(r chi.Router) {
		r.Get("/", logger.RequestLogger(handlers.GetGlobalHandler(stor)))

		r.Post("/update/{metricType}/{metricName}/{metricValue}", logger.RequestLogger(handlers.UpdateMetricsHandler(stor)))
		r.Route("/value", func(r chi.Router) {
			r.Get("/{metricType}/{metricName}", logger.RequestLogger(handlers.GetMetricHandler(stor)))
		})
	})

	// Определяем маршрут по умолчанию для некорректных запросов
	r.NotFound(logger.RequestLogger(handlers.OtherRequestHandler()))

	return r
}
