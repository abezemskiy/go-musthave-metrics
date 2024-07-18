package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/repositories"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/handlers"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/storage"
)

func MetricRouter(stor repositories.Repositories) chi.Router {

	r := chi.NewRouter()

	r.Route("/", func(r chi.Router) {
		r.Get("/", func(res http.ResponseWriter, req *http.Request) {
			handlers.GetGlobal(res, req, stor)
		})

		r.Post("/update/{metricType}/{metricName}/{metricValue}", func(res http.ResponseWriter, req *http.Request) {
			handlers.HandlerUpdate(res, req, stor)
		})
		r.Route("/value", func(r chi.Router) {
			r.Get("/{metricType}/{metricName}", func(res http.ResponseWriter, req *http.Request) {
				handlers.GetMetric(res, req, stor)
			})
		})
	})

	// Определяем маршрут по умолчанию для некорректных запросов
	r.NotFound(func(res http.ResponseWriter, req *http.Request) {
		handlers.HandlerOther(res, req)
	})

	return r
}

func main() {
	parseFlags()

	storage := storage.NewDefaultMemStorage()

	err := http.ListenAndServe(flagNetAddr, MetricRouter(storage))

	if err != nil {
		log.Printf("Error starting server: %v\n", err)
	}
}
