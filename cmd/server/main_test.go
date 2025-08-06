package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	_ "net/http/pprof" // подключаем пакет pprof
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-test/deep"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/repositories"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/handlers"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/pg"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/saver"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/storage"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/tools/encryption"
)

func testRequest(t *testing.T, ts *httptest.Server, method, path string) *http.Response {
	req, err := http.NewRequest(method, ts.URL+path, nil)
	require.NoError(t, err)

	resp, err := ts.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	return resp
}

// Вспомогательная функция для получения значения метрики ответа сервера типа html
func getPollCount(netAddr string) (int64, error) {
	// Отправляю HTTP-запрос к серверу для получения списка метрик и их значений
	resp, err := http.Get("http://" + netAddr + "/")
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("status is not 200")
	}

	// Читаю содержимое ответа
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	response := string(body)

	// Извлекаю содержимое <pre>...</pre>
	start := strings.Index(response, "<pre>")
	end := strings.Index(response, "</pre>")
	if start == -1 || end == -1 {
		return 0, fmt.Errorf("Tag <pre> not found")
	}
	preContent := response[start+len("<pre>") : end]

	// Регулярное выражение для метрики PollCount
	re := regexp.MustCompile(`type:\s*counter,\s*name:\s*PollCount,\s*value:\s*([-+]?[0-9]*\.?[0-9]+)`)
	match := re.FindStringSubmatch(preContent)

	if len(match) < 2 {
		return 0, fmt.Errorf("PollCount metric not found")
	}

	// Преобразую значение метрики PollCount в int64
	pollCount, err := strconv.ParseInt(match[1], 10, 64)
	if err != nil {
		return 0, err
	}

	return pollCount, nil
}

