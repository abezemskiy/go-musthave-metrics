package handlers

import (
	"html/template"
	"log"
	"net/http"
	"strconv"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/repositories"
	"github.com/go-chi/chi/v5"
)

var (
	tmpl *template.Template
)

func init() {
	tmpl = template.Must(template.New("example").Parse(`
        <!DOCTYPE html>
        <html lang="en">
        <head>
            <meta charset="UTF-8">
            <title>HTML Response</title>
        </head>
        <body>
            <h1>{{.}}</h1>
        </body>
        </html>
    `))
}

func OtherRequest(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "text/plain")
	res.WriteHeader(http.StatusNotFound)
}

func GetGlobal(res http.ResponseWriter, req *http.Request, storage repositories.ServerRepo) {
	res.Header().Set("Content-Type", "text/html")
	res.WriteHeader(http.StatusOK)

	metrics := storage.GetAllMetrics()

	err := tmpl.Execute(res, metrics)
	if err != nil {
		log.Printf("Template execute error in GetGlobal handler: %v\n", err)
	}
}

func GetMetric(res http.ResponseWriter, req *http.Request, storage repositories.ServerRepo) {
	res.Header().Set("Content-Type", "text/plan")
	metricType := chi.URLParam(req, "metricType")
	metricName := chi.URLParam(req, "metricName")

	value, err := storage.GetMetric(metricType, metricName)
	if err != nil {
		res.WriteHeader(http.StatusNotFound)
		return
	}

	n, err := res.Write([]byte(value))
	if err != nil {
		log.Printf("Write error in GetMetric handler: %v\n", err)
		return
	}
	if n < len(value) {
		log.Printf("Not all bytes were written in GetMetric handler. Written: %d, Total: %d", n, len(value))
	}
}

// Благодаря использованию роутера chi в этот хэндлер будут попадать только запросы POST
func UpdateMetrics(res http.ResponseWriter, req *http.Request, storage repositories.ServerRepo) {

	// Проверка на nil для storage
	if storage == nil {
		http.Error(res, "Storage not initialized", http.StatusInternalServerError)
		return
	}
	res.Header().Set("Content-Type", "text/plain")

	metricType := chi.URLParam(req, "metricType")
	metricName := chi.URLParam(req, "metricName")
	metricValue := chi.URLParam(req, "metricValue")

	if metricName == "" {
		res.WriteHeader(http.StatusNotFound)
		return
	}

	switch metricType {
	case "gauge":
		value, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			res.WriteHeader(http.StatusBadRequest)
			return
		}
		storage.AddGauge(metricName, value)
	case "counter":
		value, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			res.WriteHeader(http.StatusBadRequest)
			return
		}
		storage.AddCounter(metricName, value)
	default:
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	res.WriteHeader(http.StatusOK)
}

func GetGlobalHandler(stor repositories.ServerRepo) http.Handler {
	fn := func(res http.ResponseWriter, req *http.Request) {
		GetGlobal(res, req, stor)
	}
	return http.HandlerFunc(fn)
}

func UpdateMetricsHandler(stor repositories.ServerRepo) http.Handler {
	fn := func(res http.ResponseWriter, req *http.Request) {
		UpdateMetrics(res, req, stor)
	}
	return http.HandlerFunc(fn)
}

func GetMetricHandler(stor repositories.ServerRepo) http.Handler {
	fn := func(res http.ResponseWriter, req *http.Request) {
		GetMetric(res, req, stor)
	}
	return http.HandlerFunc(fn)
}

func OtherRequestHandler() http.Handler{
	fn := func(res http.ResponseWriter, req *http.Request) {
		OtherRequest(res, req)
	}
	return http.HandlerFunc(fn)
}
