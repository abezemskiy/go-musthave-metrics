package config

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/storage"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/repositories"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/tools/encryption"
)

var (
	pollInterval   time.Duration            = 2
	reportInterval time.Duration            = 10
	contextTimeout                          = 500 * time.Millisecond
	cryptoGrapher  encryption.Cryptographer // переменная, которая хранит структуру шифрования и расшифровки.
)

// Configs представляет структуру конфигурации
type Configs struct {
	Address        string                `json:"address"`         // аналог переменной окружения ADDRESS или флага -a
	ReportInterval repositories.Duration `json:"report_interval"` // аналог переменной окружения REPORT_INTERVAL или флага -r
	PollInterval   repositories.Duration `json:"poll_interval"`   // аналог переменной окружения POLL_INTERVAL или флага -p
	CryptoKey      string                `json:"crypto_key"`      // аналог переменной окружения CRYPTO_KEY или флага -crypto-key
	Protocol       string                `json:"protocol"`        // аналог переменной окружения PROTOCOL или флага -protocol
}

// SetPollInterval устанавливает интервал между сбором.
func SetPollInterval(interval time.Duration) {
	pollInterval = interval
}

// GetPollInterval - функция для получения интервала сбора метрик.
func GetPollInterval() time.Duration {
	return pollInterval
}

// SetReportInterval устанавливает интервал между отправками метрик на сервер.
func SetReportInterval(interval time.Duration) {
	reportInterval = interval
}

// GetReportInterval - функция для получения интервала отправки метрик на сервер.
func GetReportInterval() time.Duration {
	return reportInterval
}

// SetContextTimeout - установка таймаута.
func SetContextTimeout(timeout time.Duration) {
	contextTimeout = timeout
}

// GetContextTimeout - получение таймаута.
func GetContextTimeout() time.Duration {
	return contextTimeout
}

// SyncCollectMetrics - собирает метрики.
func SyncCollectMetrics(metrics *storage.MetricsStats) {
	metrics.CollectMetrics()
}

// SetCryptoGrapher - функция для установки структуры шифрования и расшифровки.
func SetCryptoGrapher(c *encryption.Cryptographer) {
	cryptoGrapher = *c
}

// GetCryptoGrapher - функция для получения структуры шифрования и расшифровки.
func GetCryptoGrapher() encryption.Cryptographer {
	return cryptoGrapher
}

// ParseConfigFile - функция для переопределения параметров конфигурации из файла конфигурации.
func ParseConfigFile(configFileName string) (Configs, error) {
	var configs Configs
	f, err := os.Open(configFileName)
	if err != nil {
		return Configs{}, fmt.Errorf("open cofiguration file error: %w", err)
	}
	reader := bufio.NewReader(f)
	dec := json.NewDecoder(reader)
	err = dec.Decode(&configs)
	if err != nil {
		return Configs{}, fmt.Errorf("parse cofiguration file error: %w", err)
	}

	return configs, nil
}
