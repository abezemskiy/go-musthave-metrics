package serverhandlers

import (
	"html/template"
	"log"
	"net/http"
	"strconv"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/repositories"
	"github.com/go-chi/chi/v5"
)

func HandlerOther(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "text/plain")
	res.WriteHeader(http.StatusNotFound)
}

func GetGlobal(res http.ResponseWriter, req *http.Request, storage repositories.Repositories) {
	res.Header().Set("Content-Type", "text/html")
	res.WriteHeader(http.StatusOK)

	metrics := storage.GetAllMetrics()

	tmpl := template.Must(template.New("example").Parse(`
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
	err := tmpl.Execute(res, metrics)
	log.Print(err)
}

func GetMetric(res http.ResponseWriter, req *http.Request, storage repositories.Repositories) {
	res.Header().Set("Content-Type", "text/plan")
	metricType := chi.URLParam(req, "metricType")
	metricName := chi.URLParam(req, "metricName")

	value, err := storage.GetMetric(metricType, metricName)
	if err != nil {
		res.WriteHeader(http.StatusNotFound)
	}
	_, er := res.Write([]byte(value))
	log.Print(er)
}

// Благодаря использованию роутера chi в этот хэндлер будут попадать только запросы POST
func HandlerUpdate(res http.ResponseWriter, req *http.Request, storage repositories.Repositories) {

	// Проверка на nil для storage
	if storage == nil {
		http.Error(res, "Storage not initialized", http.StatusInternalServerError)
		return
	}
	res.Header().Set("Content-Type", "text/plain")

	metricType := chi.URLParam(req, "metricType")
	metricName := chi.URLParam(req, "metricName")
	metricValue := chi.URLParam(req, "metricValue")

	//fmt.Printf("URL is: metricType: %s, metricName: %s, metricValue: %s\n", metricType, metricName, metricValue)
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
