package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/go-chi/chi/v5"
	"github.com/go-test/deep"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/rand"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/repositories"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/repositories/mocks"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/saver"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/storage"
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

type errorWriter struct {
	header     http.Header
	StatusCode int
}

func (e *errorWriter) Write(_ []byte) (int, error) {
	return 0, errors.New("simulated connection error")
}

func (e *errorWriter) Header() http.Header {
	return e.header
}

func (e *errorWriter) WriteHeader(statusCode int) {
	e.StatusCode = statusCode
}

func TestGetGlobal(t *testing.T) {
	{
		normalizeHTML := func(html string) string {
			return strings.Join(strings.Fields(html), " ")
		}

		// создаём контроллер
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		type contextKey string

		m := mocks.NewMockMetricsReader(ctrl)

		// successfull test
		k1 := contextKey("success")
		v1 := "1"
		ctx1 := context.WithValue(context.Background(), k1, v1)

		metrics := "metrcis_type_first: value_first\nmetrcis_type_second: value_second\ncounter: 1"
		m.EXPECT().GetAllMetrics(ctx1).Return(metrics, nil)
		// Проверяем тело ответа
		expectedBody := `<!DOCTYPE html>
	<html lang="en">
	<head>
		<meta charset="UTF-8">
		<title>HTML Response</title>
	</head>
	<body>
		<pre>metrcis_type_first: value_first
	metrcis_type_second: value_second
	counter: 1</pre>
	</body>
	</html>`

		// failure test
		k2 := contextKey("failure")
		v2 := "2"
		ctx2 := context.WithValue(context.Background(), k2, v2)
		m.EXPECT().GetAllMetrics(ctx2).Return("", fmt.Errorf("something was wrong"))

		// failure test2
		k3 := contextKey("failure2")
		v3 := "3"
		ctx3 := context.WithValue(context.Background(), k3, v3)
		m.EXPECT().GetAllMetrics(ctx3).Return("", fmt.Errorf("something was wrong"))

		tests := []struct {
			name           string
			ctx            context.Context
			wantBody       string
			statusCode     int
			wantChangeTmpl bool
		}{
			{
				name:           "successfull get",
				ctx:            ctx1,
				wantBody:       strings.TrimSpace(expectedBody),
				statusCode:     200,
				wantChangeTmpl: false,
			},
			{
				name:           "failure get",
				ctx:            ctx2,
				wantBody:       "",
				statusCode:     500,
				wantChangeTmpl: false,
			},
			{
				name:           "failure get",
				ctx:            ctx3,
				wantBody:       strings.TrimSpace(expectedBody),
				statusCode:     500,
				wantChangeTmpl: true,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				r := chi.NewRouter()
				r.Get("/test", func(res http.ResponseWriter, req *http.Request) {
					req = req.WithContext(tt.ctx)
					GetGlobal(res, req, m)
				})

				request := httptest.NewRequest(http.MethodGet, "/test", nil)

				originalTmpl := tmpl
				if tt.wantChangeTmpl {
					tmpl = template.Must(template.New("test").Parse(`{{ .InvalidField }}`))
				}
				defer func() { tmpl = originalTmpl }()

				w := httptest.NewRecorder()
				r.ServeHTTP(w, request)

				res := w.Result()
				defer res.Body.Close() // Закрываем тело ответа
				// проверяем код ответа
				assert.Equal(t, tt.statusCode, res.StatusCode)

				if tt.statusCode == 200 {
					assert.Contains(t, res.Header.Get("Content-Type"), "text/html")

					getBody, err := io.ReadAll(res.Body)
					require.NoError(t, err)
					defer res.Body.Close()

					// Сравниваем обработанные документы
					assert.Equal(t, normalizeHTML(tt.wantBody), normalizeHTML(string(getBody)))
				} else {
					assert.Contains(t, res.Header.Get("Content-Type"), "text/plain")
				}
			})
		}
	}
	{
		// создаём контроллер
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		m := mocks.NewMockMetricsReader(ctrl)
		m.EXPECT().GetAllMetrics(gomock.Any()).Return("metrics", nil)

		errorWriter := errorWriter{
			header: make(http.Header, 0),
		}

		r := chi.NewRouter()
		r.Get("/test", func(_ http.ResponseWriter, req *http.Request) {
			GetGlobal(&errorWriter, req, m)
		})

		request := httptest.NewRequest(http.MethodGet, "/test", nil)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, request)

		res := w.Result()
		defer res.Body.Close() // Закрываем тело ответа
		// проверяем код ответа
		assert.Equal(t, 500, errorWriter.StatusCode)
	}
}

