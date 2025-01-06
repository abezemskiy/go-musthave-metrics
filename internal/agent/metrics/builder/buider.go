package builder

import (
	"fmt"
	"strconv"

	"go.uber.org/zap"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/logger"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/storage"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/repositories"
)

// BuildSlice - функция для создания слайса метрик.
func BuildSlice(metrics *storage.MetricsStats) []repositories.Metric {
	metricsSlice := make([]repositories.Metric, 0)
	if metrics == nil {
		return metricsSlice
	}

	// создаю слайс с метриками для отправки батчем
	for _, metricName := range storage.AllMetrics {
		typeMetric, value, err := metrics.GetMetricString(metricName)
		if err != nil {
			logger.AgentLog.Error(fmt.Sprintf("Failed to get metric %s: %v\n", typeMetric, err), zap.String("action", "push metrics"))
			continue
		}
		metric, err := Build(typeMetric, metricName, value)
		if err != nil {
			logger.AgentLog.Error(fmt.Sprintf("Failed to build metric structer %s: %v\n", typeMetric, err), zap.String("action", "push metrics"))
			continue
		}
		metricsSlice = append(metricsSlice, metric)
	}
	return metricsSlice
}

// Build - строит структуру метрики из принятых параметров типа string.
func Build(typeMetric, nameMetric, valueMetric string) (metric repositories.Metric, err error) {
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
