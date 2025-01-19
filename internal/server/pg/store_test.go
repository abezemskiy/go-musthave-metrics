package pg

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strconv"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/repositories"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDisable(t *testing.T) {
	// функция для проверки двух float чисел
	floatsEqual := func(a, b, epsilon float64) bool {
		return math.Abs(a-b) < epsilon
	}

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
		stor := NewStore(conn)
		err = stor.Bootstrap(ctx)
		require.NoError(t, err)
		err = stor.Disable(ctx)
		require.NoError(t, err)
	}
	databaseDsn := "host=localhost user=benchmarkmetrics password=password dbname=benchmarkmetrics sslmode=disable"

	// Очищаю данные в БД от предыдущих запусков
	cleanBD(databaseDsn)

	// создаём соединение с СУБД PostgreSQL
	conn, err := sql.Open("pgx", databaseDsn)
	require.NoError(t, err)
	defer conn.Close()

	// Проверка соединения с БД
	ctx := context.Background()
	err = conn.PingContext(ctx)
	require.NoError(t, err)

	// создаем экземпляр хранилища pg
	stor := NewStore(conn)
	err = stor.Bootstrap(ctx)
	require.NoError(t, err)

	// Добавляю ненулевую положительную gauge метрику
	{
		value := 82352.23532
		err := stor.AddGauge(ctx, "positive gauge", value)
		require.NoError(t, err)
		// проверяю наличие метрики в базе данных
		valueGetStr, err := stor.GetMetric(ctx, "gauge", "positive gauge")
		require.NoError(t, err)
		valueGet, err := strconv.ParseFloat(valueGetStr, 64)
		require.NoError(t, err)
		require.Equal(t, true, floatsEqual(value, valueGet, 0.00001))
	}

	// Очищаю БД и проверяю, что добавленную ранее метрику не получить
	cleanBD(databaseDsn)
	{
		// проверяю наличие метрики в базе данных
		_, err := stor.GetMetric(ctx, "gauge", "positive gauge")
		require.Error(t, err)
	}

	{
		// Пытаюсь вызвать метод Disable, хотя контекст уже отменен
		ctxCanceled, cancel := context.WithCancel(context.Background())
		cancel()
		err = stor.Disable(ctxCanceled)
		require.Error(t, err)
	}
	{
		// Пытаюсь вызвать метод Bootstrap, хотя контекст уже отменен
		ctxCanceled, cancel := context.WithCancel(context.Background())
		cancel()
		err = stor.Bootstrap(ctxCanceled)
		require.Error(t, err)
	}
}

