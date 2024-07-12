package main

import (
	"flag"
	"os"
)

var flagNetAddr string

func parseFlags() {
	flag.StringVar(&flagNetAddr, "a", ":8080", "address and port to run server")
	flag.Parse()

	// для случаев, когда в переменной окружения ADDRESS присутствует непустое значение,
	// переопределим адрес запуска сервера,
	// даже если он был передан через аргумент командной строки
	if envRunAddr := os.Getenv("ADDRESS"); envRunAddr != "" {
		flagNetAddr = envRunAddr
	}
}
