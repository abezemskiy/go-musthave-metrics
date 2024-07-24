package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/logger"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/storage"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/repositories"
	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

var (
	pollInterval   time.Duration = 2
	reportInterval time.Duration = 10
)

func SetPollInterval(interval time.Duration) {
	pollInterval = interval
}

func SetReportInterval(interval time.Duration) {
	reportInterval = interval
}

// CollectMetrics собирает метрики
func SyncCollectMetrics(metrics *storage.MetricsStats) {
	metrics.Lock()
	defer metrics.Unlock()
	metrics.CollectMetrics()
}

// CollectMetricsTimer запускает сбор метрик с интервалом
func CollectMetricsTimer(metrics *storage.MetricsStats) {
	for {
		SyncCollectMetrics(metrics)
		time.Sleep(pollInterval * time.Second)
	}
}

// Push отправляет метрику на сервер в JSON формате и возвращает ошибку при неудаче
func PushJSON(address, action, typeMetric, nameMetric, valueMetric string, client *resty.Client) error {
	// Строю структуру метрики для сериализации из принятых параметров
	var metrics repositories.Metrics
	metrics.ID = nameMetric
	metrics.MType = typeMetric

	switch typeMetric {
	case "counter":
		val, err := strconv.ParseInt(valueMetric, 10, 64)
		if err != nil {
			logger.AgentLog.Error("Convert string to int64 error: ", zap.String("error: ", error.Error(err)))
			return err
		}
		metrics.Delta = &val
	case "gauge":
		val, err := strconv.ParseFloat(valueMetric, 64)
		if err != nil {
			logger.AgentLog.Error("Convert string to float64 error: ", zap.String("error: ", error.Error(err)))
			return err
		}
		metrics.Value = &val
	default:
		logger.AgentLog.Error("Invalid type of metric", zap.String("type: ", metrics.MType)) //---------------------------------------------
		return fmt.Errorf("get invalid type of metric: %s", typeMetric)
	}
	logger.AgentLog.Debug(fmt.Sprintf("Success build metric structure for JSON: name: %s, type: %s, delta: %d, value: %d", metrics.ID, metrics.MType, metrics.Delta, metrics.Value))

	// сериализую полученную струтктуру с метриками в json-представление  в виде слайса байт
	body, err := json.Marshal(metrics)
	if err != nil {
		logger.AgentLog.Error("Encode message error", zap.String("error: ", error.Error(err)))
		return err
	}

	url := fmt.Sprintf("%s/%s", address, action)
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(body).
		Post(url)

	if err != nil {
		logger.AgentLog.Error("Push json metric to server error ", zap.String("error: ", error.Error(err)))
		return err
	}

	if resp.StatusCode() != http.StatusOK {
		logger.AgentLog.Error("Geting status is not 200 ", zap.String("error: ", error.Error(err)))
		return err
	}

	respBody := resp.Body()
	if !bytes.Equal(body, respBody) {
		return fmt.Errorf("answer metric from server not equal pushing metric: get %d, want %d", body, respBody)
	}
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

// PushMetrics отправляет все метрики
func PushMetrics(address, action string, metrics *storage.MetricsStats, client *resty.Client) {
	metrics.Lock()
	defer metrics.Unlock()

	for _, metricName := range storage.AllMetrics {
		typeMetric, value, err := metrics.GetMetricString(metricName)
		if err != nil {
			logger.AgentLog.Error(fmt.Sprintf("Failed to push metric %s: %v\n", typeMetric, err), zap.String("action", "push metrics"))
		}
		er := Push(address, action, typeMetric, metricName, value, client)
		if er != nil {
			logger.AgentLog.Error(fmt.Sprintf("Failed to push metric %s: %v\n", typeMetric, err), zap.String("action", "push metrics"))
		}
	}
}

// PushMetricsTimer запускает отправку метрик с интервалом
func PushMetricsTimer(address, action string, metrics *storage.MetricsStats) {
	for {
		client := resty.New()
		PushMetrics(address, action, metrics, client)
		logger.AgentLog.Debug("Running agent", zap.String("action", "push metrics"))
		time.Sleep(reportInterval * time.Second)
	}
}
