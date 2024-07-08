package repositories

type Repositories interface {
	AddGauge(string, float64)
	AddCounter(string, int64)
}


type MemStorage struct {
	gauges   map[string]float64
	counters map[string]int64
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

func (storage *MemStorage) AddGauge(name string, guage float64) {
	storage.gauges[name] = guage
}

func (storage *MemStorage) AddCounter(name string, counter int64) {
	storage.counters[name] += counter
}
