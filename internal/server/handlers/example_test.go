package handlers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/go-chi/chi/v5"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/repositories"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/handlers"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/storage"
)

func ExampleGetMetricJSON() {
	// создание заполненного хранилища метрик
	stor := storage.NewMemStorage(map[string]float64{"gauge1": 3.134, "alloc": 233184}, map[string]int64{"counter1": 4})

	r := chi.NewRouter()
	r.Post("/value/", func(res http.ResponseWriter, req *http.Request) {
		// выполнение запроса к хэндлеру
		handlers.GetMetricJSON(res, req, stor)
	})

	{
		metric := repositories.Metric{
			ID:    "gauge1",
			MType: "gauge",
		}

		// сериализую струтктуру с желаемой метрикой в json
		body, err := json.Marshal(metric)
		if err != nil {
			fmt.Println(err.Error())
		}

		request := httptest.NewRequest(http.MethodPost, "/value/", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, request)

		res := w.Result()
		defer res.Body.Close() // Закрываем тело ответа
		// проверяем код ответа
		fmt.Println(res.StatusCode)

		// Десериализую структуру с полученной метрикой от хэндлера
		var resMetric repositories.Metric
		dec := json.NewDecoder(res.Body)
		err = dec.Decode(&resMetric)
		if err != nil {
			fmt.Println(err.Error())
		}

		// Печатаю значение метрики, которое прислал хэндлер
		fmt.Println(*resMetric.Value)
	}
	{
		metric := repositories.Metric{
			ID:    "counter1",
			MType: "wrong type",
		}

		// сериализую струтктуру с желаемой метрикой в json
		body, err := json.Marshal(metric)
		if err != nil {
			fmt.Println(err.Error())
		}

		request := httptest.NewRequest(http.MethodPost, "/value/", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, request)

		res := w.Result()
		defer res.Body.Close() // Закрываем тело ответа
		// проверяем код ответа
		fmt.Println(res.StatusCode)
	}
	{
		metric := "wrong data"

		// сериализую струтктуру с желаемой метрикой в json
		body, err := json.Marshal(metric)
		if err != nil {
			fmt.Println(err.Error())
		}

		request := httptest.NewRequest(http.MethodPost, "/value/", bytes.NewBuffer(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, request)

		res := w.Result()
		defer res.Body.Close() // Закрываем тело ответа
		// проверяем код ответа
		fmt.Println(res.StatusCode)
	}

	// Output:
	// 200
	// 3.134
	// 404
	// 400
}
