package storage

import (
	"fmt"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	"go.uber.org/zap"
	"golang.org/x/exp/rand"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/logger"
)

var GaugeMetrics []string
var AllMetrics []string

func init() {
	GaugeMetrics = []string{"Alloc", "BuckHashSys", "Frees", "GCCPUFraction", "GCSys", "HeapAlloc", "HeapIdle", "HeapInuse", "HeapObjects",
		"HeapReleased", "HeapSys", "LastGC", "Lookups", "MCacheInuse", "MCacheSys", "MSpanInuse", "MSpanSys", "Mallocs", "NextGC", "NumForcedGC", "NumGC",
		"OtherSys", "PauseTotalNs", "StackInuse", "StackSys", "Sys", "TotalAlloc", "TotalMemory", "FreeMemory", "CPUutilization1"}
	AllMetrics = []string{"Alloc", "BuckHashSys", "Frees", "GCCPUFraction", "GCSys", "HeapAlloc", "HeapIdle", "HeapInuse", "HeapObjects",
		"HeapReleased", "HeapSys", "LastGC", "Lookups", "MCacheInuse", "MCacheSys", "MSpanInuse", "MSpanSys", "Mallocs", "NextGC", "NumForcedGC", "NumGC",
		"OtherSys", "PauseTotalNs", "StackInuse", "StackSys", "Sys", "TotalAlloc", "TotalMemory", "FreeMemory", "CPUutilization1", "PollCount", "RandomValue"}
}

// MetricsStats структура для хранения метрик
type MetricsStats struct {
	sync.Mutex
	runtime.MemStats
	PollCount       int64
	RandomValue     float64
	TotalMemory     float64
	FreeMemory      float64
	CPUutilization1 float64
}

func collectExtraMetrics(ch chan<- map[string]float64) {
	res := make(map[string]float64, 0)

	v, err := mem.VirtualMemory()
	if err == nil {
		res["TotalMemory"] = float64(v.Total)
		res["FreeMemory"] = float64(v.Free)
	} else {
		logger.AgentLog.Error("collect memory metrics error by gopsutil package", zap.String("error", err.Error()))
	}

	c, err := cpu.Percent(time.Second, true)
	if err == nil {
		res["CPUutilization1"] = float64(c[0])
	} else {
		logger.AgentLog.Error("collect cpu metrics error by gopsutil package", zap.String("error", err.Error()))
	}
	ch <- res
}

// CollectMetrics собирает метрики
func (metrics *MetricsStats) CollectMetrics() {
	// Сбор дополнительных метрик в отдельной горутине
	extraM := make(chan map[string]float64, 1)
	go collectExtraMetrics(extraM)

	metrics.Lock()
	defer metrics.Unlock()

	metrics.PollCount = 1
	runtime.ReadMemStats(&metrics.MemStats)

	extraMetrics := <-extraM
	for name, value := range extraMetrics {
		switch name {
		case "TotalMemory":
			metrics.TotalMemory = value
		case "FreeMemory":
			metrics.FreeMemory = value
		case "CPUutilization1":
			metrics.CPUutilization1 = value
		}
	}
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
	case "TotalMemory":
		return "gauge", strconv.FormatUint(uint64(metrics.TotalMemory), 10), nil
	case "FreeMemory":
		return "gauge", strconv.FormatUint(uint64(metrics.FreeMemory), 10), nil
	case "CPUutilization1":
		return "gauge", strconv.FormatUint(uint64(metrics.CPUutilization1), 10), nil
	}

	return "", "", fmt.Errorf("metric %s is not exist", name)
}

func NewMetricsStats() *MetricsStats {
	return &MetricsStats{}
}
