package agenthandlers

import (
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
)

var (
	pollInterval   time.Duration = 2
	reportInterval time.Duration = 10
)

func SetPollInterval(interval time.Duration) {
	pollInterval = interval
}

func SetReportInterval(interval time.Duration) {
	reportInterval = interval
}

// MetricsStats структура для хранения метрик
type MetricsStats struct {
	sync.Mutex
	runtime.MemStats
	PollCount   int64
	RandomValue float64
}

// CollectMetrics собирает метрики
func CollectMetrics(metrics *MetricsStats) {
	metrics.Lock()
	defer metrics.Unlock()
	metrics.PollCount++
	runtime.ReadMemStats(&metrics.MemStats)
}

// CollectMetricsTimer запускает сбор метрик с интервалом
func CollectMetricsTimer(metrics *MetricsStats) {
	for {
		CollectMetrics(metrics)
		time.Sleep(pollInterval * time.Second)
	}
}

// Push отправляет метрику на сервер и возвращает ошибку при неудаче
func Push(address, action, typemetric, namemetric, valuemetric string, client *resty.Client) error {
	url := fmt.Sprintf("%s/%s/%s/%s/%s", address, action, typemetric, namemetric, valuemetric)
	//resp, err := http.Post(url, "text/plain", nil)
	resp, err := client.R().
		SetHeader("Content-Type", "text/plain").
		Post(url)

	if err != nil {
		return fmt.Errorf("error with post: %s, %w", url, err)
	}
	//defer resp.Body.Close()

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("received non-200 response status: %d for url: %s", resp.StatusCode(), url)
	}
	return nil
}

// PushMetrics отправляет все метрики
func PushMetrics(address, action string, metrics *MetricsStats, client *resty.Client) {
	metrics.Lock()
	defer metrics.Unlock()

	metricsToSend := []struct {
		typemetricgauge string
		name            string
		value           string
	}{
		{"gauge", "alloc", strconv.FormatUint(metrics.Alloc, 10)},
		{"gauge", "buckhashsys", strconv.FormatUint(metrics.BuckHashSys, 10)},
		{"gauge", "formatunit", strconv.FormatUint(metrics.Frees, 10)},
		{"gauge", "gccpufraction", strconv.FormatFloat(metrics.GCCPUFraction, 'f', 6, 64)},
		{"gauge", "gcsys", strconv.FormatUint(metrics.GCSys, 10)},
		{"gauge", "heapalloc", strconv.FormatUint(metrics.HeapAlloc, 10)},
		{"gauge", "heapidle", strconv.FormatUint(metrics.HeapIdle, 10)},
		{"gauge", "heapinuse", strconv.FormatUint(metrics.HeapInuse, 10)},
		{"gauge", "heapobjects", strconv.FormatUint(metrics.HeapObjects, 10)},
		{"gauge", "heapreleased", strconv.FormatUint(metrics.HeapReleased, 10)},
		{"gauge", "heapsys", strconv.FormatUint(metrics.HeapSys, 10)},
		{"gauge", "lastgc", strconv.FormatUint(metrics.LastGC, 10)},
		{"gauge", "lookups", strconv.FormatUint(metrics.Lookups, 10)},
		{"gauge", "mcacheinuse", strconv.FormatUint(metrics.MCacheInuse, 10)},
		{"gauge", "mcachesys", strconv.FormatUint(metrics.MCacheSys, 10)},
		{"gauge", "mspaninuse", strconv.FormatUint(metrics.MSpanInuse, 10)},
		{"gauge", "mspansys", strconv.FormatUint(metrics.MSpanSys, 10)},
		{"gauge", "mallocs", strconv.FormatUint(metrics.Mallocs, 10)},
		{"gauge", "nextgc", strconv.FormatUint(metrics.NextGC, 10)},
		{"gauge", "numforcedgc", strconv.FormatUint(uint64(metrics.NumForcedGC), 10)},
		{"gauge", "numgc", strconv.FormatUint(uint64(metrics.NumGC), 10)},
		{"gauge", "othersys", strconv.FormatUint(metrics.OtherSys, 10)},
		{"gauge", "pausetotalns", strconv.FormatUint(metrics.PauseTotalNs, 10)},
		{"gauge", "stackinuse", strconv.FormatUint(metrics.StackInuse, 10)},
		{"gauge", "stacksys", strconv.FormatUint(metrics.StackSys, 10)},
		{"gauge", "sys", strconv.FormatUint(metrics.Sys, 10)},
		{"gauge", "totalalloc", strconv.FormatUint(metrics.TotalAlloc, 10)},

		{"counter", "pollcount", strconv.FormatUint(uint64(metrics.PollCount), 10)},
		{"gauge", "randomvalue", strconv.FormatFloat(metrics.RandomValue, 'f', 6, 64)},
	}

	for _, metric := range metricsToSend {
		err := Push(address, action, metric.typemetricgauge, metric.name, metric.value, client)
		if err != nil {
			fmt.Printf("Failed to push metric %s: %v\n", metric.name, err)
		}
	}
}

// PushMetricsTimer запускает отправку метрик с интервалом
func PushMetricsTimer(address, action string, metrics *MetricsStats) {
	for {
		client := resty.New()
		PushMetrics(address, action, metrics, client)
		log.Print("Push metrics\n")
		time.Sleep(reportInterval * time.Second)
	}
}
