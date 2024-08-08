package repositories

import (
	"context"
	"fmt"
)

type (
	Repositories interface {
		GetMetric(context.Context, string, string) (string, error)
	}

	ServerRepo interface {
		Repositories
		AddGauge(context.Context, string, float64) error
		AddCounter(context.Context, string, int64) error
		GetAllMetrics(context.Context) (string, error)
		AddMetricsFromSlice(context.Context, []Metrics) error
		//GetCounters() map[string]int64
		// GetGauges() map[string]float64
		GetAllMetricsSlice(context.Context) ([]Metrics, error)
		Bootstrap(context.Context) error
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
