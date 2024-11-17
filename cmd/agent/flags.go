package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/hasher"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/metrics/config"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/repositories"
)

var (
	flagNetAddr    string
	reportInterval *int
	pollInterval   *int
	flagLogLevel   string
	flagKey        string
	rateLimit      *int
	cryptoKey      string
	flagConfigFile string
)

// configs представляет структуру конфигурации
type configs struct {
	Address        string                `json:"address"`         // аналог переменной окружения ADDRESS или флага -a
	ReportInterval repositories.Duration `json:"report_interval"` // аналог переменной окружения REPORT_INTERVAL или флага -r
	PollInterval   repositories.Duration `json:"poll_interval"`   // аналог переменной окружения POLL_INTERVAL или флага -p
	CryptoKey      string                `json:"crypto_key"`      // аналог переменной окружения CRYPTO_KEY или флага -crypto-key
}

func parseFlags() {
	flag.StringVar(&flagNetAddr, "a", ":8080", "address and port to run server")

	reportInterval = flag.Int("r", 10, "report interval")
	pollInterval = flag.Int("p", 2, "poll interval")
	flag.StringVar(&flagLogLevel, "log", "info", "log level")
	flag.StringVar(&flagKey, "k", "", "key for hashing data")
	rateLimit = flag.Int("l", 1, "count of concurrent messages to server")
	flag.StringVar(&cryptoKey, "crypto-key", "", "public key for asymmetric encryption")
	flag.StringVar(&flagConfigFile, "c", "", "name of configuration file")

	flag.Parse()

	// для случаев, когда в переменной окружения ADDRESS присутствует непустое значение,
	// переопределим адрес агента,
	// даже если он был передан через аргумент командной строки
	parseEnvironment()

	// параметры конфигурации переопределяются параметрами из файла конфигурции, даже если они были переданы через аргументы командной строки
	// или глобальные переменные
	parseConfigFile()

	config.SetReportInterval(time.Duration(*reportInterval))
	config.SetPollInterval(time.Duration(*pollInterval))
	hasher.SetKey(flagKey)
}

// parseEnvironment - функция для переопределения параметров конфигурации из глобальных переменных.
func parseEnvironment() {
	if envRunAddr := os.Getenv("ADDRESS"); envRunAddr != "" {
		flagNetAddr = envRunAddr
	}
	if envReportInterval := os.Getenv("REPORT_INTERVAL"); envReportInterval != "" {
		val, err := strconv.Atoi(envReportInterval)
		if err != nil {
			log.Fatalln("Environment variable \"REPORT_INTERVAL\" must be int")
		}
		*reportInterval = val
	}
	if envPollInterval := os.Getenv("POLL_INTERVAL"); envPollInterval != "" {
		val, err := strconv.Atoi(envPollInterval)
		if err != nil {
			log.Fatalln("Environment variable \"POLL_INTERVAL\" must be int")
		}
		*pollInterval = val
	}
	if envLogLevel := os.Getenv("AGENT_LOG_LEVEL"); envLogLevel != "" {
		flagLogLevel = envLogLevel
	}
	if envKey := os.Getenv("KEY"); envKey != "" {
		flagKey = envKey
	}

	if envRateLimit := os.Getenv("RATE_LIMIT"); envRateLimit != "" {
		val, err := strconv.Atoi(envRateLimit)
		if err != nil {
			log.Fatalln("Environment variable \"POLL_INTERVAL\" must be int")
		}
		*rateLimit = val
	}
	if envCryptoKey := os.Getenv("CRYPTO_KEY"); envCryptoKey != "" {
		cryptoKey = envCryptoKey
	}
	if envConfigFile := os.Getenv("CONFIG"); envConfigFile != "" {
		flagConfigFile = envConfigFile
	}
}

// parseConfigFile - функция для переопределения параметров конфигурации из файла конфигурации.
func parseConfigFile() {
	// елси на указан файл конфигурации, то оставляю параметры запуска без изменения
	if flagConfigFile == "" {
		return
	}
	var configs configs
	f, err := os.Open(flagConfigFile)
	if err != nil {
		log.Fatalf("Open cofiguration file error: %v\n", err)
	}
	reader := bufio.NewReader(f)
	dec := json.NewDecoder(reader)
	err = dec.Decode(&configs)
	if err != nil {
		log.Fatalf("Open cofiguration file error: %v\n", err)
	}

	// обновляю параметры запуска
	flagNetAddr = configs.Address
	*reportInterval = int(configs.ReportInterval.Duration.Seconds())
	*pollInterval = int(configs.PollInterval.Duration.Seconds())
	cryptoKey = configs.CryptoKey
}
