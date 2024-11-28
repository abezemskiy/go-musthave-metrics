package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	_ "net/http/pprof" // подключаем пакет pprof
	"testing"

	"github.com/go-test/deep"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/storage"
)

func testRequest(t *testing.T, ts *httptest.Server, method, path string) *http.Response {
	req, err := http.NewRequest(method, ts.URL+path, nil)
	require.NoError(t, err)

	resp, err := ts.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	return resp
}

func TestHandlerUpdate(t *testing.T) {
	stor := storage.NewMemStorage(nil, map[string]int64{"testcount1": 1})

	ts := httptest.NewServer(MetricRouter(stor, nil))

	defer ts.Close()

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
		resp := testRequest(t, ts, "POST", tt.request)
		assert.Equal(t, tt.want.code, resp.StatusCode)

		wantAllSlice, errWantSlice := tt.want.storage.GetAllMetricsSlice(context.Background())
		require.NoError(t, errWantSlice)
		getAllSlice, errGetSlice := stor.GetAllMetricsSlice(context.Background())
		require.NoError(t, errGetSlice)
		deep.Equal(wantAllSlice, getAllSlice)
		resp.Body.Close()
	}
}

// func TestMusthaveMetrics(t *testing.T) {
// 	// Функция для очистки данных в базе
// 	cleanBD := func(dsn string) {
// 		// очищаю данные в тестовой бд------------------------------------------------------
// 		// создаём соединение с СУБД PostgreSQL
// 		conn, err := sql.Open("pgx", dsn)
// 		require.NoError(t, err)
// 		defer conn.Close()

// 		// Проверка соединения с БД
// 		ctx := context.Background()
// 		err = conn.PingContext(ctx)
// 		require.NoError(t, err)

// 		// создаем экземпляр хранилища pg
// 		stor := pg.NewStore(conn)
// 		err = stor.Bootstrap(ctx)
// 		require.NoError(t, err)
// 		err = stor.Disable(ctx)
// 		require.NoError(t, err)
// 	}

// 	// функция для получения свободного порта для запуска приложений
// 	getFreePort := func() (int, error) {
// 		// Слушаем на порту 0, чтобы операционная система выбрала свободный порт
// 		listener, err := net.Listen("tcp", ":0")
// 		if err != nil {
// 			return 0, err
// 		}
// 		defer listener.Close()

// 		// Получаем назначенный системой порт
// 		port := listener.Addr().(*net.TCPAddr).Port
// 		return port, nil
// 	}

// 	// функция для очистки файлов с ключами
// 	removeFile := func(file string) {
// 		err := os.Remove(file)
// 		require.NoError(t, err)
// 	}

// 	// генирирую ключи для ассиметричного шифрования
// 	pathKeys := "."
// 	err := encryption.GenerateKeys(pathKeys)
// 	require.NoError(t, err)

// 	key := "secret key"
// 	var done = make(chan struct{})
// 	var wg sync.WaitGroup
// 	// Определяю параметры для запуска сервера
// 	serverPort, err := getFreePort()
// 	require.NoError(t, err)
// 	serverAdress := fmt.Sprintf(":%d", serverPort)
// 	databaseDsn := "host=localhost user=benchmarkmetrics password=password dbname=benchmarkmetrics sslmode=disable"

// 	// Очищаю данные в БД от предыдущих запусков
// 	cleanBD(databaseDsn)

// 	// Запускаю server-----------------------------------------------------
// 	cmdServer := exec.Command("./server", fmt.Sprintf("-a=%s", serverAdress),
// 		fmt.Sprintf("-k=%s", key), fmt.Sprintf("-d=%s", databaseDsn), "-l=info", fmt.Sprintf("-crypto-key=%s", pathKeys+"/private_key.pem"))
// 	// Связываем стандартный вывод и ошибки программы с выводом программы Go
// 	cmdServer.Stdout = log.Writer()
// 	cmdServer.Stderr = log.Writer()

// 	// // Функция остановки сервера
// 	stopServer := func() {
// 		err = cmdServer.Process.Signal(os.Interrupt) // Посылаем сигнал прерывания
// 		require.NoError(t, err)
// 	}

// 	// Функция запуска сервера
// 	startServer := func(w *sync.WaitGroup) {
// 		// Запуск программы
// 		err = cmdServer.Start()
// 		require.NoError(t, err)

// 		<-done
// 		stopServer()
// 		w.Done()
// 	}

// 	// Запускаю сервис агента--------------------------------------------------
// 	agentPort, err := getFreePort()
// 	require.NoError(t, err)
// 	agentAdress := fmt.Sprintf("localhost:%d", agentPort)
// 	cmdAgent := exec.Command("./../agent/agent", fmt.Sprintf("-a=%s", agentAdress), fmt.Sprintf("-k=%s", key), fmt.Sprintf("-r=%d", 5),
// 		fmt.Sprintf("-crypto-key=%s", pathKeys+"/public_key.pem"))
// 	// Связываем стандартный вывод и ошибки программы с выводом программы Go
// 	cmdAgent.Stdout = log.Writer()
// 	cmdAgent.Stderr = log.Writer()

// 	// Функция остановки агента
// 	stopAgent := func() {
// 		err = cmdAgent.Process.Signal(os.Interrupt) // Посылаем сигнал прерывания
// 		require.NoError(t, err)
// 	}

// 	// Функция запуска агента
// 	startAgent := func(w *sync.WaitGroup) {
// 		err = cmdAgent.Start()
// 		require.NoError(t, err)

// 		<-done
// 		stopAgent()
// 		w.Done()
// 	}

// 	wg.Add(1)
// 	go startServer(&wg)
// 	wg.Add(1)
// 	go startAgent(&wg)
// 	time.Sleep(2 * time.Second) // Жду 2 секунды для запуска сервиса

// 	time.Sleep(5 * time.Minute) // Жду 2 минуты для сбора профиля работы сервиса
// 	close(done)
// 	wg.Wait()

// 	// создаём файл журнала профилирования памяти
// 	fmem, err := os.Create(`./../../profiles/result.pprof`)
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer fmem.Close()
// 	runtime.GC() // получаем статистику по использованию памяти
// 	if err := pprof.WriteHeapProfile(fmem); err != nil {
// 		panic(err)
// 	}

// 	defer removeFile(pathKeys + "/private_key.pem")
// 	defer removeFile(pathKeys + "/public_key.pem")
// }
