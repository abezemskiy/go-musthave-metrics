package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/go-chi/chi/v5"
	"github.com/go-test/deep"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/rand"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/repositories"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/pg"
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
		saver.AddMetricsFromFile(stor, reader)

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

			wantAllSlice, errWantSlice := tt.want.storage.GetAllMetricsSlice(context.Background())
			require.NoError(t, errWantSlice)
			getAllSlice, errGetSlice := stor.GetAllMetricsSlice(context.Background())
			require.NoError(t, errGetSlice)
			deep.Equal(wantAllSlice, getAllSlice)
		})
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
}

func BenchmarkUpdateMetricsJSON(b *testing.B) {
	// // Функция для очистки данных в базе
	cleanBD := func(dsn string) {
		// очищаю данные в тестовой бд------------------------------------------------------
		// создаём соединение с СУБД PostgreSQL
		conn, err := sql.Open("pgx", dsn)
		require.NoError(b, err)
		defer conn.Close()

		// Проверка соединения с БД
		ctx := context.Background()
		err = conn.PingContext(ctx)
		require.NoError(b, err)

		// создаем экземпляр хранилища pg
		stor := pg.NewStore(conn)
		err = stor.Bootstrap(ctx)
		require.NoError(b, err)
		err = stor.Disable(ctx)
		require.NoError(b, err)
		defer conn.Close()
	}

	databaseDsn := "host=localhost user=benchmarkmetrics password=password dbname=benchmarkmetrics sslmode=disable"

	// Запускаю базу данных для сохранения метрик------------------------------------------
	db, err := sql.Open("pgx", databaseDsn)
	if err != nil {
		log.Fatalf("Error create database connection for saving metrics : %v\n", err)
	}
	// Проверка соединения с БД
	ctx := context.Background()
	err = db.PingContext(ctx)
	if err != nil {
		log.Fatalf("Error checking connection with database: %v\n", err)
	}
	// создаем экземпляр хранилища pg
	stor := pg.NewStore(db)
	err = stor.Bootstrap(ctx)
	if err != nil {
		log.Fatalf("Error prepare database to work: %v\n", err)
	}
	defer db.Close()
	// -------------------------------------------------------------------------------------

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
		cleanBD(databaseDsn)
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
