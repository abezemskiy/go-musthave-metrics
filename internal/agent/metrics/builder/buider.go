package builder

import (
	"fmt"
	"strconv"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/logger"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/repositories"
)

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
