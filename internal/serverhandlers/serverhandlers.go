package serverhandlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/repositories"
)

func HandlerOther(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "text/plain")
	res.WriteHeader(http.StatusBadRequest)
}

func HandlerUpdate(res http.ResponseWriter, req *http.Request, storage repositories.Repositories) {
	// Проверка на nil для storage
	if storage == nil {
		http.Error(res, "Storage not initialized", http.StatusInternalServerError)
		return
	}

	if req.Method != http.MethodPost {
		res.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	res.Header().Set("Content-Type", "text/plain")

	args := strings.Split(req.URL.Path, "/")

	if len(args) < 5 {
		res.WriteHeader(http.StatusNotFound)
		return
	}

	metricType := args[2]
	metricName := args[3]
	metricValue := args[4]
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