func TestGetMetric(t *testing.T) {
	// функция для проверки двух float чисел
	floatsEqual := func(a, b, epsilon float64) bool {
		return math.Abs(a-b) < epsilon
	}

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
		stor := NewStore(conn)
		err = stor.Bootstrap(ctx)
		require.NoError(t, err)
		err = stor.Disable(ctx)
		require.NoError(t, err)
	}
	databaseDsn := "host=localhost user=benchmarkmetrics password=password dbname=benchmarkmetrics sslmode=disable"

	// Очищаю данные в БД от предыдущих запусков
	cleanBD(databaseDsn)

	// создаём соединение с СУБД PostgreSQL
	conn, err := sql.Open("pgx", databaseDsn)
	require.NoError(t, err)
	defer conn.Close()

	// Проверка соединения с БД
	ctx := context.Background()
	err = conn.PingContext(ctx)
	require.NoError(t, err)

	// создаем экземпляр хранилища pg
	stor := NewStore(conn)
	err = stor.Bootstrap(ctx)
	require.NoError(t, err)

	// Добавляю ненулевую положительную gauge метрику
	{
		value := 82352.23532
		err := stor.AddGauge(ctx, "positive gauge", value)
		require.NoError(t, err)
		// проверяю наличие метрики в базе данных
		valueGetStr, err := stor.GetMetric(ctx, "gauge", "positive gauge")
		require.NoError(t, err)
		valueGet, err := strconv.ParseFloat(valueGetStr, 64)
		require.NoError(t, err)
		require.Equal(t, true, floatsEqual(value, valueGet, 0.00001))
	}
	// Добавляю ненулевую положительную counter метрику
	{
		value := int64(82352)
		err := stor.AddCounter(ctx, "positive counter", value)
		require.NoError(t, err)
		// проверяю наличие метрики в базе данных
		valueGetStr, err := stor.GetMetric(ctx, "counter", "positive counter")
		require.NoError(t, err)
		valueGet, err := strconv.ParseInt(valueGetStr, 10, 64)
		require.NoError(t, err)
		require.Equal(t, value, valueGet)
	}
	// Получение ошибки при разных типах метрики и в запросе и в базе
	{
		_, err := stor.GetMetric(ctx, "gauge", "positive counter")
		require.Error(t, err)
	}
	{
		_, err := stor.GetMetric(ctx, "counter", "positive gauge")
		require.Error(t, err)
	}
	// Попытка получения метрики, не хранящейся в базе
	{
		_, err := stor.GetMetric(ctx, "counter", "not found metric")
		require.Error(t, err)
	}
	// попытка получить метрику с неверным типом
	{
		_, err := stor.GetMetric(ctx, "wrong metric type", "")
		require.Error(t, err)
	}
	// попытка получить метрику с отмененным контекстом
	{
		ctxCanceled, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := stor.GetMetric(ctxCanceled, "wrong metric type", "")
		require.Error(t, err)
	}
}

func TestGetAllMetrics(t *testing.T) {
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
		stor := NewStore(conn)
		err = stor.Bootstrap(ctx)
		require.NoError(t, err)
		err = stor.Disable(ctx)
		require.NoError(t, err)
	}
	databaseDsn := "host=localhost user=benchmarkmetrics password=password dbname=benchmarkmetrics sslmode=disable"

	// Очищаю данные в БД от предыдущих запусков
	cleanBD(databaseDsn)

	// создаём соединение с СУБД PostgreSQL
	conn, err := sql.Open("pgx", databaseDsn)
	require.NoError(t, err)
	defer conn.Close()

	// Проверка соединения с БД
	ctx := context.Background()
	err = conn.PingContext(ctx)
	require.NoError(t, err)

	// создаем экземпляр хранилища pg
	stor := NewStore(conn)
	err = stor.Bootstrap(ctx)
	require.NoError(t, err)

	// Проверка обращения к пустой базе
	get, err := stor.GetAllMetrics(ctx)
	require.NoError(t, err)
	assert.Equal(t, "", get)

	// Заполняю базу
	valueGauge := 82352.23532
	err = stor.AddGauge(ctx, "positive gauge", valueGauge)
	require.NoError(t, err)
	valueCounter := int64(82352)
	err = stor.AddCounter(ctx, "positive counter", valueCounter)
	require.NoError(t, err)

	// Проверяю результат
	var result string
	result += fmt.Sprintf("type: %s, name: %s, value: %g\n", "gauge", "positive gauge", valueGauge)
	result += fmt.Sprintf("type: %s, name: %s, value: %d\n", "counter", "positive counter", valueCounter)
	get, err = stor.GetAllMetrics(ctx)
	require.NoError(t, err)
	assert.Equal(t, result, get)

	// попытка получения метрик, когда контекст уже отменен
	ctx1, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = stor.GetAllMetrics(ctx1)
	require.Error(t, err)
}

