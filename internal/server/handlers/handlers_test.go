package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/repositories"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/saver"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/storage"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOtherRequest(t *testing.T) {

	type want struct {
		code        int
		contentType string
	}
	tests := []struct {
		name    string
		request string
		want    want
	}{
		{
			name:    "Global addres",
			request: "/",
			want: want{
				code:        404,
				contentType: "text/plain",
			},
		},
		{
			name:    "Whrong addres",
			request: "/whrong",
			want: want{
				code:        404,
				contentType: "text/plain",
			},
		},
		{
			name:    "Mistake addres",
			request: "/updat",
			want: want{
				code:        404,
				contentType: "text/plain",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, tt.request, nil)
			w := httptest.NewRecorder()
			OtherRequest(w, request)

			res := w.Result()
			defer res.Body.Close() // Закрываем тело ответа
			// проверяем код ответа
			assert.Equal(t, tt.want.code, res.StatusCode)
		})
	}
}

func TestGetMetricJSON(t *testing.T) {
	stor := storage.NewMemStorage(map[string]float64{"testgauge1": 3.134, "testgauge2": 10, "alloc": 233184}, map[string]int64{"testcount1": 4, "testcount2": 1})

	delta := func(d int64) *int64 {
		return &d
	}
	value := func(v float64) *float64 {
		return &v
	}
	type want struct {
		code        int
		contentType string
		metrics     repositories.Metrics
	}
	tests := []struct {
		name    string
		request string
		body    repositories.Metrics
		want    want
	}{
		{
			name:    "Counter testcount#1",
			request: "/value/",
			body: repositories.Metrics{
				ID:    "testcount1",
				MType: "counter",
			},
			want: want{
				code:        200,
				contentType: "application/json",
				metrics: repositories.Metrics{
					ID:    "testcount1",
					MType: "counter",
					Delta: delta(4),
				},
			},
		},
		{
			name:    "Counter error#1",
			request: "/value/",
			body: repositories.Metrics{
				ID:    "testcount3",
				MType: "counter",
			},
			want: want{
				code:        404,
				contentType: "application/json",
				metrics: repositories.Metrics{
					ID:    "testcount3",
					MType: "counter",
				},
			},
		},
		{
			name:    "Counter error#1",
			request: "/value/",
			body: repositories.Metrics{
				ID:    "testcount2",
				MType: "couunter",
			},
			want: want{
				code:        404,
				contentType: "application/json",
				metrics: repositories.Metrics{
					ID:    "testcount2",
					MType: "couunter",
				},
			},
		},
		{
			name:    "Gauge testgauge#1",
			request: "/value/",
			body: repositories.Metrics{
				ID:    "testgauge1",
				MType: "gauge",
			},
			want: want{
				code:        200,
				contentType: "application/json",
				metrics: repositories.Metrics{
					ID:    "testgauge1",
					MType: "gauge",
					Value: value(3.134),
				},
			},
		},
		{
			name:    "Gauge error#1",
			request: "/value",
			body: repositories.Metrics{
				ID:    "testgauge3",
				MType: "gauge",
			},
			want: want{
				code:        404,
				contentType: "application/json",
				metrics: repositories.Metrics{
					ID:    "testgauge3",
					MType: "gauge",
				},
			},
		},
		{
			name:    "Gauge error#2",
			request: "/value",
			body: repositories.Metrics{
				ID:    "testgauge2",
				MType: "gauuge",
			},
			want: want{
				code:        404,
				contentType: "application/json",
				metrics: repositories.Metrics{
					ID:    "testgauge2",
					MType: "gauuge",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Post("/value/", func(res http.ResponseWriter, req *http.Request) {
				GetMetricJSON(res, req, stor)
			})

			// сериализую струтктуру с метриками в json
			body, err := json.Marshal(tt.body)
			if err != nil {
				t.Error(err, "Marshall message error")
			}

			request := httptest.NewRequest(http.MethodPost, tt.request, bytes.NewBuffer(body))
			w := httptest.NewRecorder()
			r.ServeHTTP(w, request)

			res := w.Result()
			defer res.Body.Close() // Закрываем тело ответа
			// проверяем код ответа
			assert.Equal(t, tt.want.code, res.StatusCode)

			// Проверяю тело ответа, если код ответа 200
			if res.StatusCode == http.StatusOK {
				// Десериализую структуру с метриками
				var resMetric repositories.Metrics
				dec := json.NewDecoder(res.Body)
				er := dec.Decode(&resMetric)
				require.NoError(t, er)

				assert.Equal(t, tt.want.metrics, resMetric)
			}
		})
	}
}

func TestUpdateMetrics(t *testing.T) {
	stor := storage.NewMemStorage(nil, map[string]int64{"testcount1": 1})
	saver, err := saver.NewSaverWriter("./TestUpdateMetrics.json")

	require.NoError(t, err)
	type want struct {
		code        int
		contentType string
		storage     storage.MemStorage
	}
	tests := []struct {
		name    string
		arg     storage.MemStorage
		request string
		want    want
	}{
		{
			name:    "Counter testcount#1",
			arg:     *stor,
			request: "/update/counter/testcount1/3",
			want: want{
				code:        200,
				contentType: "text/plain",
				storage:     *storage.NewMemStorage(nil, map[string]int64{"testcount1": 4}),
			},
		},
		{
			name:    "Counter testcount#2",
			arg:     *stor,
			request: "/update/counter/testcount2/1",
			want: want{
				code:        200,
				contentType: "text/plain",
				storage:     *storage.NewMemStorage(nil, map[string]int64{"testcount1": 4, "testcount2": 1}),
			},
		},
		{
			name:    "Counter testguage#1",
			arg:     *stor,
			request: "/update/gauge/testgauge1/1",
			want: want{
				code:        200,
				contentType: "text/plain",
				storage:     *storage.NewMemStorage(map[string]float64{"testgauge1": 1}, map[string]int64{"testcount1": 4, "testcount2": 1}),
			},
		},
		{
			name:    "Counter testguage#2",
			arg:     *stor,
			request: "/update/gauge/testgauge1/3",
			want: want{
				code:        200,
				contentType: "text/plain",
				storage:     *storage.NewMemStorage(map[string]float64{"testgauge1": 3}, map[string]int64{"testcount1": 4, "testcount2": 1}),
			},
		},
		{
			name:    "Counter testguage#3",
			arg:     *stor,
			request: "/update/gauge/testgauge2/10",
			want: want{
				code:        200,
				contentType: "text/plain",
				storage:     *storage.NewMemStorage(map[string]float64{"testgauge1": 3, "testgauge2": 10}, map[string]int64{"testcount1": 4, "testcount2": 1}),
			},
		},
		{
			name:    "Counter errort#1",
			arg:     *stor,
			request: "/update/counter/testcount1/aaaaa",
			want: want{
				code:        400,
				contentType: "text/plain",
				storage:     *storage.NewMemStorage(map[string]float64{"testgauge1": 3, "testgauge2": 10}, map[string]int64{"testcount1": 4, "testcount2": 1}),
			},
		},
		{
			name:    "Counter errort#2",
			arg:     *stor,
			request: "/update/counter/testcount1/",
			want: want{
				code:        404,
				contentType: "text/plain",
				storage:     *storage.NewMemStorage(map[string]float64{"testgauge1": 3, "testgauge2": 10}, map[string]int64{"testcount1": 4, "testcount2": 1}),
			},
		},
		{
			name:    "Counter errort#3",
			arg:     *stor,
			request: "/update/counter/testcount1/1.12",
			want: want{
				code:        400,
				contentType: "text/plain",
				storage:     *storage.NewMemStorage(map[string]float64{"testgauge1": 3, "testgauge2": 10}, map[string]int64{"testcount1": 4, "testcount2": 1}),
			},
		},
		{
			name:    "Guage errort#1",
			arg:     *stor,
			request: "/update/gauge/testguage1/aaaaa",
			want: want{
				code:        400,
				contentType: "text/plain",
				storage:     *storage.NewMemStorage(map[string]float64{"testgauge1": 3, "testgauge2": 10}, map[string]int64{"testcount1": 4, "testcount2": 1}),
			},
		},
		{
			name:    "Guage errort#2",
			arg:     *stor,
			request: "/update/gauge/testguage1/",
			want: want{
				code:        404,
				contentType: "text/plain",
				storage:     *storage.NewMemStorage(map[string]float64{"testgauge1": 3, "testgauge2": 10}, map[string]int64{"testcount1": 4, "testcount2": 1}),
			},
		},
		{
			name:    "BadRequest status#1",
			arg:     *stor,
			request: "/update/gauges/testguage1/aaaaa",
			want: want{
				code:        400,
				contentType: "text/plain",
				storage:     *storage.NewMemStorage(map[string]float64{"testgauge1": 3, "testgauge2": 10}, map[string]int64{"testcount1": 4, "testcount2": 1}),
			},
		},
		{
			name:    "Notfound status#1",
			arg:     *stor,
			request: "/update/gauge/testguage1",
			want: want{
				code:        404,
				contentType: "text/plain",
				storage:     *storage.NewMemStorage(map[string]float64{"testgauge1": 3, "testgauge2": 10}, map[string]int64{"testcount1": 4, "testcount2": 1}),
			},
		},
		{
			name:    "Counter errort#4",
			arg:     *stor,
			request: "/update/gauge/alloc/233184",
			want: want{
				code:        200,
				contentType: "text/plain",
				storage:     *storage.NewMemStorage(map[string]float64{"testgauge1": 3, "testgauge2": 10, "alloc": 233184}, map[string]int64{"testcount1": 4, "testcount2": 1}),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Post("/update/{metricType}/{metricName}/{metricValue}", func(res http.ResponseWriter, req *http.Request) {
				UpdateMetrics(res, req, &tt.arg, saver)
			})

			request := httptest.NewRequest(http.MethodPost, tt.request, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, request)

			res := w.Result()
			defer res.Body.Close() // Закрываем тело ответа
			// проверяем код ответа
			assert.Equal(t, tt.want.code, res.StatusCode)
			assert.Equal(t, tt.want.storage, tt.arg)
		})
	}
	// Удаляю тестовый файл
	er := os.Remove("./TestUpdateMetrics.json")
	require.NoError(t, er)
}

func TestUpdateMetricsJSON(t *testing.T) {
	stor := storage.NewMemStorage(nil, map[string]int64{"testcount1": 1})
	saver, err := saver.NewSaverWriter("./TestUpdateMetricsJSON.json")
	require.NoError(t, err)

	delta := func(d int64) *int64 {
		return &d
	}
	value := func(v float64) *float64 {
		return &v
	}
	type want struct {
		code        int
		contentType string
		storage     storage.MemStorage
	}
	tests := []struct {
		name    string
		arg     storage.MemStorage
		request string
		body    repositories.Metrics
		want    want
	}{
		{
			name:    "Counter testcount#1",
			arg:     *stor,
			request: "/update",
			body: repositories.Metrics{
				ID:    "testcount1",
				MType: "counter",
				Delta: delta(3),
			},
			want: want{
				code:        200,
				contentType: "application/json",
				storage:     *storage.NewMemStorage(nil, map[string]int64{"testcount1": 4}),
			},
		},
		{
			name:    "Counter testcount#2",
			arg:     *stor,
			request: "/update",
			body: repositories.Metrics{
				ID:    "testcount2",
				MType: "counter",
				Delta: delta(1),
			},
			want: want{
				code:        200,
				contentType: "application/json",
				storage:     *storage.NewMemStorage(nil, map[string]int64{"testcount1": 4, "testcount2": 1}),
			},
		},
		{
			name:    "Counter testguage#1",
			arg:     *stor,
			request: "/update",
			body: repositories.Metrics{
				ID:    "testgauge1",
				MType: "gauge",
				Value: value(1),
			},
			want: want{
				code:        200,
				contentType: "application/json",
				storage:     *storage.NewMemStorage(map[string]float64{"testgauge1": 1}, map[string]int64{"testcount1": 4, "testcount2": 1}),
			},
		},
		{
			name:    "Counter testguage#2",
			arg:     *stor,
			request: "/update",
			body: repositories.Metrics{
				ID:    "testgauge1",
				MType: "gauge",
				Value: value(3),
			},
			want: want{
				code:        200,
				contentType: "application/json",
				storage:     *storage.NewMemStorage(map[string]float64{"testgauge1": 3}, map[string]int64{"testcount1": 4, "testcount2": 1}),
			},
		},
		{
			name:    "Counter testguage#3",
			arg:     *stor,
			request: "/update",
			body: repositories.Metrics{
				ID:    "testgauge2",
				MType: "gauge",
				Value: value(10),
			},
			want: want{
				code:        200,
				contentType: "application/json",
				storage:     *storage.NewMemStorage(map[string]float64{"testgauge1": 3, "testgauge2": 10}, map[string]int64{"testcount1": 4, "testcount2": 1}),
			},
		},
		{
			name:    "Counter errort#1",
			arg:     *stor,
			request: "/update",
			body: repositories.Metrics{
				ID:    "testcount1",
				MType: "counteer",
				Delta: delta(10),
			},
			want: want{
				code:        400,
				contentType: "application/json",
				storage:     *storage.NewMemStorage(map[string]float64{"testgauge1": 3, "testgauge2": 10}, map[string]int64{"testcount1": 4, "testcount2": 1}),
			},
		},
		{
			name:    "Guage errort#1",
			arg:     *stor,
			request: "/update",
			body: repositories.Metrics{
				ID:    "testguage1",
				MType: "gauuge",
				Value: value(10),
			},
			want: want{
				code:        400,
				contentType: "application/json",
				storage:     *storage.NewMemStorage(map[string]float64{"testgauge1": 3, "testgauge2": 10}, map[string]int64{"testcount1": 4, "testcount2": 1}),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Post("/update", func(res http.ResponseWriter, req *http.Request) {
				UpdateMetricsJSON(res, req, &tt.arg, saver)
			})

			// сериализую струтктуру с метриками в json
			body, err := json.Marshal(tt.body)
			if err != nil {
				t.Error(err, "Marshall message error")
			}

			request := httptest.NewRequest(http.MethodPost, tt.request, bytes.NewBuffer(body))
			w := httptest.NewRecorder()
			r.ServeHTTP(w, request)

			res := w.Result()
			defer res.Body.Close() // Закрываем тело ответа
			// проверяем код ответа
			assert.Equal(t, tt.want.code, res.StatusCode)
			assert.Equal(t, tt.want.storage, tt.arg)

			// Проверяю тело ответа, если код ответа 200
			if res.StatusCode == http.StatusOK {
				// Десериализую структуру с метриками
				var resMetric repositories.Metrics
				dec := json.NewDecoder(res.Body)
				er := dec.Decode(&resMetric)
				require.NoError(t, er)

				assert.Equal(t, tt.body, resMetric)
			}
		})
	}
	// Удаляю тестовый файл
	er := os.Remove("./TestUpdateMetricsJSON.json")
	require.NoError(t, er)
}
