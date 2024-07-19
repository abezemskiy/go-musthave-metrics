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
)