func TestPingDatabase(t *testing.T) {
	// Подключение к базе данных
	flagDatabaseDsn := "host=localhost user=benchmarkmetrics password=password dbname=benchmarkmetrics sslmode=disable"
	db, err := sql.Open("pgx", flagDatabaseDsn)
	require.NoError(t, err)
	defer db.Close()

	failureDB, _ := sql.Open("pgx", "host=wronghost user=benchmarkmetrics password=WRONGpassword dbname=benchmarkmetrics sslmode=disable")

	tests := []struct {
		name       string
		db         *sql.DB
		statusCode int
	}{
		{
			name:       "successfull ping",
			db:         db,
			statusCode: 200,
		},
		{
			name:       "failure ping",
			db:         failureDB,
			statusCode: 500,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Get("/test", PingDatabaseHandler(tt.db))

			request := httptest.NewRequest(http.MethodGet, "/test", nil)

			w := httptest.NewRecorder()
			r.ServeHTTP(w, request)

			res := w.Result()
			defer res.Body.Close() // Закрываем тело ответа
			// проверяем код ответа
			assert.Equal(t, tt.statusCode, res.StatusCode)
		})
	}
}

func TestGetMetricJSON(t *testing.T) {
	{
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
			metrics     repositories.Metric
		}
		tests := []struct {
			name    string
			request string
			body    repositories.Metric
			want    want
		}{
			{
				name:    "Counter testcount#1",
				request: "/value/",
				body: repositories.Metric{
					ID:    "testcount1",
					MType: "counter",
				},
				want: want{
					code:        200,
					contentType: "application/json",
					metrics: repositories.Metric{
						ID:    "testcount1",
						MType: "counter",
						Delta: delta(4),
					},
				},
			},
			{
				name:    "Counter error#1",
				request: "/value/",
				body: repositories.Metric{
					ID:    "testcount3",
					MType: "counter",
				},
				want: want{
					code:        404,
					contentType: "application/json",
					metrics: repositories.Metric{
						ID:    "testcount3",
						MType: "counter",
					},
				},
			},
			{
				name:    "Counter error#1",
				request: "/value/",
				body: repositories.Metric{
					ID:    "testcount2",
					MType: "couunter",
				},
				want: want{
					code:        404,
					contentType: "application/json",
					metrics: repositories.Metric{
						ID:    "testcount2",
						MType: "couunter",
					},
				},
			},
			{
				name:    "Gauge testgauge#1",
				request: "/value/",
				body: repositories.Metric{
					ID:    "testgauge1",
					MType: "gauge",
				},
				want: want{
					code:        200,
					contentType: "application/json",
					metrics: repositories.Metric{
						ID:    "testgauge1",
						MType: "gauge",
						Value: value(3.134),
					},
				},
			},
			{
				name:    "Gauge error#1",
				request: "/value",
				body: repositories.Metric{
					ID:    "testgauge3",
					MType: "gauge",
				},
				want: want{
					code:        404,
					contentType: "application/json",
					metrics: repositories.Metric{
						ID:    "testgauge3",
						MType: "gauge",
					},
				},
			},
			{
				name:    "Gauge error#2",
				request: "/value",
				body: repositories.Metric{
					ID:    "testgauge2",
					MType: "gauuge",
				},
				want: want{
					code:        404,
					contentType: "application/json",
					metrics: repositories.Metric{
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
					var resMetric repositories.Metric
					dec := json.NewDecoder(res.Body)
					er := dec.Decode(&resMetric)
					require.NoError(t, er)

					assert.Equal(t, tt.want.metrics, resMetric)
				}
			})
		}
	}

	// Тест с проверкой загрузки метрик из файла при инициализации
	{
		stor := storage.NewMemStorage(map[string]float64{"testgauge1": 3.189, "testgauge2": 10, "alloc": 233184}, map[string]int64{"testcount1": 4, "testcount2": 1})

		delta := func(d int64) *int64 {
			return &d
		}
		value := func(v float64) *float64 {
			return &v
		}

		nameTestFile := "./TestGetMetricJSON_2.json"
		saverVar, err := saver.NewWriter(nameTestFile)
		require.NoError(t, err)

		m0 := repositories.Metric{
			ID:    "testGauge0",
			MType: "gauge",
			Value: value(111.11),
		}
		m1 := repositories.Metric{
			ID:    "testGauge1",
			MType: "gauge",
			Value: value(1234.124),
		}
		m2 := repositories.Metric{
			ID:    "testCounter1",
			MType: "counter",
			Delta: delta(30),
		}

		// Записываю метрики в файл, для загрузки в сервер при его инициализации
		storForFluahFile := storage.NewDefaultMemStorage()
		metrcSlice := []repositories.Metric{
			m0,
			m1,
			m2,
		}
		errWrite := storForFluahFile.AddMetricsFromSlice(context.Background(), metrcSlice)
		require.NoError(t, errWrite)

		errFlush := saverVar.WriteMetrics(storForFluahFile)
		require.NoError(t, errFlush)

		reader, erReader := saver.NewReader(nameTestFile)
		require.NoError(t, erReader)

		saver.SetRestore(true)
		err = saver.AddMetricsFromFile(stor, reader)
		require.NoError(t, err)

		type want struct {
			code        int
			contentType string
			metrics     repositories.Metric
		}
		tests := []struct {
			name    string
			request string
			body    repositories.Metric
			want    want
		}{
			{
				name:    "Test gauge#0",
				request: "/value/",
				body: repositories.Metric{
					ID:    "testGauge0",
					MType: "gauge",
				},
				want: want{
					code:        200,
					contentType: "application/json",
					metrics: repositories.Metric{
						ID:    "testGauge0",
						MType: "gauge",
						Value: value(111.11),
					},
				},
			},
			{
				name:    "Counter testcount#1",
				request: "/value/",
				body: repositories.Metric{
					ID:    "testCounter1",
					MType: "counter",
				},
				want: want{
					code:        200,
					contentType: "application/json",
					metrics: repositories.Metric{
						ID:    "testCounter1",
						MType: "counter",
						Delta: delta(30),
					},
				},
			},
			{
				name:    "Test gauge#1",
				request: "/value/",
				body: repositories.Metric{
					ID:    "testGauge1",
					MType: "gauge",
				},
				want: want{
					code:        200,
					contentType: "application/json",
					metrics: repositories.Metric{
						ID:    "testGauge1",
						MType: "gauge",
						Value: value(1234.124),
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
					var resMetric repositories.Metric
					dec := json.NewDecoder(res.Body)
					er := dec.Decode(&resMetric)
					require.NoError(t, er)

					assert.Equal(t, tt.want.metrics.ID, resMetric.ID)
					if tt.want.metrics.MType == "counter" {
						assert.NotEqual(t, nil, tt.want.metrics.Delta)
						assert.NotEqual(t, nil, resMetric.Delta)
						assert.Equal(t, *tt.want.metrics.Delta, *resMetric.Delta)
					} else {
						assert.NotEqual(t, nil, tt.want.metrics.Value)
						assert.NotEqual(t, nil, resMetric.Value)
						assert.Equal(t, *tt.want.metrics.Value, *resMetric.Value)
					}
					assert.Equal(t, tt.want.metrics.MType, resMetric.MType)
					assert.Equal(t, tt.want.metrics, resMetric)
				}
			})
		}
		// Удаляю тестовый файл
		er := os.Remove(nameTestFile)
		require.NoError(t, er)
	}
	// Тесты с невалидными метриками
	{
		// Мокирую хранилище метрик
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		m := mocks.NewMockMetricsReader(ctrl)

		// Тест с невалидной counter метрикой--------------------------------
		conter1 := repositories.Metric{
			MType: "counter",
			ID:    "invalid conter",
		}
		m.EXPECT().GetMetric(gomock.Any(), "counter", "invalid conter").Return("invalid counter delta", nil)

		// Тест с невалидной gauge метрикой--------------------------------
		gauge1 := repositories.Metric{
			MType: "gauge",
			ID:    "invalid gauge",
		}
		m.EXPECT().GetMetric(gomock.Any(), "gauge", "invalid gauge").Return("invalid gauge value", nil)

		// Тест с метрикой с невалидным типом --------------------------------
		wrongType := repositories.Metric{
			MType: "wrong type",
			ID:    "invalid type m",
		}
		m.EXPECT().GetMetric(gomock.Any(), "wrong type", "invalid type m").Return("invalid type", nil)

		tests := []struct {
			name string
			body repositories.Metric
			code int
		}{
			{
				name: "Test wrong counter",
				body: conter1,
				code: 500,
			},
			{
				name: "Test wrong gauge",
				body: gauge1,
				code: 500,
			},
			{
				name: "Test wrong type",
				body: wrongType,
				code: 400,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				r := chi.NewRouter()
				r.Post("/value/", func(res http.ResponseWriter, req *http.Request) {
					GetMetricJSON(res, req, m)
				})

				// сериализую струтктуру с метриками в json
				body, err := json.Marshal(tt.body)
				require.NoError(t, err)

				request := httptest.NewRequest(http.MethodPost, "/value/", bytes.NewBuffer(body))
				w := httptest.NewRecorder()
				r.ServeHTTP(w, request)

				res := w.Result()
				defer res.Body.Close() // Закрываем тело ответа
				// проверяем код ответа
				assert.Equal(t, tt.code, res.StatusCode)
			})
		}
	}
}