func TestAddGauge(t *testing.T) {
	// функция для проверки двух float чисел
	floatsEqual := func(a, b, epsilon float64) bool {
		return math.Abs(a-b) < epsilon
	}

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
		stor := NewStore(conn)
		err = stor.Bootstrap(ctx)
		require.NoError(t, err)
		err = stor.Disable(ctx)
		require.NoError(t, err)
	}
	databaseDsn := "host=localhost user=benchmarkmetrics password=password dbname=benchmarkmetrics sslmode=disable"

	// Очищаю данные в БД от предыдущих запусков
	cleanBD(databaseDsn)

	// создаём соединение с СУБД PostgreSQL
	conn, err := sql.Open("pgx", databaseDsn)
	require.NoError(t, err)
	defer conn.Close()

	// Проверка соединения с БД
	ctx := context.Background()
	err = conn.PingContext(ctx)
	require.NoError(t, err)

	// создаем экземпляр хранилища pg
	stor := NewStore(conn)
	err = stor.Bootstrap(ctx)
	require.NoError(t, err)

	// Добавляю ненулевую положительную метрику
	{
		value := 82352.23532
		err := stor.AddGauge(ctx, "positive gauge", value)
		require.NoError(t, err)
		// проверяю наличие метрики в базе данных
		valueGetStr, err := stor.GetMetric(ctx, "gauge", "positive gauge")
		require.NoError(t, err)
		valueGet, err := strconv.ParseFloat(valueGetStr, 64)
		require.NoError(t, err)
		require.Equal(t, true, floatsEqual(value, valueGet, 0.00001))
	}
	// Добавляю ненулевую отрицательную метрику
	{
		value := -82352.23532
		err := stor.AddGauge(ctx, "negative gauge", value)
		require.NoError(t, err)
		// проверяю наличие метрики в базе данных
		valueGetStr, err := stor.GetMetric(ctx, "gauge", "negative gauge")
		require.NoError(t, err)
		valueGet, err := strconv.ParseFloat(valueGetStr, 64)
		require.NoError(t, err)
		require.Equal(t, true, floatsEqual(value, valueGet, 0.00001))
	}
	// Добавляю нулевую метрику
	{
		value := 0.0
		err := stor.AddGauge(ctx, "zero gauge", value)
		require.NoError(t, err)
		// проверяю наличие метрики в базе данных
		valueGetStr, err := stor.GetMetric(ctx, "gauge", "zero gauge")
		require.NoError(t, err)
		valueGet, err := strconv.ParseFloat(valueGetStr, 64)
		require.NoError(t, err)
		require.Equal(t, true, floatsEqual(value, valueGet, 0.00001))
	}
	// пытаюсь добавить метрику, хотя контекст уже отменен
	{
		ctx1, cancel := context.WithCancel(context.Background())
		cancel()
		value := 0.0
		err := stor.AddGauge(ctx1, "zero gauge", value)
		require.Error(t, err)
	}
}

func TestAddCounter(t *testing.T) {
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
		stor := NewStore(conn)
		err = stor.Bootstrap(ctx)
		require.NoError(t, err)
		err = stor.Disable(ctx)
		require.NoError(t, err)
	}
	databaseDsn := "host=localhost user=benchmarkmetrics password=password dbname=benchmarkmetrics sslmode=disable"

	// Очищаю данные в БД от предыдущих запусков
	cleanBD(databaseDsn)

	// создаём соединение с СУБД PostgreSQL
	conn, err := sql.Open("pgx", databaseDsn)
	require.NoError(t, err)
	defer conn.Close()

	// Проверка соединения с БД
	ctx := context.Background()
	err = conn.PingContext(ctx)
	require.NoError(t, err)

	// создаем экземпляр хранилища pg
	stor := NewStore(conn)
	err = stor.Bootstrap(ctx)
	require.NoError(t, err)

	// Добавляю ненулевую положительную метрику
	{
		value := int64(82352)
		err := stor.AddCounter(ctx, "positive counter", value)
		require.NoError(t, err)
		// проверяю наличие метрики в базе данных
		valueGetStr, err := stor.GetMetric(ctx, "counter", "positive counter")
		require.NoError(t, err)
		valueGet, err := strconv.ParseInt(valueGetStr, 10, 64)
		require.NoError(t, err)
		require.Equal(t, value, valueGet)
	}
	// Добавляю ненулевую отрицательную метрику
	{
		value := int64(-82352)
		err := stor.AddCounter(ctx, "negative counter", value)
		require.NoError(t, err)
		// проверяю наличие метрики в базе данных
		valueGetStr, err := stor.GetMetric(ctx, "counter", "negative counter")
		require.NoError(t, err)
		valueGet, err := strconv.ParseInt(valueGetStr, 10, 64)
		require.NoError(t, err)
		require.Equal(t, value, valueGet)
	}
	// Добавляю нулевую метрику
	{
		value := int64(0)
		err := stor.AddCounter(ctx, "zero counter", value)
		require.NoError(t, err)
		// проверяю наличие метрики в базе данных
		valueGetStr, err := stor.GetMetric(ctx, "counter", "zero counter")
		require.NoError(t, err)
		valueGet, err := strconv.ParseInt(valueGetStr, 10, 64)
		require.NoError(t, err)
		require.Equal(t, value, valueGet)
	}
	// пытаюсь добавить метрику, хотя контекст уже отменен
	{
		ctx1, cancel := context.WithCancel(context.Background())
		cancel()
		value := int64(0)
		err := stor.AddCounter(ctx1, "zero gauge", value)
		require.Error(t, err)
	}
}

