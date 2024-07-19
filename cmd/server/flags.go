package main

import (
	"flag"
	"os"
)

var (
	flagNetAddr string
	flagLogLevel string
)

func parseFlags() {
	flag.StringVar(&flagNetAddr, "a", ":8080", "address and port to run server")
	flag.StringVar(&flagLogLevel, "l", "info", "log level")
	flag.Parse()

	// для случаев, когда в переменной окружения ADDRESS присутствует непустое значение,
	// переопределим адрес запуска сервера,
	// даже если он был передан через аргумент командной строки
	if envRunAddr := os.Getenv("SERVER_ADDRESS"); envRunAddr != "" {
		flagNetAddr = envRunAddr
	}
	if envLogLevel := os.Getenv("SERVER_LOG_LEVEL"); envLogLevel != "" {
        flagLogLevel = envLogLevel
    }
}