func TestUpdateMetricsBatch(t *testing.T) {
	tests := []struct {
		name string
		body []byte
		stor repositories.MetricsWriter
		code int
	}{
		{
			name: "Storage is nil",
			body: nil,
			stor: nil,
			code: 500,
		},
		{
			name: "Invalid body",
			body: []byte("wrong body"),
			stor: storage.NewDefaultMemStorage(),
			code: 500,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Post("/", func(res http.ResponseWriter, req *http.Request) {
				UpdateMetricsBatch(res, req, tt.stor)
			})

			request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(tt.body))
			w := httptest.NewRecorder()
			r.ServeHTTP(w, request)

			res := w.Result()
			defer res.Body.Close() // Закрываем тело ответа
			// проверяем код ответа
			assert.Equal(t, tt.code, res.StatusCode)
		})
	}
}

type MockResponseWriter struct {
	HeaderMap http.Header
	Body      []byte
	Status    int
}

func (m *MockResponseWriter) Header() http.Header {
	if m.HeaderMap == nil {
		m.HeaderMap = make(http.Header)
	}
	return m.HeaderMap
}

func (m *MockResponseWriter) Write(body []byte) (int, error) {
	m.Body = body
	return len(body), nil
}

func (m *MockResponseWriter) WriteHeader(statusCode int) {
	m.Status = statusCode
}

