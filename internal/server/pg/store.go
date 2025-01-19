package pg

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/repositories"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/logger"
)

// Store реализует интерфейс store.Store и позволяет взаимодействовать с СУБД PostgreSQL.
// Так же Store реализует интерфейс repositories.ServerRepo, для возможности использования структуры в качестве хранилища метрик.
type Store struct {
	// Поле conn содержит объект соединения с СУБД
	conn *sql.DB
}

// NewStore возвращает новый экземпляр PostgreSQL-хранилища
func NewStore(conn *sql.DB) *Store {
	return &Store{conn: conn}
}

// Bootstrap - подготавливает БД к работе, создавая необходимые таблицы и индексы.
func (s Store) Bootstrap(ctx context.Context) error {
	// запускаем транзакцию
	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	// в случае неуспешного коммита все изменения транзакции будут отменены
	defer tx.Rollback()

	// создаём таблицу с метриками и необходимые индексы, если таблица ещё не существует
	_, errExec := tx.ExecContext(ctx, `
        CREATE TABLE IF NOT EXISTS metrics (
			id varchar(128) PRIMARY KEY,
			mtype varchar(128),
			delta bigint DEFAULT NULL,
			value double precision DEFAULT NULL
        )
    `)
	if errExec != nil {
		return errExec
	}
	_, errExec = tx.ExecContext(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS id ON metrics (id)`)
	if errExec != nil {
		return errExec
	}

	// коммитим транзакцию
	return tx.Commit()
}

// Disable - очищает БД, удаляя записи из таблиц.
// Метод необходим для тестирования, чтобы в процессе удалять тестовые записи.
func (s Store) Disable(ctx context.Context) (err error) {
	logger.ServerLog.Debug("truncate all data in all tables")

	// запускаем транзакцию
	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	// в случае неуспешного коммита все изменения транзакции будут отменены
	defer tx.Rollback()

	// удаляю все записи в таблице auth
	_, err = tx.ExecContext(ctx, `
			TRUNCATE TABLE metrics 
	`)
	if err != nil {
		return err
	}

	// коммитим транзакцию
	return tx.Commit()
}

// GetMetric -возвращает значение метрики в строчном представлении по имени и типу метрики.
func (s Store) GetMetric(ctx context.Context, metricType string, metricName string) (string, error) {
	query := `
		SELECT id,
			   mtype,
			   delta,
			   value
		FROM metrics
		WHERE id = $1
	`
	stmt, err := s.conn.PrepareContext(ctx, query)
	if err != nil {
		return "", fmt.Errorf("prepare context error in DB, %w", err)
	}
	defer stmt.Close()
	row := stmt.QueryRowContext(ctx, metricName)

	var metric repositories.Metric
	err = row.Scan(&metric.ID, &metric.MType, &metric.Delta, &metric.Value)
	if err != nil {
		return "", err
	}
	if metric.MType != metricType {
		return "", fmt.Errorf("metric type is different, metric type in database is: %s, metric type in request is: %s", metric.MType, metricType)
	}

	switch metric.MType {
	case "gauge":
		if metric.Value == nil {
			return "", fmt.Errorf("value of gauge metric is nil")
		}
		return fmt.Sprintf("%g", *metric.Value), nil
	case "counter":
		if metric.Delta == nil {
			return "", fmt.Errorf("value of counter metric is nil")
		}
		return fmt.Sprintf("%d", *metric.Delta), nil
	default:
		return "", fmt.Errorf("whrong type of metric")
	}
}

// AddGauge - реализует метод AddGauge интерфейса repositories.ServerRepo.
func (s Store) AddGauge(ctx context.Context, nameMetric string, value float64) (err error) {
	queryUpsert := `
				INSERT INTO metrics (id, mtype, value)
				VALUES ($1, $2, $3)
				ON CONFLICT (id) 
				DO UPDATE SET value = EXCLUDED.value;
				`
	stmt, err := s.conn.PrepareContext(ctx, queryUpsert)
	if err != nil {
		return fmt.Errorf("prepare context error in DB, %w", err)
	}
	defer stmt.Close()
	_, err = stmt.ExecContext(ctx, nameMetric, "gauge", value)
	return err
}

// AddCounter - реализует метод AddCounter интерфейса repositories.ServerRepo.
func (s Store) AddCounter(ctx context.Context, nameMetric string, value int64) (err error) {
	queryUpsert := `
				INSERT INTO metrics (id, mtype, delta)
				VALUES ($1, $2, $3)
				ON CONFLICT (id) 
				DO UPDATE SET delta = metrics.delta + EXCLUDED.delta;
				`
	stmt, err := s.conn.PrepareContext(ctx, queryUpsert)
	if err != nil {
		return fmt.Errorf("prepare context error in DB, %w", err)
	}
	defer stmt.Close()
	_, err = stmt.ExecContext(ctx, nameMetric, "counter", value)
	return err
}

// GetAllMetrics - реализует метод GetAllMetrics интерфейса repositories.ServerRepo.
func (s Store) GetAllMetrics(ctx context.Context) (string, error) {
	metrics, err := s.GetAllMetricsSlice(ctx)
	if err != nil {
		return "", err
	}
	var result string
	for _, metric := range metrics {
		if metric.MType == "gauge" {
			result += fmt.Sprintf("type: %s, name: %s, value: %g\n", metric.MType, metric.ID, *metric.Value)
		} else {
			result += fmt.Sprintf("type: %s, name: %s, value: %d\n", metric.MType, metric.ID, *metric.Delta)
		}
	}
	return result, nil
}

// AddMetricsFromSlice - реализует метод AddMetricsFromSlice интерфейса repositories.ServerRepo.
func (s Store) AddMetricsFromSlice(ctx context.Context, metrics []repositories.Metric) error {
	// запускаем транзакцию
	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	// в случае неуспешного коммита все изменения транзакции будут отменены
	defer tx.Rollback()

	for _, metric := range metrics {

		if metric.MType == "gauge" {
			queryUpsert := `
				INSERT INTO metrics (id, mtype, value)
				VALUES ($1, $2, $3)
				ON CONFLICT (id) 
				DO UPDATE SET value = EXCLUDED.value;
				`
			stmt, err := tx.PrepareContext(ctx, queryUpsert)
			if err != nil {
				return fmt.Errorf("prepare context error in DB, %w", err)
			}
			defer stmt.Close()
			_, err = stmt.ExecContext(ctx, metric.ID, "gauge", metric.Value)
			if err != nil {
				return err
			}
		} else {
			queryUpsert := `
					INSERT INTO metrics (id, mtype, delta)
					VALUES ($1, $2, $3)
					ON CONFLICT (id) 
					DO UPDATE SET delta = metrics.delta + EXCLUDED.delta;
					`
			stmt, err := tx.PrepareContext(ctx, queryUpsert)
			if err != nil {
				return fmt.Errorf("prepare context error in DB, %w", err)
			}
			defer stmt.Close()
			_, err = stmt.ExecContext(ctx, metric.ID, "counter", metric.Delta)
			if err != nil {
				return err
			}
		}

	}
	// коммитим транзакцию
	return tx.Commit()
}

// GetAllMetricsSlice - реализует метод GetAllMetricsSlice интерфейса repositories.ServerRepo.
func (s Store) GetAllMetricsSlice(ctx context.Context) ([]repositories.Metric, error) {
	metrics := make([]repositories.Metric, 0)

	stmt, err := s.conn.PrepareContext(ctx, "SELECT id, mtype, delta, value FROM metrics")
	if err != nil {
		return nil, fmt.Errorf("prepare context error in DB, %w", err)
	}
	defer stmt.Close()
	rows, err := stmt.QueryContext(ctx)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	for rows.Next() {
		var metric repositories.Metric
		err = rows.Scan(&metric.ID, &metric.MType, &metric.Delta, &metric.Value)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, metric)
	}
	// проверяем на ошибки
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return metrics, nil
}
