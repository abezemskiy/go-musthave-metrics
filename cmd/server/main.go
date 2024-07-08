package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/repositories"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/serverhandlers"
)

func MetricRouter(stor repositories.Repositories) chi.Router {

	r := chi.NewRouter()

	//fmt.Println("start server")

	r.Route("/", func(r chi.Router) {
		r.Get("/", func(res http.ResponseWriter, req *http.Request) {
			serverhandlers.GetGlobal(res, req, stor)
		})

		r.Post("/update/{metricType}/{metricName}/{metricValue}", func(res http.ResponseWriter, req *http.Request) {
			serverhandlers.HandlerUpdate(res, req, stor)
		})
		r.Route("/value", func(r chi.Router) {
			r.Get("/{metricType}/{metricName}", func(res http.ResponseWriter, req *http.Request) {
				serverhandlers.GetMetric(res, req, stor)
			})
		})
	})

	// Определяем маршрут по умолчанию для некорректных запросов
	r.NotFound(func(res http.ResponseWriter, req *http.Request) {
		serverhandlers.HandlerOther(res, req)
	})

	return r
}

func main() {
	storage := repositories.NewDefaultMemStorage()

	log.Fatal(http.ListenAndServe(":8080", MetricRouter(storage)))
}