func TestHandlerUpdate(t *testing.T) {
	// Вспомогательная функция для дережирования http запросов к серверу.
	metricRouter := func(stor repositories.IStorage) chi.Router {
		r := chi.NewRouter()

		r.Route("/", func(r chi.Router) {
			r.Route("/update", func(r chi.Router) {
				r.Post("/{metricType}/{metricName}/{metricValue}", handlers.UpdateMetricsHandler(stor))
			})
		})

		// Определяем маршрут по умолчанию для некорректных запросов
		r.NotFound(handlers.OtherRequestHandler())

		return r
	}

	stor := storage.NewMemStorage(nil, map[string]int64{"testcount1": 1})

	ts := httptest.NewServer(metricRouter(stor))

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

func TestRun(t *testing.T) {
	// Тест с ошибкой инициализации логера из-за невалидного флага уровня логирования
	{
		flagLogLevel = "wrong file"
		err := run(nil, nil, nil, 0)
		require.Error(t, err)
	}
	{
		// Тест с ошибкой запуска из-за неправильного пути к файлу хранения данных
		flagLogLevel = "info"
		saver.SetFilestoragePath("./wrong/path")
		err := run(nil, nil, nil, 1)
		require.Error(t, err)
	}
}

func TestMusthaveMetrics(t *testing.T) {
	// Функция для очистки данных в базе
	cleanBD := func(dsn string) {
		// очищаю данные в тестовой бд------------------------------------------------------
		// создаём соединение с СУБД PostgreSQL
		conn, err := sql.Open("pgx", dsn)
		require.NoError(t, err)
		defer conn.Close()

		// Проверка соединения с БД
		ctx := context.Background()
		err = conn.PingContext(ctx)
		require.NoError(t, err)

		// создаем экземпляр хранилища pg
		stor := pg.NewStore(conn)
		err = stor.Bootstrap(ctx)
		require.NoError(t, err)
		err = stor.Disable(ctx)
		require.NoError(t, err)
	}

	// функция для получения свободного порта для запуска приложений
	getFreePort := func() (int, error) {
		// Слушаем на порту 0, чтобы операционная система выбрала свободный порт
		listener, err := net.Listen("tcp", ":0")
		if err != nil {
			return 0, err
		}
		defer listener.Close()

		// Получаем назначенный системой порт
		port := listener.Addr().(*net.TCPAddr).Port
		return port, nil
	}

	// функция для очистки файлов с ключами
	removeDir := func(dir string) {
		err := os.RemoveAll(dir)
		require.NoError(t, err)
	}

	// генирирую ключи для ассиметричного шифрования
	pathKeys := "./http-keys"
	err := os.Mkdir(pathKeys, 0755) // 0755 - права доступа
	require.NoError(t, err)

	err = encryption.GenerateKeys(pathKeys)
	require.NoError(t, err)
	defer removeDir(pathKeys)

	key := "secret key"
	var done = make(chan struct{})
	var wg sync.WaitGroup
	// Определяю параметры для запуска сервера
	serverPort, err := getFreePort()
	require.NoError(t, err)
	serverAdress := fmt.Sprintf(":%d", serverPort)

	// Определяю параметры для запуска grpc сервера
	serverGRPCPort, err := getFreePort()
	require.NoError(t, err)
	serverGRPCAdress := fmt.Sprintf(":%d", serverGRPCPort)

	databaseDsn := "host=localhost user=benchmarkmetrics password=password dbname=benchmarkmetrics sslmode=disable"

	// Очищаю данные в БД от предыдущих запусков
	cleanBD(databaseDsn)
	// Очищаю данные в БД после завершения теста
	defer cleanBD(databaseDsn)

	// Запускаю server-----------------------------------------------------
	cmdServer := exec.Command("./server", fmt.Sprintf("-a=%s", serverAdress), fmt.Sprintf("-grpc-address=%s", serverGRPCAdress),
		fmt.Sprintf("-k=%s", key), fmt.Sprintf("-d=%s", databaseDsn), "-l=info", fmt.Sprintf("-crypto-key=%s", pathKeys+"/private_key.pem"))
	// Связываем стандартный вывод и ошибки программы с выводом программы Go
	cmdServer.Stdout = log.Writer()
	cmdServer.Stderr = log.Writer()

	// // Функция остановки сервера
	stopServer := func() {
		err = cmdServer.Process.Signal(os.Interrupt) // Посылаем сигнал прерывания
		require.NoError(t, err)
	}

	// Функция запуска сервера
	startServer := func(w *sync.WaitGroup) {
		// Запуск программы
		err = cmdServer.Start()
		require.NoError(t, err)

		<-done
		stopServer()
		w.Done()
	}

	// Запускаю сервис агента--------------------------------------------------
	agentAdress := serverAdress
	reportInterval := 10
	cmdAgent := exec.Command("./../agent/agent", fmt.Sprintf("-a=%s", agentAdress), fmt.Sprintf("-k=%s", key), fmt.Sprintf("-r=%d", reportInterval),
		fmt.Sprintf("-crypto-key=%s", pathKeys+"/public_key.pem"), fmt.Sprintf("-protocol=%s", "http"))
	// Связываем стандартный вывод и ошибки программы с выводом программы Go
	cmdAgent.Stdout = log.Writer()
	cmdAgent.Stderr = log.Writer()

	// Функция остановки агента
	stopAgent := func() {
		err = cmdAgent.Process.Signal(os.Interrupt) // Посылаем сигнал прерывания
		require.NoError(t, err)
	}

	// Функция запуска агента
	startAgent := func(w *sync.WaitGroup) {
		err = cmdAgent.Start()
		require.NoError(t, err)

		<-done
		stopAgent()
		w.Done()
	}

	wg.Add(1)
	go startServer(&wg)
	wg.Add(1)
	go startAgent(&wg)
	time.Sleep(2 * time.Second) // Жду 2 секунды для запуска сервиса

	// проверяю, что с каждой новой отправкой метрик на сервер счетчик PollCount увеличивается----------------------------------------
	// создаю для этой проверки собственный канал, чтобы завершить отправку запросов к серверу перед остановкой самого сервера.
	var checkerDone = make(chan struct{})
	wg.Add(1)
	oldPollCount := int64(0)
	go func(done chan struct{}, reportInterval int, wg *sync.WaitGroup) {
		defer wg.Done()
		time.Sleep(time.Second * time.Duration(reportInterval+5))

		for {
			select {
			case <-done:
				return
			default:
				// Устанавливаю ожидание, чтобы агент успел отправить новые метрики
				time.Sleep(time.Second * time.Duration(reportInterval+5))
				newPollCount, err := getPollCount(serverAdress)
				require.NoError(t, err)
				assert.Equal(t, true, newPollCount > oldPollCount)

				oldPollCount = newPollCount
			}
		}
	}(checkerDone, reportInterval, &wg)

	// Ожидаю, пока тест отработает----------------------------------------------------
	time.Sleep(1 * time.Minute) // Жду 5 минут для сбора профиля работы сервиса

	// Останавливаю тест----------------------------------------------------------------
	// Останавливаю функцию проверки метрики PollCount
	close(checkerDone)
	time.Sleep(2 * time.Second)
	// Останавливаю сам сервис
	close(done)
	wg.Wait()

	// создаём файл журнала профилирования памяти
	fmem, err := os.Create(`./../../profiles/result.pprof`)
	if err != nil {
		panic(err)
	}
	defer fmem.Close()
	runtime.GC() // получаем статистику по использованию памяти
	if err := pprof.WriteHeapProfile(fmem); err != nil {
		panic(err)
	}
}

func TestGRPCMusthaveMetrics(t *testing.T) {
	// Функция для очистки данных в базе
	cleanBD := func(dsn string) {
		// очищаю данные в тестовой бд------------------------------------------------------
		// создаём соединение с СУБД PostgreSQL
		conn, err := sql.Open("pgx", dsn)
		require.NoError(t, err)
		defer conn.Close()

		// Проверка соединения с БД
		ctx := context.Background()
		err = conn.PingContext(ctx)
		require.NoError(t, err)

		// создаем экземпляр хранилища pg
		stor := pg.NewStore(conn)
		err = stor.Bootstrap(ctx)
		require.NoError(t, err)
		err = stor.Disable(ctx)
		require.NoError(t, err)
	}

	// функция для получения свободного порта для запуска приложений
	getFreePort := func() (int, error) {
		// Слушаем на порту 0, чтобы операционная система выбрала свободный порт
		listener, err := net.Listen("tcp", ":0")
		if err != nil {
			return 0, err
		}
		defer listener.Close()

		// Получаем назначенный системой порт
		port := listener.Addr().(*net.TCPAddr).Port
		return port, nil
	}

	// функция для очистки файлов с ключами
	removeDir := func(dir string) {
		err := os.RemoveAll(dir)
		require.NoError(t, err)
	}

	// генирирую ключи для ассиметричного шифрования
	pathKeys := "./grpc-and-http-keys"
	err := os.Mkdir(pathKeys, 0755) // 0755 - права доступа
	require.NoError(t, err)

	err = encryption.GenerateKeys(pathKeys)
	require.NoError(t, err)
	defer removeDir(pathKeys)

	key := "secret key"

	var done = make(chan struct{})
	var wg sync.WaitGroup

	// Определяю параметры для запуска http сервера
	httpServerPort, err := getFreePort()
	require.NoError(t, err)
	httpServerAdress := fmt.Sprintf(":%d", httpServerPort)

	// Определяю параметры для запуска grpc сервера
	grpcServerPort, err := getFreePort()
	require.NoError(t, err)
	grpcServerAdress := fmt.Sprintf(":%d", grpcServerPort)

	databaseDsn := "host=localhost user=benchmarkmetrics password=password dbname=benchmarkmetrics sslmode=disable"

	// Очищаю данные в БД от предыдущих запусков
	cleanBD(databaseDsn)
	// Очищаю данные в БД после завершения теста
	defer cleanBD(databaseDsn)

	// Запускаю server-----------------------------------------------------
	cmdServer := exec.Command("./server", fmt.Sprintf("-a=%s", httpServerAdress), fmt.Sprintf("-grpc-address=%s", grpcServerAdress),
		fmt.Sprintf("-k=%s", key), fmt.Sprintf("-d=%s", databaseDsn), "-l=info", fmt.Sprintf("-crypto-key=%s", pathKeys+"/private_key.pem"))
	// Связываем стандартный вывод и ошибки программы с выводом программы Go
	cmdServer.Stdout = log.Writer()
	cmdServer.Stderr = log.Writer()

	// // Функция остановки сервера
	stopServer := func() {
		err = cmdServer.Process.Signal(os.Interrupt) // Посылаем сигнал прерывания
		require.NoError(t, err)
	}

	// Функция запуска сервера
	startServer := func(w *sync.WaitGroup) {
		// Запуск программы
		err = cmdServer.Start()
		require.NoError(t, err)

		<-done
		stopServer()
		w.Done()
	}

	// Запускаю сервис агента--------------------------------------------------
	agentAdress := grpcServerAdress
	reportInterval := 10
	cmdAgent := exec.Command("./../agent/agent", fmt.Sprintf("-a=%s", agentAdress), fmt.Sprintf("-k=%s", key), fmt.Sprintf("-r=%d", reportInterval),
		fmt.Sprintf("-crypto-key=%s", pathKeys+"/public_key.pem"), fmt.Sprintf("-protocol=%s", "grpc"))
	// Связываем стандартный вывод и ошибки программы с выводом программы Go
	cmdAgent.Stdout = log.Writer()
	cmdAgent.Stderr = log.Writer()

	// Функция остановки агента
	stopAgent := func() {
		err = cmdAgent.Process.Signal(os.Interrupt) // Посылаем сигнал прерывания
		require.NoError(t, err)
	}

	// Функция запуска агента
	startAgent := func(w *sync.WaitGroup) {
		err = cmdAgent.Start()
		require.NoError(t, err)

		<-done
		stopAgent()
		w.Done()
	}

	wg.Add(1)
	go startServer(&wg)
	wg.Add(1)
	go startAgent(&wg)
	time.Sleep(2 * time.Second) // Жду 2 секунды для запуска сервиса

	// проверяю, что с каждой новой отправкой метрик на сервер счетчик PollCount увеличивается----------------------------------------
	// создаю для этой проверки собственный канал, чтобы завершить отправку запросов к серверу перед остановкой самого сервера.
	var checkerDone = make(chan struct{})
	wg.Add(1)
	oldPollCount := int64(0)
	go func(done chan struct{}, reportInterval int, wg *sync.WaitGroup) {
		defer wg.Done()
		time.Sleep(time.Second * time.Duration(reportInterval+5))

		for {
			select {
			case <-done:
				return
			default:
				// Устанавливаю ожидание, чтобы агент успел отправить новые метрики
				time.Sleep(time.Second * time.Duration(reportInterval+5))
				newPollCount, err := getPollCount(httpServerAdress)
				require.NoError(t, err)
				assert.Equal(t, true, newPollCount > oldPollCount)

				oldPollCount = newPollCount
			}
		}
	}(checkerDone, reportInterval, &wg)

	// Ожидаю, пока тест отработает----------------------------------------------------
	time.Sleep(1 * time.Minute) // Жду 5 минут для сбора профиля работы сервиса

	// Останавливаю тест----------------------------------------------------------------
	// Останавливаю функцию проверки метрики PollCount
	close(checkerDone)
	time.Sleep(2 * time.Second)
	// Останавливаю сам сервис
	close(done)
	wg.Wait()

	// создаём файл журнала профилирования памяти
	fmem, err := os.Create(`./../../profiles/grpc_result.pprof`)
	if err != nil {
		panic(err)
	}
	defer fmem.Close()
	runtime.GC() // получаем статистику по использованию памяти
	if err := pprof.WriteHeapProfile(fmem); err != nil {
		panic(err)
	}
}
