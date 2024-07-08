package main

import (
	"flag"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agenthandlers"
)

var flagNetAddr string

var (
	reportInterval *int
	pollInterval   *int
)

func parseFlags() {
	flag.StringVar(&flagNetAddr, "a", ":8080", "address and port to run server")

	reportInterval = flag.Int("r", 10, "report interval")
	pollInterval = flag.Int("p", 2, "poll interval")

	flag.Parse()

	// для случаев, когда в переменной окружения ADDRESS присутствует непустое значение,
	// переопределим адрес сервера,
	// даже если он был передан через аргумент командной строки
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

	agenthandlers.SetReportInterval(time.Duration(*reportInterval))
	agenthandlers.SetPollInterval(time.Duration(*pollInterval))
}
