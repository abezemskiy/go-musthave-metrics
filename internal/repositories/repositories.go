package repositories

import "fmt"

type (
	Repositories interface {
		GetMetric(string, string) (string, error)
	}

	ServerRepo interface {
		Repositories
		AddGauge(string, float64)
		AddCounter(string, int64)
		GetAllMetrics() string
		AddMetricsFromSlice([]Metrics) error
	}

	// Структура для работы с метриками json формата

	Metrics struct {
		ID    string   `json:"id"`              // имя метрики
		MType string   `json:"type"`            // параметр, принимающий значение gauge или counter
		Delta *int64   `json:"delta,omitempty"` // значение метрики в случае передачи counter
		Value *float64 `json:"value,omitempty"` // значение метрики в случае передачи gauge
	}
)

func (metrcic Metrics) String() string {
	var delta = "nil"
	if metrcic.Delta != nil {
		delta = fmt.Sprintf("%d", *metrcic.Delta)
	}
	var value = "nil"
	if metrcic.Value != nil {
		value = fmt.Sprintf("%g", *metrcic.Value)
	}
	return fmt.Sprintf("ID: %s, MType: %s, Delta: %s, Value: %s", metrcic.ID, metrcic.MType, delta, value)
}
