package serverhandlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/repositories"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

func TestHandlerOther(t *testing.T) {

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
			HandlerOther(w, request)

			res := w.Result()
			defer res.Body.Close() // Закрываем тело ответа
			// проверяем код ответа
			assert.Equal(t, tt.want.code, res.StatusCode)
		})
	}
}

func TestHandlerUpdate(t *testing.T) {
	stor := repositories.NewMemStorage(nil, map[string]int64{"testcount1": 1})
	type want struct {
		code        int
		contentType string
		storage     repositories.MemStorage
	}
	tests := []struct {
		name    string
		arg     repositories.MemStorage
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
				storage:     *repositories.NewMemStorage(nil, map[string]int64{"testcount1": 4}),
			},
		},
		{
			name:    "Counter testcount#2",
			arg:     *stor,
			request: "/update/counter/testcount2/1",
			want: want{
				code:        200,
				contentType: "text/plain",
				storage:     *repositories.NewMemStorage(nil, map[string]int64{"testcount1": 4, "testcount2": 1}),
			},
		},
		{
			name:    "Counter testguage#1",
			arg:     *stor,
			request: "/update/gauge/testgauge1/1",
			want: want{
				code:        200,
				contentType: "text/plain",
				storage:     *repositories.NewMemStorage(map[string]float64{"testgauge1": 1}, map[string]int64{"testcount1": 4, "testcount2": 1}),
			},
		},
		{
			name:    "Counter testguage#2",
			arg:     *stor,
			request: "/update/gauge/testgauge1/3",
			want: want{
				code:        200,
				contentType: "text/plain",
				storage:     *repositories.NewMemStorage(map[string]float64{"testgauge1": 3}, map[string]int64{"testcount1": 4, "testcount2": 1}),
			},
		},
		{
			name:    "Counter testguage#3",
			arg:     *stor,
			request: "/update/gauge/testgauge2/10",
			want: want{
				code:        200,
				contentType: "text/plain",
				storage:     *repositories.NewMemStorage(map[string]float64{"testgauge1": 3, "testgauge2": 10}, map[string]int64{"testcount1": 4, "testcount2": 1}),
			},
		},
		{
			name:    "Counter errort#1",
			arg:     *stor,
			request: "/update/counter/testcount1/aaaaa",
			want: want{
				code:        400,
				contentType: "text/plain",
				storage:     *repositories.NewMemStorage(map[string]float64{"testgauge1": 3, "testgauge2": 10}, map[string]int64{"testcount1": 4, "testcount2": 1}),
			},
		},
		{
			name:    "Counter errort#2",
			arg:     *stor,
			request: "/update/counter/testcount1/",
			want: want{
				code:        404,
				contentType: "text/plain",
				storage:     *repositories.NewMemStorage(map[string]float64{"testgauge1": 3, "testgauge2": 10}, map[string]int64{"testcount1": 4, "testcount2": 1}),
			},
		},
		{
			name:    "Counter errort#3",
			arg:     *stor,
			request: "/update/counter/testcount1/1.12",
			want: want{
				code:        400,
				contentType: "text/plain",
				storage:     *repositories.NewMemStorage(map[string]float64{"testgauge1": 3, "testgauge2": 10}, map[string]int64{"testcount1": 4, "testcount2": 1}),
			},
		},
		{
			name:    "Guage errort#1",
			arg:     *stor,
			request: "/update/gauge/testguage1/aaaaa",
			want: want{
				code:        400,
				contentType: "text/plain",
				storage:     *repositories.NewMemStorage(map[string]float64{"testgauge1": 3, "testgauge2": 10}, map[string]int64{"testcount1": 4, "testcount2": 1}),
			},
		},
		{
			name:    "Guage errort#2",
			arg:     *stor,
			request: "/update/gauge/testguage1/",
			want: want{
				code:        404,
				contentType: "text/plain",
				storage:     *repositories.NewMemStorage(map[string]float64{"testgauge1": 3, "testgauge2": 10}, map[string]int64{"testcount1": 4, "testcount2": 1}),
			},
		},
		{
			name:    "BadRequest status#1",
			arg:     *stor,
			request: "/update/gauges/testguage1/aaaaa",
			want: want{
				code:        400,
				contentType: "text/plain",
				storage:     *repositories.NewMemStorage(map[string]float64{"testgauge1": 3, "testgauge2": 10}, map[string]int64{"testcount1": 4, "testcount2": 1}),
			},
		},
		{
			name:    "Notfound status#1",
			arg:     *stor,
			request: "/update/gauge/testguage1",
			want: want{
				code:        404,
				contentType: "text/plain",
				storage:     *repositories.NewMemStorage(map[string]float64{"testgauge1": 3, "testgauge2": 10}, map[string]int64{"testcount1": 4, "testcount2": 1}),
			},
		},
		{
			name:    "Counter errort#4",
			arg:     *stor,
			request: "/update/gauge/alloc/233184",
			want: want{
				code:        200,
				contentType: "text/plain",
				storage:     *repositories.NewMemStorage(map[string]float64{"testgauge1": 3, "testgauge2": 10, "alloc": 233184}, map[string]int64{"testcount1": 4, "testcount2": 1}),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Post("/update/{metricType}/{metricName}/{metricValue}", func(res http.ResponseWriter, req *http.Request) {
				HandlerUpdate(res, req, &tt.arg)
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
}
