package repositories

import (
	"context"
	"fmt"
)

// Интерфесы хранилища метрик.
type (
	// Repositories - интерфейс для получения метрики по типу и имени метрики.
	Repositories interface {
		GetMetric(context.Context, string, string) (string, error) // Метод для получения метрики по типу и имени метрики.
	}

	// ServerRepo - интерфейс для серверного хранилища метрик.
	ServerRepo interface {
		Repositories
		AddGauge(context.Context, string, float64) error      // Добавлеет в сервис новую метрики типа "gauge"
		AddCounter(context.Context, string, int64) error      // Добавлеет в сервис новую метрики типа "counter"
		GetAllMetrics(context.Context) (string, error)        // Возвращает все хранимые в сервисе метрики в виде строки
		AddMetricsFromSlice(context.Context, []Metric) error  // Добавляет в сервис метрики из слайса метрик
		GetAllMetricsSlice(context.Context) ([]Metric, error) // Возвращает все хранимые в сервисе метрики в виде слайса метрик
		Bootstrap(context.Context) error                      // Инициализирует хранилище метрик
	}

	// Metric - структура для работы с метриками json формата
	Metric struct {
		ID    string   `json:"id"`              // имя метрики
		MType string   `json:"type"`            // параметр, принимающий значение gauge или counter
		Delta *int64   `json:"delta,omitempty"` // значение метрики в случае передачи counter
		Value *float64 `json:"value,omitempty"` // значение метрики в случае передачи gauge
	}
)

// Metric_String возвращает представление метрики в виде строки
func (metrcic Metric) String() string {
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
