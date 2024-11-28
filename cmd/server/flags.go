package main

import (
	"flag"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/config"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/encrypt"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/hasher"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/ipfilter"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/saver"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/tools/encryption"
)

var (
	flagNetAddr         string
	flagLogLevel        string
	flagStoreInterval   int
	flagFileStoragePath string
	flagRestore         bool
	flagDatabaseDsn     string
	flagKey             string
	flagCryptoKey       string
	flagConfigFile      string
	flagTrustedSubnet   string
)

// Определяют способ хранения метрик.
const (
	// SAVEINRAM устанавливает созранение метрик в оперативную память
	SAVEINRAM = iota
	// SAVEINRAM устанавливает созранение метрик в файл
	SAVEINFILE
	// SAVEINRAM устанавливает созранение метрик в базу данных
	SAVEINDATABASE
)

func parseFlags() int {
	flag.StringVar(&flagNetAddr, "a", ":8080", "address and port to run server")
	flag.StringVar(&flagLogLevel, "l", "info", "log level")
	// настройка флагов для хранения метрик в файле
	flagStoreIntervalTemp := flag.Int("i", 300, "interval of saving metrics to the file")
	flag.StringVar(&flagFileStoragePath, "f", "", "path address to saving metrics file") // Путь к файлу по умолчанию: ./metrics.json
	flagRestoreTemp := flag.Bool("r", true, "for define needed of loading metrics from file while server starting")
	// настройка флагов для хранения метрик в базе данных
	flag.StringVar(&flagDatabaseDsn, "d", "", "database connection address") // host=localhost user=metrics password=metrics dbname=metricsdb  sslmode=disable
	flag.StringVar(&flagKey, "k", "", "key for hashing data")
	flag.StringVar(&flagCryptoKey, "crypto-key", "", "private key for asymmetric encryption")
	flag.StringVar(&flagConfigFile, "c", "", "name of configuration file")
	flag.StringVar(&flagTrustedSubnet, "t", "", "Classless Distributed Ranging (CIDR) string representation")

	flag.Parse()
	flagStoreInterval = *flagStoreIntervalTemp
	flagRestore = *flagRestoreTemp

	// параметры конфигурации переопределяются глобальными переменными, даже если они были переданы через аргументы командной строки
	parseEnvironment()

	// параметры конфигурации переопределяются параметрами из файла конфигурции, даже если они были переданы через аргументы командной строки
	// или глобальные переменные
	parseConfigFile()

	saver.SetStoreInterval(time.Duration(flagStoreInterval))
	saver.SetFilestoragePath(flagFileStoragePath)
	saver.SetRestore(flagRestore)
	hasher.SetKey(flagKey)
	encrypt.SetCryptoGrapher(encryption.Initialize("", flagCryptoKey))
	ipfilter.SetTrustedSubnet(flagTrustedSubnet)

	if flagDatabaseDsn != "" {
		return SAVEINDATABASE
	} else if flagFileStoragePath != "" {
		return SAVEINFILE
	}
	return SAVEINRAM
}

// parseEnvironment - функция для переопределения параметров конфигурации из глобальных переменных.
func parseEnvironment() {
	if envRunAddr := os.Getenv("ADDRESS"); envRunAddr != "" {
		flagNetAddr = envRunAddr
	}
	if envLogLevel := os.Getenv("SERVER_LOG_LEVEL"); envLogLevel != "" {
		flagLogLevel = envLogLevel
	}
	if envStoreInterval := os.Getenv("STORE_INTERVAL"); envStoreInterval != "" {
		interval, err := strconv.Atoi(envStoreInterval)
		if err != nil {
			log.Fatalf("Parse STORE_INTERVAL global variable error: %v\n", err)
		}
		flagStoreInterval = interval
	}
	if envFileStoragePath := os.Getenv("FILE_STORAGE_PATH"); envFileStoragePath != "" {
		flagFileStoragePath = envFileStoragePath
	}
	if envRestore := os.Getenv("RESTORE"); envRestore != "" {
		r, err := strconv.ParseBool(envRestore)
		if err != nil {
			log.Fatalf("Parse RESTORE global variable error: %v\n", err)
		}
		flagRestore = r
	}
	if envDatabaseDsn := os.Getenv("DATABASE_DSN"); envDatabaseDsn != "" {
		flagDatabaseDsn = envDatabaseDsn
	}
	if envKey := os.Getenv("KEY"); envKey != "" {
		flagKey = envKey
	}
	if envCryptoKey := os.Getenv("CRYPTO_KEY"); envCryptoKey != "" {
		flagCryptoKey = envCryptoKey
	}
	if envConfigFile := os.Getenv("CONFIG"); envConfigFile != "" {
		flagConfigFile = envConfigFile
	}
	if envTrustedSubnet := os.Getenv("TRUSTED_SUBNET"); envTrustedSubnet != "" {
		flagTrustedSubnet = envTrustedSubnet
	}
}

// parseConfigFile - функция для переопределения параметров конфигурации из файла конфигурации.
func parseConfigFile() {
	// если не указан файл конфигурации, то оставляю параметры запуска без изменения
	if flagConfigFile == "" {
		return
	}
	configs, err := config.ParseConfigFile(flagConfigFile)
	if err != nil {
		log.Fatalf("parse config file error: %v\n", err)
	}

	// обновляю параметры запуска
	flagNetAddr = configs.Address
	flagRestore = configs.Restore
	flagStoreInterval = int(configs.StoreInterval.Duration.Seconds())
	flagFileStoragePath = configs.StoreFile
	flagDatabaseDsn = configs.DatabaseDSN
	flagCryptoKey = configs.CryptoKey
	flagTrustedSubnet = configs.TrustedSubnet
}
