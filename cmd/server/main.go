package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/repositories"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/serverhandlers"
)

var addr = &NetAddress{
	Host: "localhost",
	Port: 8080,
}

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
	// если интерфейс не реализован,
	// здесь будет ошибка компиляции
	_ = flag.Value(addr)
	// проверка реализации
	flag.Var(addr, "a", "Net address host:port")
	flag.Parse()

	storage := repositories.NewDefaultMemStorage()

	log.Fatal(http.ListenAndServe(addr.String(), MetricRouter(storage)))
}
