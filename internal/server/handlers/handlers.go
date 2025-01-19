// Packet handlers contain endpoints of interaction with service.
package handlers

import (
	"database/sql"
	"encoding/json"
	"html/template"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/repositories"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/logger"
)

var (
	tmpl *template.Template
)

// Парсин шаблона для вывода всех метрик в виде html страницы.
func init() {
	tmpl = template.Must(template.New("example").Parse(`
        <!DOCTYPE html>
        <html lang="en">
        <head>
            <meta charset="UTF-8">
            <title>HTML Response</title>
        </head>
        <body>
            <pre>{{.}}</pre>
        </body>
        </html>
    `))
}

// OtherRequest - обработка нераспознанных http запросов к сервису.
func OtherRequest(res http.ResponseWriter, _ *http.Request) {
	res.Header().Set("Content-Type", "text/plain")
	res.WriteHeader(http.StatusNotFound)
}

// GetGlobal - возвращаю все хранящие на сервере метрики в виде html страницы.
func GetGlobal(res http.ResponseWriter, req *http.Request, storage repositories.MetricsReader) {
	res.Header().Set("Content-Type", "text/html")

	// устанавливаю заголовок таким образом вместо WriteHeader(http.StatusOK), потому что
	// далее в методе Write в middleware необходимо установить заголовок Hash со значением хэша,
	// а после WriteHeader заголовки уже не устанавливаются
	res.Header().Set("Status-Code", "200")
	metrics, err := storage.GetAllMetrics(req.Context())

	if err != nil {
		logger.ServerLog.Error("get all metrics error in GetGlobal handler", zap.String("error", error.Error(err)))
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(res, metrics); err != nil {
		logger.ServerLog.Error("template execute error in GetGlobal handler", zap.String("error", error.Error(err)))
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
}

// PingDatabase - проверка связи с базой данных.
func PingDatabase(res http.ResponseWriter, req *http.Request, db *sql.DB) {
	if err := db.PingContext(req.Context()); err != nil {
		logger.ServerLog.Error("fail to ping database", zap.String("error", error.Error(err)))
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	res.WriteHeader(http.StatusOK)
}

// GetMetricJSON - возвращает метрику в json представлении.
func GetMetricJSON(res http.ResponseWriter, req *http.Request, storage repositories.MetricsReader) {
	logger.ServerLog.Debug("In GetMetricJSON", zap.String("address", req.URL.String()))

	res.Header().Set("Content-Type", "application/json")

	defer req.Body.Close()

	var metrics repositories.Metric
	if err := json.NewDecoder(req.Body).Decode(&metrics); err != nil {
		logger.ServerLog.Error("In GetMetricJSON decode body error", zap.String("address", req.URL.String()), zap.String("error", error.Error(err)))
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	metricType := metrics.MType
	metricName := metrics.ID

	value, err := storage.GetMetric(req.Context(), metricType, metricName)
	if err != nil {
		http.Error(res, err.Error(), http.StatusNotFound)
		return
	}

	switch metricType {
	case "counter":
		val, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			logger.ServerLog.Error("Convert string to int64 error: ", zap.String("address", req.URL.String()), zap.String("error: ", error.Error(err)))
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		metrics.Delta = &val
	case "gauge":
		val, err := strconv.ParseFloat(value, 64)
		if err != nil {
			logger.ServerLog.Error("Convert string to float64 error: ", zap.String("address", req.URL.String()), zap.String("error: ", error.Error(err)))
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		metrics.Value = &val
	default:
		logger.ServerLog.Debug("In GetMetricJSON invalid type of metric", zap.String("address", req.URL.String()))
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	// устанавливаю заголовок таким образом вместо WriteHeader(http.StatusOK), потому что
	// далее в методе Write в middleware необходимо установить заголовок Hash со значением хэша,
	// а после WriteHeader заголовки уже не устанавливаются
	res.Header().Set("Status-Code", "200")
	enc := json.NewEncoder(res)
	if err := enc.Encode(metrics); err != nil {
		logger.ServerLog.Error("error encoding response", zap.String("error", error.Error(err)))
		return
	}
}

// GetMetric - возвращает метрику в виде строки.
func GetMetric(res http.ResponseWriter, req *http.Request, storage repositories.MetricsReader) {
	logger.ServerLog.Debug("in GetMetric handler", zap.String("address", req.URL.String()))

	res.Header().Set("Content-Type", "text/plan")
	metricType := chi.URLParam(req, "metricType")
	metricName := chi.URLParam(req, "metricName")

	value, err := storage.GetMetric(req.Context(), metricType, metricName)
	if err != nil {
		logger.ServerLog.Error("get metric error", zap.String("address", req.URL.String()), zap.String("error", error.Error(err)))
		http.Error(res, err.Error(), http.StatusNotFound)
		return
	}
	// устанавливаю заголовок таким образом вместо WriteHeader(http.StatusOK), потому что
	// далее в методе Write в middleware необходимо установить заголовок Hash со значением хэша,
	// а после WriteHeader заголовки уже не устанавливаются
	res.Header().Set("Status-Code", "200")

	n, err := res.Write([]byte(value))
	if err != nil {
		log.Printf("Write error in GetMetric handler: %v\n", err)
		return
	}
	if n < len(value) {
		log.Printf("Not all bytes were written in GetMetric handler. Written: %d, Total: %d", n, len(value))
	}
}

// UpdateMetricsBatch - обновляет метрики через json батч, который является слайсом метрик.
func UpdateMetricsBatch(res http.ResponseWriter, req *http.Request, storage repositories.MetricsWriter) {
	// Проверка на nil для storage
	if storage == nil {
		http.Error(res, "Storage not initialized", http.StatusInternalServerError)
		return
	}

	res.Header().Set("Content-Type", "application/json")

	metrics := make([]repositories.Metric, 0)

	if err := json.NewDecoder(req.Body).Decode(&metrics); err != nil {
		logger.ServerLog.Error("Decode message error", zap.String("address", req.URL.String()))
		http.Error(res, "Decode message error", http.StatusInternalServerError)
		return
	}

	err := storage.AddMetricsFromSlice(req.Context(), metrics)
	if err != nil {
		logger.ServerLog.Error("add metric into server error", zap.String("address", req.URL.String()), zap.String("error", error.Error(err)))
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	logger.ServerLog.Debug("Successful decode metrcic from json", zap.String("address: ", req.URL.String()))

	// устанавливаю заголовок таким образом вместо WriteHeader(http.StatusOK), потому что
	// далее в методе Write в middleware необходимо установить заголовок Hash со значением хэша,
	// а после WriteHeader заголовки уже не устанавливаются
	res.Header().Set("Status-Code", "200")

	enc := json.NewEncoder(res)
	if err := enc.Encode(metrics); err != nil {
		logger.ServerLog.Error("error encoding response", zap.String("error", error.Error(err)))
		return
	}

	logger.ServerLog.Debug("successful write encode data, server answer is", zap.String("Content-Encoding", res.Header().Get("Content-Encoding")),
		zap.String("Status-Code", res.Header().Get("Status-Code")),
		zap.String("Content-Type", res.Header().Get("Content-Type")),
		zap.String("HashSHA256", res.Header().Get("HashSHA256")))
}

// UpdateMetricsJSON - для обновления метрик через json.
// Благодаря использованию роутера chi в этот хэндлер будут попадать только запросы POST.
func UpdateMetricsJSON(res http.ResponseWriter, req *http.Request, storage repositories.MetricsWriter) {
	// Проверка на nil для storage
	if storage == nil {
		http.Error(res, "Storage not initialized", http.StatusInternalServerError)
		return
	}

	res.Header().Set("Content-Type", "application/json")

	var metrics = repositories.Metric{}

	if err := json.NewDecoder(req.Body).Decode(&metrics); err != nil {
		logger.ServerLog.Error("Decode message error", zap.String("address", req.URL.String()))
		http.Error(res, "Decode message error", http.StatusInternalServerError)
		return
	}

	switch metrics.MType {
	case "gauge":
		if metrics.Value == nil {
			logger.ServerLog.Error("Decode message error, value in gauge metric is nil", zap.String("address", req.URL.String()))
			res.WriteHeader(http.StatusBadRequest)
			return
		}
		err := storage.AddGauge(req.Context(), metrics.ID, *metrics.Value)
		if err != nil {
			logger.ServerLog.Error("add gauge error", zap.String("address", req.URL.String()), zap.String("error", error.Error(err)))
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
	case "counter":
		if metrics.Delta == nil {
			logger.ServerLog.Error("Decode message error, delta in counter metric is nil", zap.String("address", req.URL.String()))
			res.WriteHeader(http.StatusBadRequest)
			return
		}
		err := storage.AddCounter(req.Context(), metrics.ID, *metrics.Delta)
		if err != nil {
			logger.ServerLog.Error("add counter error", zap.String("address", req.URL.String()), zap.String("error", error.Error(err)))
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
	default:
		logger.ServerLog.Error("Invalid type of metric", zap.String("type", metrics.MType)) //---------------------------------------------
		res.WriteHeader(http.StatusBadRequest)
		return
	}
	logger.ServerLog.Debug("Successful decode metrcic from json", zap.String("address: ", req.URL.String()))

	bodyDebug, _ := io.ReadAll(req.Body)
	logger.ServerLog.Debug("message before compress", zap.String("bytes: ", string(bodyDebug)))

	// устанавливаю заголовок таким образом вместо WriteHeader(http.StatusOK), потому что
	// далее в методе Write в middleware необходимо установить заголовок Hash со значением хэша,
	// а после WriteHeader заголовки уже не устанавливаются
	res.Header().Set("Status-Code", "200")

	enc := json.NewEncoder(res)
	if err := enc.Encode(metrics); err != nil {
		logger.ServerLog.Error("error encoding response", zap.String("error", error.Error(err)))
		return
	}

	logger.ServerLog.Debug("successful write encode data to answer message")

	logger.ServerLog.Debug("server answer is", zap.String("Content-Encoding", res.Header().Get("Content-Encoding")),
		zap.String("Status-Code", res.Header().Get("Status-Code")),
		zap.String("HashSHA256", res.Header().Get("HashSHA256")),
		zap.String("Content-Type", res.Header().Get("Content-Type")))
}

// UpdateMetrics - обновляет метрику на сервере. Параметры метрики извлекаются из http запроса.
func UpdateMetrics(res http.ResponseWriter, req *http.Request, storage repositories.MetricsWriter) {

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
		err = storage.AddGauge(req.Context(), metricName, value)
		if err != nil {
			logger.ServerLog.Error("add gauge error", zap.String("address", req.URL.String()), zap.String("error", error.Error(err)))
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
	case "counter":
		value, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			res.WriteHeader(http.StatusBadRequest)
			return
		}
		err = storage.AddCounter(req.Context(), metricName, value)
		if err != nil {
			logger.ServerLog.Error("add counter error", zap.String("address", req.URL.String()), zap.String("error", error.Error(err)))
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
	default:
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	res.WriteHeader(http.StatusOK)
}

// GetGlobalHandler - обертка над GetGlobal для возможности установить хранилище метрик.
func GetGlobalHandler(stor repositories.MetricsReader) http.HandlerFunc {
	fn := func(res http.ResponseWriter, req *http.Request) {
		GetGlobal(res, req, stor)
	}
	return fn
}

// PingDatabaseHandler - обертка над PingDatabase для возможности установить базу данных.
func PingDatabaseHandler(db *sql.DB) http.HandlerFunc {
	fn := func(res http.ResponseWriter, req *http.Request) {
		PingDatabase(res, req, db)
	}
	return fn
}

// UpdateMetricsBatchHandler - обертка над UpdateMetricsBatch для возможности установить хранилище метрик.
func UpdateMetricsBatchHandler(stor repositories.MetricsWriter) http.HandlerFunc {
	fn := func(res http.ResponseWriter, req *http.Request) {
		UpdateMetricsBatch(res, req, stor)
	}
	return fn
}

// UpdateMetricsJSONHandler - обертка над UpdateMetricsJSON для возможности установить хранилище метрик.
func UpdateMetricsJSONHandler(stor repositories.MetricsWriter) http.HandlerFunc {
	fn := func(res http.ResponseWriter, req *http.Request) {
		UpdateMetricsJSON(res, req, stor)
	}
	return fn
}

// UpdateMetricsHandler - обертка над UpdateMetrics для возможности установить хранилище метрик.
func UpdateMetricsHandler(stor repositories.MetricsWriter) http.HandlerFunc {
	fn := func(res http.ResponseWriter, req *http.Request) {
		UpdateMetrics(res, req, stor)
	}
	return fn
}

// GetMetricJSONHandler - обертка над GetMetricJSON для возможности установить хранилище метрик.
func GetMetricJSONHandler(stor repositories.MetricsReader) http.HandlerFunc {
	fn := func(res http.ResponseWriter, req *http.Request) {
		GetMetricJSON(res, req, stor)
	}
	return fn
}

// GetMetricHandler - обертка над GetMetric для возможности установить хранилище метрик.
func GetMetricHandler(stor repositories.MetricsReader) http.HandlerFunc {
	fn := func(res http.ResponseWriter, req *http.Request) {
		GetMetric(res, req, stor)
	}
	return fn
}

// OtherRequestHandler - обертка над OtherRequest для возможности установить хранилище метрик.
func OtherRequestHandler() http.HandlerFunc {
	fn := func(res http.ResponseWriter, req *http.Request) {
		OtherRequest(res, req)
	}
	return fn
}
