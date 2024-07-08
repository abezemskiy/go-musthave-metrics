package main

import (
	"net/http"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/repositories"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/serverhandlers"
)

func main() {
	storage := repositories.NewMemStorage()

	mux := http.NewServeMux()
	mux.HandleFunc("/", serverhandlers.HandlerOther)
	mux.HandleFunc("/update/", func(res http.ResponseWriter, req *http.Request) {
		serverhandlers.HandlerUpdate(res, req, storage)
	})

	err := http.ListenAndServe("localhost:8080", mux)
	if err != nil {
		panic(err)
	}
}
