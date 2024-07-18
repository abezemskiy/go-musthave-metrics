package repositories

type Repositories interface {
	AddGauge(string, float64)
	AddCounter(string, int64)
	GetMetric(string, string) (string, error)
	GetAllMetrics() string
}
