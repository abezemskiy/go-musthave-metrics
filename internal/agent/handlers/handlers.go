package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/compress"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/logger"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/storage"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/repositories"
	"github.com/go-resty/resty/v2"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
)

var (
	pollInterval   time.Duration = 2
	reportInterval time.Duration = 10
	contextTimeout               = 500 * time.Millisecond
)

func SetPollInterval(interval time.Duration) {
	pollInterval = interval
}

func GetPollInterval() time.Duration {
	return pollInterval
}

func SetReportInterval(interval time.Duration) {
	reportInterval = interval
}

func GetReportInterval() time.Duration {
	return reportInterval
}

// CollectMetrics собирает метрики
func SyncCollectMetrics(metrics *storage.MetricsStats) {
	metrics.Lock()
	defer metrics.Unlock()
	metrics.CollectMetrics()
}

// CollectMetricsTimer запускает сбор метрик с интервалом
func CollectMetricsTimer(metrics *storage.MetricsStats) {
	sleepInterval := GetPollInterval() * time.Second
	for {
		SyncCollectMetrics(metrics)
		time.Sleep(sleepInterval)
	}
}

// Строю структуру метрики из принятых параметров
func BuildMetric(typeMetric, nameMetric, valueMetric string) (metric repositories.Metric, err error) {
	metric.ID = nameMetric
	metric.MType = typeMetric

	switch typeMetric {
	case "counter":
		val, errParse := strconv.ParseInt(valueMetric, 10, 64)
		if errParse != nil {
			err = errParse
			return
		}
		metric.Delta = &val
	case "gauge":
		val, errParse := strconv.ParseFloat(valueMetric, 64)
		if errParse != nil {
			err = errParse
			return
		}
		metric.Value = &val
	default:
		err = fmt.Errorf("get invalid type of metric: %s", typeMetric)
		return
	}
	logger.AgentLog.Debug(fmt.Sprintf("Success build metric structure for JSON: name: %s, type: %s, delta: %d, value: %d", metric.ID, metric.MType, metric.Delta, metric.Value))
	return
}

// Push отправляет метрику на сервер в JSON формате и возвращает ошибку при неудаче
func PushJSON(address, action, typeMetric, nameMetric, valueMetric string, client *resty.Client) error {
	metric, err := BuildMetric(typeMetric, nameMetric, valueMetric)
	if err != nil {
		logger.AgentLog.Error("Build metric error", zap.String("error", error.Error(err)))
		return err
	}

	// сериализую полученную струтктуру с метриками в json-представление  в виде слайса байт
	var bufEncode bytes.Buffer
	enc := json.NewEncoder(&bufEncode)
	if err := enc.Encode(metric); err != nil {
		logger.AgentLog.Error("Encode message error", zap.String("error", error.Error(err)))
		return err
	}

	// Сжатие данных для передачи
	compressBody, err := compress.Compress(bufEncode.Bytes())
	if err != nil {
		logger.AgentLog.Error("Fail to comperess push data ", zap.String("error", error.Error(err)))
		return err
	}

	url := fmt.Sprintf("%s/%s", address, action)
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Content-Encoding", "gzip").
		SetHeader("Accept-Encoding", "gzip").
		SetBody(compressBody).
		Post(url)

	if err != nil {
		logger.AgentLog.Error("Push json metric to server error ", zap.String("error", error.Error(err)))
		return err
	}

	logger.AgentLog.Debug("Get answer from server", zap.String("Content-Encoding", resp.Header().Get("Content-Encoding")),
		zap.String("statusCode", fmt.Sprintf("%d", resp.StatusCode())),
		zap.String("Content-Type", resp.Header().Get("Content-Type")),
		zap.String("Content-Encoding", fmt.Sprint(resp.Header().Values("Content-Encoding"))))

	if resp.StatusCode() != http.StatusOK {
		logger.AgentLog.Error("Geting status is not 200 ", zap.String("statusCode", fmt.Sprintf("%d", resp.StatusCode())))
		return fmt.Errorf("status code is: %d %w", resp.StatusCode(), errors.New(resp.String()))
	}

	contentEncoding := resp.Header().Get("Content-Encoding")
	if strings.Contains(contentEncoding, "gzip") {
		logger.AgentLog.Debug("Get compress answer data in PushJSON function", zap.String("Content-Encoding", contentEncoding))
	} else {
		logger.AgentLog.Debug("Get uncompress answer data in PushJSON function", zap.String("Content-Encoding", contentEncoding))
	}

	responceMetric := resp.Body()
	if !bytes.Equal(bufEncode.Bytes(), responceMetric) {
		return fmt.Errorf("answer metric from server not equal pushing metric: get %d, want %d", responceMetric, bufEncode.Bytes())
	}

	// Десериализую данные полученные от сервера, в основном для дебага
	var resJSON repositories.Metric
	buRes := bytes.NewBuffer(responceMetric)
	dec := json.NewDecoder(buRes)
	if err := dec.Decode(&resJSON); err != nil {
		logger.AgentLog.Error("decode decompress data from server error ", zap.String("error", error.Error(err)))
		return err
	}
	logger.AgentLog.Debug(fmt.Sprintf("decode metric from server %s", resJSON.String()))

	logger.AgentLog.Debug(fmt.Sprintf("Success push metric in JSON format: typeMetric - %s, nameMetric - %s, valueMetric - %s", typeMetric, nameMetric, valueMetric))
	return nil
}

