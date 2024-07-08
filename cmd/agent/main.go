package main

import (
	"flag"
	"time"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agenthandlers"
)

var addr = &NetAddress{
	Host: "localhost",
	Port: 8080,
}

func main() {
	// если интерфейс не реализован,
	// здесь будет ошибка компиляции
	_ = flag.Value(addr)
	// проверка реализации
	flag.Var(addr, "a", "Net address host:port")

	report := flag.Int("r", 10, "report interval")
	poll := flag.Int("p", 2, "poll interval")

	flag.Parse()

	agenthandlers.SetReportInterval(time.Duration(*report))
	agenthandlers.SetPollInterval(time.Duration(*poll))

	var metrics agenthandlers.MetricsStats
	go agenthandlers.CollectMetricsTimer(&metrics)
	time.Sleep(50 * time.Millisecond)
	go agenthandlers.PushMetricsTimer("http://"+addr.String(), "update", &metrics)

	// блокировка main, чтобы функции бесконечно выполнялись
	select {}
}
