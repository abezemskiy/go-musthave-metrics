package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/saver"
)

var (
	flagNetAddr         string
	flagLogLevel        string
	flagStoreInterval   int
	flagFileStoragePath string
	flagRestore         bool
	flagDatabaseDsn    string
)

func parseFlags() {
	flag.StringVar(&flagNetAddr, "a", ":8080", "address and port to run server")
	flag.StringVar(&flagLogLevel, "l", "info", "log level")
	flagStoreIntervalTemp := flag.Int("i", 300, "interval of saving metrics to the file")
	flag.StringVar(&flagFileStoragePath, "f", "./metrics.json", "path address to saving metrics file")
	flagRestoreTemp := flag.Bool("r", true, "for define needed of loading metrics from file while server starting")
	ps := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable",
		`localhost`, `default`, `XXXXXXXX`, `default`)
	flag.StringVar(&flagDatabaseDsn, "d", ps, "database connection address")

	flag.Parse()
	flagStoreInterval = *flagStoreIntervalTemp
	flagRestore = *flagRestoreTemp

	// для случаев, когда в переменной окружения ADDRESS присутствует непустое значение,
	// переопределим адрес запуска сервера,
	// даже если он был передан через аргумент командной строки
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

	saver.SetStoreInterval(time.Duration(flagStoreInterval))
	saver.SetFilestoragePath(flagFileStoragePath)
	saver.SetRestore(flagRestore)
}