// Push отправляет метрику на сервер и возвращает ошибку при неудаче
func Push(address, action, typemetric, namemetric, valuemetric string, client *resty.Client) error {
	url := fmt.Sprintf("%s/%s/%s/%s/%s", address, action, typemetric, namemetric, valuemetric)
	resp, err := client.R().
		SetHeader("Content-Type", "text/plain").
		Post(url)

	if err != nil {
		return fmt.Errorf("error with post: %s, %w", url, err)
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("received non-200 response status: %d for url: %s", resp.StatusCode(), url)
	}
	return nil
}

// Отправляет все накопленные метрики на сервер, поочередно отправляя каждую метрику по отдельности
func PushMetrics(address, action string, metrics *storage.MetricsStats, client *resty.Client) {
	metrics.Lock()
	defer metrics.Unlock()

	for _, metricName := range storage.AllMetrics {
		typeMetric, value, err := metrics.GetMetricString(metricName)
		if err != nil {
			logger.AgentLog.Error(fmt.Sprintf("Failed to get metric %s: %v\n", typeMetric, err), zap.String("action", "push metrics"))
			continue
		}
		er := PushJSON(address, action, typeMetric, metricName, value, client)
		if er != nil {
			logger.AgentLog.Error(fmt.Sprintf("Failed to push metric %s: %v\n", typeMetric, er), zap.String("action", "push metrics"))
		}
	}
}

// Отправляет батч метрик на сервер
func PushBatch(address, action string, metricsSlice []repositories.Metric, client *resty.Client) error {

	// сериализую полученную слайс с метриками в json-представление  в виде слайса байт
	var bufEncode bytes.Buffer
	enc := json.NewEncoder(&bufEncode)
	if err := enc.Encode(metricsSlice); err != nil {
		logger.AgentLog.Error("Encode message error", zap.String("error", error.Error(err)))
		return err
	}

	// Сжатие данных для передачи
	compressBody, err := compress.Compress(bufEncode.Bytes())
	if err != nil {
		logger.AgentLog.Error("Fail to comperess push data ", zap.String("error", error.Error(err)))
		return err
	}

	// Создаю контекст с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), contextTimeout)
	defer cancel()

	url := fmt.Sprintf("%s/%s", address, action)
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Content-Encoding", "gzip").
		SetHeader("Accept-Encoding", "gzip").
		SetBody(compressBody).
		SetContext(ctx).
		Post(url)

	if err != nil {
		logger.AgentLog.Error("Push batch json metrics to server error ", zap.String("error", error.Error(err)))
		return err
	}

	logger.AgentLog.Debug("Get answer from server", zap.String("Content-Encoding", resp.Header().Get("Content-Encoding")),
		zap.String("statusCode", fmt.Sprintf("%d", resp.StatusCode())),
		zap.String("Content-Type", resp.Header().Get("Content-Type")),
		zap.String("Content-Encoding", fmt.Sprint(resp.Header().Values("Content-Encoding"))))

	if resp.StatusCode() != http.StatusOK {
		logger.AgentLog.Error("Geting status is not 200 ", zap.String("statusCode", fmt.Sprintf("%d", resp.StatusCode())))
		return fmt.Errorf("status code is: %d %w", resp.StatusCode(), errors.New(resp.String()))
	}
	contentEncoding := resp.Header().Get("Content-Encoding")
	if strings.Contains(contentEncoding, "gzip") {
		logger.AgentLog.Debug("Get compress answer data in PushBatch function", zap.String("Content-Encoding", contentEncoding))
	} else {
		logger.AgentLog.Debug("Get uncompress answer data in PushBatch function", zap.String("Content-Encoding", contentEncoding))
	}

	responceMetrics := resp.Body()
	if !bytes.Equal(bufEncode.Bytes(), responceMetrics) {
		return fmt.Errorf("answer metric from server not equal pushing metric: get %d, want %d", responceMetrics, bufEncode.Bytes())
	}

	// Десериализую данные полученные от сервера, в основном для дебага
	var resJSON []repositories.Metric
	buRes := bytes.NewBuffer(responceMetrics)
	dec := json.NewDecoder(buRes)
	if err := dec.Decode(&resJSON); err != nil {
		logger.AgentLog.Error("decode decompress data from server error ", zap.String("error", error.Error(err)))
		return err
	}

	logger.AgentLog.Debug("Success push batch metrics in JSON format")
	return nil
}

