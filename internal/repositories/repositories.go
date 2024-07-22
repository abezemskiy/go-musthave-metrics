package repositories

type (
	Repositories interface {
		GetMetric(string, string) (string, error)
	}

	ServerRepo interface {
		Repositories
		AddGauge(string, float64)
		AddCounter(string, int64)
		GetAllMetrics() string
	}

	// Структура для работы с метриками json формата

	Metrics struct {
		ID    string   `json:"id"`              // имя метрики
		MType string   `json:"type"`            // параметр, принимающий значение gauge или counter
		Delta *int64   `json:"delta,omitempty"` // значение метрики в случае передачи counter
		Value *float64 `json:"value,omitempty"` // значение метрики в случае передачи gauge
	}
)