func (m *MockResponseWriter) GetStatus() int {
	return m.Status
}

func TestGetMetric(t *testing.T) {

	// successful test#1
	// Создаем тестовый запрос
	req := &http.Request{
		URL: &url.URL{},
	}

	// Добавляем параметры маршрута в контекст
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("metricType", "counter")
	rctx.URLParams.Add("metricName", "counter_1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	metricName := "counter_1"
	metricValue := int64(2352)

	type request struct {
		res     http.ResponseWriter
		req     *http.Request
		storage repositories.MetricsReader
	}
	type want struct {
		res *MockResponseWriter
	}
	tests := []struct {
		name    string
		request request
		want    want
	}{
		{
			name: "successful test#1",
			request: request{
				res:     &MockResponseWriter{},
				req:     req,
				storage: storage.NewMemStorage(map[string]float64{}, map[string]int64{metricName: metricValue}),
			},
			want: want{
				res: &MockResponseWriter{
					Status: 0,
					Body:   []byte(strconv.FormatInt(metricValue, 10)),
				},
			},
		},
		{
			name: "metric is not contain in storage",
			request: request{
				res:     &MockResponseWriter{},
				req:     req,
				storage: storage.NewDefaultMemStorage(),
			},
			want: want{
				res: &MockResponseWriter{
					Status: 404,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			GetMetric(tt.request.res, tt.request.req, tt.request.storage)

			response := tt.request.res.(*MockResponseWriter)

			assert.Equal(t, tt.want.res.GetStatus(), response.GetStatus())

			if c := tt.want.res.GetStatus(); c == 0 {
				header := response.Header()
				assert.Equal(t, "200", header.Get("Status-Code"))
				assert.Equal(t, len(tt.want.res.Body), len(response.Body))
				assert.Equal(t, string(tt.want.res.Body), string(response.Body))
			}
		})
	}
}

func TestUpdateMetrics(t *testing.T) {
	stor := storage.NewMemStorage(nil, map[string]int64{"testcount1": 1})

	type want struct {
		code        int
		contentType string
		storage     *storage.MemStorage
	}
	tests := []struct {
		name    string
		request string
		want    want
	}{
		{
			name:    "Counter testcount#1",
			request: "/update/counter/testcount1/3",
			want: want{
				code:        200,
				contentType: "text/plain",
				storage:     storage.NewMemStorage(nil, map[string]int64{"testcount1": 4}),
			},
		},
		{
			name:    "Counter testcount#2",
			request: "/update/counter/testcount2/1",
			want: want{
				code:        200,
				contentType: "text/plain",
				storage:     storage.NewMemStorage(nil, map[string]int64{"testcount1": 4, "testcount2": 1}),
			},
		},
		{
			name:    "Counter testguage#1",
			request: "/update/gauge/testgauge1/1",
			want: want{
				code:        200,
				contentType: "text/plain",
				storage:     storage.NewMemStorage(map[string]float64{"testgauge1": 1}, map[string]int64{"testcount1": 4, "testcount2": 1}),
			},
		},
		{
			name:    "Counter testguage#2",
			request: "/update/gauge/testgauge1/3",
			want: want{
				code:        200,
				contentType: "text/plain",
				storage:     storage.NewMemStorage(map[string]float64{"testgauge1": 3}, map[string]int64{"testcount1": 4, "testcount2": 1}),
			},
		},
		{
			name:    "Counter testguage#3",
			request: "/update/gauge/testgauge2/10",
			want: want{
				code:        200,
				contentType: "text/plain",
				storage:     storage.NewMemStorage(map[string]float64{"testgauge1": 3, "testgauge2": 10}, map[string]int64{"testcount1": 4, "testcount2": 1}),
			},
		},
		{
			name:    "Counter errort#1",
			request: "/update/counter/testcount1/aaaaa",
			want: want{
				code:        400,
				contentType: "text/plain",
				storage:     storage.NewMemStorage(map[string]float64{"testgauge1": 3, "testgauge2": 10}, map[string]int64{"testcount1": 4, "testcount2": 1}),
			},
		},
		{
			name:    "Counter errort#2",
			request: "/update/counter/testcount1/",
			want: want{
				code:        404,
				contentType: "text/plain",
				storage:     storage.NewMemStorage(map[string]float64{"testgauge1": 3, "testgauge2": 10}, map[string]int64{"testcount1": 4, "testcount2": 1}),
			},
		},
		{
			name:    "Counter errort#3",
			request: "/update/counter/testcount1/1.12",
			want: want{
				code:        400,
				contentType: "text/plain",
				storage:     storage.NewMemStorage(map[string]float64{"testgauge1": 3, "testgauge2": 10}, map[string]int64{"testcount1": 4, "testcount2": 1}),
			},
		},
		{
			name:    "Guage errort#1",
			request: "/update/gauge/testguage1/aaaaa",
			want: want{
				code:        400,
				contentType: "text/plain",
				storage:     storage.NewMemStorage(map[string]float64{"testgauge1": 3, "testgauge2": 10}, map[string]int64{"testcount1": 4, "testcount2": 1}),
			},
		},
		{
			name:    "Guage errort#2",
			request: "/update/gauge/testguage1/",
			want: want{
				code:        404,
				contentType: "text/plain",
				storage:     storage.NewMemStorage(map[string]float64{"testgauge1": 3, "testgauge2": 10}, map[string]int64{"testcount1": 4, "testcount2": 1}),
			},
		},
		{
			name:    "BadRequest status#1",
			request: "/update/gauges/testguage1/aaaaa",
			want: want{
				code:        400,
				contentType: "text/plain",
				storage:     storage.NewMemStorage(map[string]float64{"testgauge1": 3, "testgauge2": 10}, map[string]int64{"testcount1": 4, "testcount2": 1}),
			},
		},
		{
			name:    "Notfound status#1",
			request: "/update/gauge/testguage1",
			want: want{
				code:        404,
				contentType: "text/plain",
				storage:     storage.NewMemStorage(map[string]float64{"testgauge1": 3, "testgauge2": 10}, map[string]int64{"testcount1": 4, "testcount2": 1}),
			},
		},
		{
			name:    "Counter errort#4",
			request: "/update/gauge/alloc/233184",
			want: want{
				code:        200,
				contentType: "text/plain",
				storage:     storage.NewMemStorage(map[string]float64{"testgauge1": 3, "testgauge2": 10, "alloc": 233184}, map[string]int64{"testcount1": 4, "testcount2": 1}),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Post("/update/{metricType}/{metricName}/{metricValue}", func(res http.ResponseWriter, req *http.Request) {
				UpdateMetrics(res, req, stor)
			})

			request := httptest.NewRequest(http.MethodPost, tt.request, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, request)

			res := w.Result()
			defer res.Body.Close() // Закрываем тело ответа
			// проверяем код ответа
			assert.Equal(t, tt.want.code, res.StatusCode)

			if tt.want.code == 200 {
				wantAllSlice, errWantSlice := tt.want.storage.GetAllMetricsSlice(context.Background())
				require.NoError(t, errWantSlice)
				getAllSlice, errGetSlice := stor.GetAllMetricsSlice(context.Background())
				require.NoError(t, errGetSlice)
				deep.Equal(wantAllSlice, getAllSlice)
			}
		})
	}
	// Storage is nil test
	{
		r := chi.NewRouter()
		r.Post("/update/{metricType}/{metricName}/{metricValue}", func(res http.ResponseWriter, req *http.Request) {
			UpdateMetrics(res, req, nil)
		})

		request := httptest.NewRequest(http.MethodPost, "/update/gauge/alloc/233184", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, request)

		res := w.Result()
		defer res.Body.Close() // Закрываем тело ответа
		// проверяем код ответа
		assert.Equal(t, 500, res.StatusCode)
	}
}

func TestUpdateMetricsJSON(t *testing.T) {
	{
		stor := storage.NewMemStorage(nil, map[string]int64{"testcount1": 1})

		delta := func(d int64) *int64 {
			return &d
		}
		value := func(v float64) *float64 {
			return &v
		}
		type want struct {
			code        int
			contentType string
			storage     *storage.MemStorage
		}
		tests := []struct {
			name    string
			request string
			body    repositories.Metric
			want    want
		}{
			{
				name:    "Counter testcount#1",
				request: "/update",
				body: repositories.Metric{
					ID:    "testcount1",
					MType: "counter",
					Delta: delta(3),
				},
				want: want{
					code:        200,
					contentType: "application/json",
					storage:     storage.NewMemStorage(nil, map[string]int64{"testcount1": 4}),
				},
			},
			{
				name:    "Counter testcount#2",
				request: "/update",
				body: repositories.Metric{
					ID:    "testcount2",
					MType: "counter",
					Delta: delta(1),
				},
				want: want{
					code:        200,
					contentType: "application/json",
					storage:     storage.NewMemStorage(nil, map[string]int64{"testcount1": 4, "testcount2": 1}),
				},
			},
			{
				name:    "Counter testguage#1",
				request: "/update",
				body: repositories.Metric{
					ID:    "testgauge1",
					MType: "gauge",
					Value: value(1),
				},
				want: want{
					code:        200,
					contentType: "application/json",
					storage:     storage.NewMemStorage(map[string]float64{"testgauge1": 1}, map[string]int64{"testcount1": 4, "testcount2": 1}),
				},
			},
			{
				name:    "Counter testguage#2",
				request: "/update",
				body: repositories.Metric{
					ID:    "testgauge1",
					MType: "gauge",
					Value: value(3),
				},
				want: want{
					code:        200,
					contentType: "application/json",
					storage:     storage.NewMemStorage(map[string]float64{"testgauge1": 3}, map[string]int64{"testcount1": 4, "testcount2": 1}),
				},
			},
			{
				name:    "Counter testguage#3",
				request: "/update",
				body: repositories.Metric{
					ID:    "testgauge2",
					MType: "gauge",
					Value: value(10),
				},
				want: want{
					code:        200,
					contentType: "application/json",
					storage:     storage.NewMemStorage(map[string]float64{"testgauge1": 3, "testgauge2": 10}, map[string]int64{"testcount1": 4, "testcount2": 1}),
				},
			},
			{
				name:    "Counter errort#1",
				request: "/update",
				body: repositories.Metric{
					ID:    "testcount1",
					MType: "counteer",
					Delta: delta(10),
				},
				want: want{
					code:        400,
					contentType: "application/json",
					storage:     storage.NewMemStorage(map[string]float64{"testgauge1": 3, "testgauge2": 10}, map[string]int64{"testcount1": 4, "testcount2": 1}),
				},
			},
			{
				name:    "Counter errort#2",
				request: "/update",
				body: repositories.Metric{
					ID:    "testcount1",
					MType: "counter",
				},
				want: want{
					code:        400,
					contentType: "application/json",
					storage:     storage.NewDefaultMemStorage(),
				},
			},
			{
				name:    "Guage errort#1",
				request: "/update",
				body: repositories.Metric{
					ID:    "testguage1",
					MType: "gauuge",
					Value: value(10),
				},
				want: want{
					code:        400,
					contentType: "application/json",
					storage:     storage.NewMemStorage(map[string]float64{"testgauge1": 3, "testgauge2": 10}, map[string]int64{"testcount1": 4, "testcount2": 1}),
				},
			},
			{
				name:    "Guage errort#2",
				request: "/update",
				body: repositories.Metric{
					ID:    "testguage1",
					MType: "gauge",
				},
				want: want{
					code:        400,
					contentType: "application/json",
					storage:     storage.NewDefaultMemStorage(),
				},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				r := chi.NewRouter()
				r.Post("/update", func(res http.ResponseWriter, req *http.Request) {
					UpdateMetricsJSON(res, req, stor)
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

				wantAllSlice, errWantSlice := tt.want.storage.GetAllMetricsSlice(context.Background())
				require.NoError(t, errWantSlice)
				getAllSlice, errGetSlice := stor.GetAllMetricsSlice(context.Background())
				require.NoError(t, errGetSlice)
				deep.Equal(wantAllSlice, getAllSlice)

				// Проверяю тело ответа, если код ответа 200
				if res.StatusCode == http.StatusOK {
					// Десериализую структуру с метриками
					var resMetric repositories.Metric
					dec := json.NewDecoder(res.Body)
					er := dec.Decode(&resMetric)
					require.NoError(t, er)

					assert.Equal(t, tt.body, resMetric)
				}
			})
		}
	}
	// Тесты с невалидными данными
	{
		tests := []struct {
			name string
			body []byte
			stor repositories.MetricsWriter
			code int
		}{
			{
				name: "Storage is nil",
				body: nil,
				stor: nil,
				code: 500,
			},
			{
				name: "Invalid body",
				body: []byte("wrong body"),
				stor: storage.NewDefaultMemStorage(),
				code: 500,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				r := chi.NewRouter()
				r.Post("/", func(res http.ResponseWriter, req *http.Request) {
					UpdateMetricsJSON(res, req, tt.stor)
				})

				request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(tt.body))
				w := httptest.NewRecorder()
				r.ServeHTTP(w, request)

				res := w.Result()
				defer res.Body.Close() // Закрываем тело ответа
				// проверяем код ответа
				assert.Equal(t, tt.code, res.StatusCode)
			})
		}
	}
}

func BenchmarkUpdateMetricsJSON(b *testing.B) {
	// В качестве хранилища использую оперативную память.
	// В данной конфигурации автотестов использовать в качестве хранилища не представляется возможным,
	// так как нет корректного dsn адреса для запуска бд.
	stor := storage.NewDefaultMemStorage()
	ctx := context.Background()

	// Генерация метрик для заполения сервиса-----------------------------------------------------
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	// Функция для генерации случайного имени метрики
	randomMetricName := func(n int) string {
		sb := strings.Builder{}
		sb.Grow(n)
		for i := 0; i < n; i++ {
			sb.WriteByte(letterBytes[rand.Intn(len(letterBytes))])
		}
		return sb.String()
	}
	// Функция для генерации случайного значения метрики
	randomGaugeValue := func() float64 {
		return -1000000 + rand.Float64()*(2000000) // от -1000000 до 1000000
	}
	randomCounterValue := func() int64 {
		return -1000000 + rand.Int63()*(2000000) // от -1000000 до 1000000
	}
	// Функция для случайного выбора типа метрики (gauge или counter)
	randomMetricType := func() string {
		types := []string{"gauge", "counter"}
		return types[rand.Intn(len(types))]
	}

	metricsNumber := 10000
	metrics := make([][]byte, 0, metricsNumber)
	for i := 0; i < metricsNumber; i++ {
		var metric repositories.Metric

		lenName := 5 + rand.Intn(5)
		name := randomMetricName(lenName)
		typeMetric := randomMetricType()
		switch typeMetric {
		case "gauge":
			value := randomGaugeValue()
			metric = repositories.Metric{
				ID:    name,
				MType: "gauge",
				Value: &value,
			}
		case "counter":
			value := randomCounterValue()
			metric = repositories.Metric{
				ID:    name,
				MType: "counter",
				Delta: &value,
			}
		}

		// сериализую струтктуру с метриками в json
		body, err := json.Marshal(metric)
		if err != nil {
			b.Error(err, "Marshall message error")
		}
		// заполняю слайс с сериализованными метриками
		metrics = append(metrics, body)
	}
	// -------------------------------------------------------------------------------------

	// Сбрасываю счетчик
	b.ResetTimer()

	// запускаю бенчмарк
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		stor.Clean(ctx)
		b.StartTimer()

		for _, m := range metrics {
			// не учитываю подготовительные операции -----------------
			b.StopTimer()
			r := chi.NewRouter()
			r.Post("/update", func(res http.ResponseWriter, req *http.Request) {
				UpdateMetricsJSON(res, req, stor)
			})
			request := httptest.NewRequest(http.MethodPost, "/update", bytes.NewBuffer(m))
			w := httptest.NewRecorder()
			b.StartTimer() //---------------------------------------------

			r.ServeHTTP(w, request)

			// проверка кода ответа сервера
			b.StopTimer()
			res := w.Result()
			defer res.Body.Close() // Закрываем тело ответа
			// проверяем код ответа
			assert.Equal(b, 200, res.StatusCode)
			b.StartTimer()
		}
	}
}
