package storage

import (
	"fmt"
	"runtime"
	"strconv"
	"sync"
	"time"

	"golang.org/x/exp/rand"
)

var GaugeMetrics []string
var AllMetrics []string

func init() {
	GaugeMetrics = []string{"Alloc", "BuckHashSys", "Frees", "GCCPUFraction", "GCSys", "HeapAlloc", "HeapIdle", "HeapInuse", "HeapObjects",
		"HeapReleased", "HeapSys", "LastGC", "Lookups", "MCacheInuse", "MCacheSys", "MSpanInuse", "MSpanSys", "Mallocs", "NextGC", "NumForcedGC", "NumGC",
		"OtherSys", "PauseTotalNs", "StackInuse", "StackSys", "Sys", "TotalAlloc"}
	AllMetrics = []string{"Alloc", "BuckHashSys", "Frees", "GCCPUFraction", "GCSys", "HeapAlloc", "HeapIdle", "HeapInuse", "HeapObjects",
		"HeapReleased", "HeapSys", "LastGC", "Lookups", "MCacheInuse", "MCacheSys", "MSpanInuse", "MSpanSys", "Mallocs", "NextGC", "NumForcedGC", "NumGC",
		"OtherSys", "PauseTotalNs", "StackInuse", "StackSys", "Sys", "TotalAlloc", "PollCount", "RandomValue"}
}

// MetricsStats структура для хранения метрик
type MetricsStats struct {
	sync.Mutex
	runtime.MemStats
	PollCount   int64
	RandomValue float64
}

// CollectMetrics собирает метрики
func (metrics *MetricsStats) CollectMetrics() {
	metrics.PollCount++
	runtime.ReadMemStats(&metrics.MemStats)
}

func GetRandomMetricName() string {
	// Инициализируем случайный источник с текущим временем
	rand.Seed(uint64(time.Now().UnixNano()))
	// Выбираем случайный индекс
	randomIndex := rand.Intn(len(GaugeMetrics))
	// Выбираем случайный элемент
	return GaugeMetrics[randomIndex]
}

func (metrics *MetricsStats) GetMetricString(name string) (typeMetric, value string, err error) {
	switch name {
	case "Alloc":
		return "gauge", strconv.FormatUint(metrics.Alloc, 10), nil
	case "BuckHashSys":
		return "gauge", strconv.FormatUint(metrics.BuckHashSys, 10), nil
	case "Frees":
		return "gauge", strconv.FormatUint(metrics.Frees, 10), nil
	case "GCCPUFraction":
		return "gauge", strconv.FormatFloat(metrics.GCCPUFraction, 'f', 6, 64), nil
	case "GCSys":
		return "gauge", strconv.FormatUint(metrics.GCSys, 10), nil
	case "HeapAlloc":
		return "gauge", strconv.FormatUint(metrics.HeapAlloc, 10), nil
	case "HeapIdle":
		return "gauge", strconv.FormatUint(metrics.HeapIdle, 10), nil
	case "HeapInuse":
		return "gauge", strconv.FormatUint(metrics.HeapInuse, 10), nil
	case "HeapObjects":
		return "gauge", strconv.FormatUint(metrics.HeapObjects, 10), nil
	case "HeapReleased":
		return "gauge", strconv.FormatUint(metrics.HeapReleased, 10), nil
	case "HeapSys":
		return "gauge", strconv.FormatUint(metrics.HeapSys, 10), nil
	case "LastGC":
		return "gauge", strconv.FormatUint(metrics.LastGC, 10), nil
	case "Lookups":
		return "gauge", strconv.FormatUint(metrics.Lookups, 10), nil
	case "MCacheInuse":
		return "gauge", strconv.FormatUint(metrics.MCacheInuse, 10), nil
	case "MCacheSys":
		return "gauge", strconv.FormatUint(metrics.MCacheSys, 10), nil
	case "MSpanInuse":
		return "gauge", strconv.FormatUint(metrics.MSpanInuse, 10), nil
	case "MSpanSys":
		return "gauge", strconv.FormatUint(metrics.MSpanSys, 10), nil
	case "Mallocs":
		return "gauge", strconv.FormatUint(metrics.Mallocs, 10), nil
	case "NextGC":
		return "gauge", strconv.FormatUint(metrics.NextGC, 10), nil
	case "NumForcedGC":
		return "gauge", strconv.FormatUint(uint64(metrics.NumForcedGC), 10), nil
	case "NumGC":
		return "gauge", strconv.FormatUint(uint64(metrics.NumGC), 10), nil
	case "OtherSys":
		return "gauge", strconv.FormatUint(metrics.OtherSys, 10), nil
	case "PauseTotalNs":
		return "gauge", strconv.FormatUint(metrics.PauseTotalNs, 10), nil
	case "StackInuse":
		return "gauge", strconv.FormatUint(metrics.StackInuse, 10), nil
	case "StackSys":
		return "gauge", strconv.FormatUint(metrics.StackSys, 10), nil
	case "Sys":
		return "gauge", strconv.FormatUint(metrics.Sys, 10), nil
	case "TotalAlloc":
		return "gauge", strconv.FormatUint(metrics.TotalAlloc, 10), nil
	case "PollCount":
		return "counter", strconv.FormatUint(uint64(metrics.PollCount), 10), nil
	case "RandomValue":
		return metrics.GetMetricString(GetRandomMetricName())
	}
	return "", "", fmt.Errorf("metric %s is not exist", name)
}

func NewMetricsStats() *MetricsStats {
	return &MetricsStats{}
}