func TestAddMetricsFromSlice(t *testing.T) {
	// функция для проверки двух float чисел
	floatsEqual := func(a, b, epsilon float64) bool {
		return math.Abs(a-b) < epsilon
	}

	delta := func(d int64) *int64 {
		return &d
	}
	value := func(v float64) *float64 {
		return &v
	}

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
		stor := NewStore(conn)
		err = stor.Bootstrap(ctx)
		require.NoError(t, err)
		err = stor.Disable(ctx)
		require.NoError(t, err)
	}
	databaseDsn := "host=localhost user=benchmarkmetrics password=password dbname=benchmarkmetrics sslmode=disable"

	// Очищаю данные в БД от предыдущих запусков
	cleanBD(databaseDsn)

	// создаём соединение с СУБД PostgreSQL
	conn, err := sql.Open("pgx", databaseDsn)
	require.NoError(t, err)
	defer conn.Close()

	// Проверка соединения с БД
	ctx := context.Background()
	err = conn.PingContext(ctx)
	require.NoError(t, err)

	// создаем экземпляр хранилища pg
	stor := NewStore(conn)
	err = stor.Bootstrap(ctx)
	require.NoError(t, err)

	// отправляю в запросе к базе nil слайс
	err = stor.AddMetricsFromSlice(ctx, nil)
	require.NoError(t, err)

	// Создаю слайс с метриками для загрузки в базу
	slice := []repositories.Metric{
		{
			MType: "counter",
			ID:    "positive counter",
			Delta: delta(82352),
		},
		{
			MType: "counter",
			ID:    "negative counter",
			Delta: delta(-82352),
		},
		{
			MType: "counter",
			ID:    "zero counter",
			Delta: delta(0),
		},
		{
			MType: "gauge",
			ID:    "positive gauge",
			Value: value(82352.34534),
		},
		{
			MType: "gauge",
			ID:    "negative gauge",
			Value: value(-82352.34534),
		},
		{
			MType: "gauge",
			ID:    "zero gauge",
			Value: value(0.0),
		},
	}
	err = stor.AddMetricsFromSlice(ctx, slice)
	require.NoError(t, err)

	// Проверяю наличие метрик в базе
	valueGetStr, err := stor.GetMetric(ctx, "counter", "positive counter")
	require.NoError(t, err)
	valueGet, err := strconv.ParseInt(valueGetStr, 10, 64)
	require.NoError(t, err)
	require.Equal(t, int64(82352), valueGet)

	valueGetStr, err = stor.GetMetric(ctx, "counter", "negative counter")
	require.NoError(t, err)
	valueGet, err = strconv.ParseInt(valueGetStr, 10, 64)
	require.NoError(t, err)
	require.Equal(t, int64(-82352), valueGet)

	valueGetStr, err = stor.GetMetric(ctx, "counter", "zero counter")
	require.NoError(t, err)
	valueGet, err = strconv.ParseInt(valueGetStr, 10, 64)
	require.NoError(t, err)
	require.Equal(t, int64(0), valueGet)

	valueGetStr, err = stor.GetMetric(ctx, "gauge", "positive gauge")
	require.NoError(t, err)
	valueGaugeGet, err := strconv.ParseFloat(valueGetStr, 64)
	require.NoError(t, err)
	require.Equal(t, true, floatsEqual(float64(82352.34534), valueGaugeGet, 0.00001))

	valueGetStr, err = stor.GetMetric(ctx, "gauge", "negative gauge")
	require.NoError(t, err)
	valueGaugeGet, err = strconv.ParseFloat(valueGetStr, 64)
	require.NoError(t, err)
	require.Equal(t, true, floatsEqual(float64(-82352.34534), valueGaugeGet, 0.00001))

	valueGetStr, err = stor.GetMetric(ctx, "gauge", "zero gauge")
	require.NoError(t, err)
	valueGaugeGet, err = strconv.ParseFloat(valueGetStr, 64)
	require.NoError(t, err)
	require.Equal(t, true, floatsEqual(float64(0), valueGaugeGet, 0.00001))

	// попытка добавить метрики, когда контекст уже отменен
	ctx1, cancel := context.WithCancel(context.Background())
	cancel()
	err = stor.AddMetricsFromSlice(ctx1, slice)
	require.Error(t, err)
}

