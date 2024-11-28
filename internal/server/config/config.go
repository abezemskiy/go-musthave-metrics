package config

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/repositories"
)

// Configs представляет структуру конфигурации.
type Configs struct {
	Address       string                `json:"address"`        // аналог переменной окружения ADDRESS или флага -a
	Restore       bool                  `json:"restore"`        // аналог переменной окружения RESTORE или флага -r
	StoreInterval repositories.Duration `json:"store_interval"` // аналог переменной окружения STORE_INTERVAL или флага -i
	StoreFile     string                `json:"store_file"`     // аналог переменной окружения FILE_STORAGE_PATH или -f
	DatabaseDSN   string                `json:"database_dsn"`   // аналог переменной окружения DATABASE_DSN или флага -d
	CryptoKey     string                `json:"crypto_key"`     // аналог переменной окружения CRYPTO_KEY или флага -crypto-key
	TrustedSubnet string                `json:"trusted_subnet"` // аналог переменной окружения TRUSTED_SUBNET или флага -t
}

// ParseConfigFile - функция для переопределения параметров конфигурации из файла конфигурации.
func ParseConfigFile(configFileNAme string) (Configs, error) {
	var configs Configs
	f, err := os.Open(configFileNAme)
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