// Строит батч метрик и отправляет полученный батч на сервер в рамках одной передачи
func PushMetricsBatch(address, action string, metrics *storage.MetricsStats, client *resty.Client) error {
	metrics.Lock()
	defer metrics.Unlock()
	metricsSlice := make([]repositories.Metric, 0)

	// создаю слайс с метриками для отправки батчем
	for _, metricName := range storage.AllMetrics {
		typeMetric, value, err := metrics.GetMetricString(metricName)
		if err != nil {
			logger.AgentLog.Error(fmt.Sprintf("Failed to get metric %s: %v\n", typeMetric, err), zap.String("action", "push metrics"))
			continue
		}
		metric, err := BuildMetric(typeMetric, metricName, value)
		if err != nil {
			logger.AgentLog.Error(fmt.Sprintf("Failed to build metric structer %s: %v\n", typeMetric, err), zap.String("action", "push metrics"))
			continue
		}
		metricsSlice = append(metricsSlice, metric)
	}
	err := PushBatch(address, action, metricsSlice, client)
	if err != nil {
		logger.AgentLog.Error("Failed to push batch metrics", zap.String("action", "push metrics"), zap.String("error", error.Error(err)))
		return err
	}
	return nil
}

// Проверка того, что ошибка это "connect: connection refused"
func isConnectionRefused(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, syscall.ECONNREFUSED) || strings.Contains(err.Error(), "dial tcp: connect: connection refused")
}

func isDBTransportError(err error) bool {
	if err == nil {
		return false
	}
	asPgError := false
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		asPgError = (pgerrcode.IsConnectionException(pgErr.Code) ||
			pgErr.Code == pgerrcode.ConnectionDoesNotExist ||
			pgErr.Code == pgerrcode.ConnectionFailure ||
			pgErr.Code == pgerrcode.SQLClientUnableToEstablishSQLConnection)
	}
	asString := false
	asString = strings.Contains(err.Error(), "connection exception") ||
		strings.Contains(err.Error(), "connection does not exist") ||
		strings.Contains(err.Error(), "connection failure") ||
		strings.Contains(err.Error(), "SQL client unable to establish SQL connection")
	return asPgError || asString
}

func isFileLockedError(err error) bool {
	if err == nil {
		return false
	}
	asError := false
	asError = errors.Is(err, syscall.EACCES) ||
		errors.Is(err, syscall.EROFS) ||
		errors.Is(err, os.ErrPermission)

	asString := false
	asString = strings.Contains(err.Error(), "permission denied") ||
		strings.Contains(err.Error(), "read-only file system")
	return asError || asString
}

type PushFunction = func(string, string, *storage.MetricsStats, *resty.Client) error

// Для повторной отправки запроса в случае, если сервер не отвечает. Установлено три дополнительных попыток
func RetryExecPushFunction(address, action string, metrics *storage.MetricsStats, client *resty.Client, pushFunction PushFunction) {
	sleepIntervals := []time.Duration{0, 1, 3, 5}

	for i := 0; i < 4; i++ {
		logger.AgentLog.Debug(fmt.Sprintf("Push metrics to server, attemption %d", i+1))

		time.Sleep(sleepIntervals[i] * time.Second)

		err := pushFunction(address, action, metrics, client)
		if err != nil && (errors.Is(err, context.DeadlineExceeded) ||
			isConnectionRefused(err) ||
			isDBTransportError(err)) ||
			isFileLockedError(err) {
			continue
		} else {
			return
		}
	}
}

// PushMetricsTimer запускает отправку метрик с интервалом
func PushMetricsTimer(address, action string, metrics *storage.MetricsStats) {
	sleepInterval := GetReportInterval() * time.Second
	for {
		client := resty.New()
		RetryExecPushFunction(address, action, metrics, client, PushMetricsBatch)
		logger.AgentLog.Debug("Running agent", zap.String("action", "push metrics"))
		time.Sleep(sleepInterval)
	}
}