func TestGetAllMetricsSlice(t *testing.T) {
	delta := func(d int64) *int64 {
		return &d
	}
	value := func(v float64) *float64 {
		return &v
	}

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
		stor := NewStore(conn)
		err = stor.Bootstrap(ctx)
		require.NoError(t, err)
		err = stor.Disable(ctx)
		require.NoError(t, err)
	}
	databaseDsn := "host=localhost user=benchmarkmetrics password=password dbname=benchmarkmetrics sslmode=disable"

	// Очищаю данные в БД от предыдущих запусков
	cleanBD(databaseDsn)

	// создаём соединение с СУБД PostgreSQL
	conn, err := sql.Open("pgx", databaseDsn)
	require.NoError(t, err)
	defer conn.Close()

	// Проверка соединения с БД
	ctx := context.Background()
	err = conn.PingContext(ctx)
	require.NoError(t, err)

	// создаем экземпляр хранилища pg
	stor := NewStore(conn)
	err = stor.Bootstrap(ctx)
	require.NoError(t, err)

	// Создаю слайс с метриками для загрузки в базу
	slice := []repositories.Metric{
		{
			MType: "counter",
			ID:    "positive counter",
			Delta: delta(82352),
		},
		{
			MType: "counter",
			ID:    "negative counter",
			Delta: delta(-82352),
		},
		{
			MType: "counter",
			ID:    "zero counter",
			Delta: delta(0),
		},
		{
			MType: "gauge",
			ID:    "positive gauge",
			Value: value(82352.34534),
		},
		{
			MType: "gauge",
			ID:    "negative gauge",
			Value: value(-82352.34534),
		},
		{
			MType: "gauge",
			ID:    "zero gauge",
			Value: value(0.0),
		},
	}
	err = stor.AddMetricsFromSlice(ctx, slice)
	require.NoError(t, err)

	// Проверяю наличие метрик в базе
	resSlice, err := stor.GetAllMetricsSlice(ctx)
	require.NoError(t, err)
	assert.Equal(t, slice, resSlice)

	// попытка добавить метрики, когда контекст уже отменен
	ctx1, cancel := context.WithCancel(context.Background())
	cancel()
	err = stor.AddMetricsFromSlice(ctx1, slice)
	require.Error(t, err)
}
