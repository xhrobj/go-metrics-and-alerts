package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/xhrobj/go-metrics-and-alerts/internal/model"
)

// PostgresStorage реализует хранение метрик в базе данных PostgreSQL.
type PostgresStorage struct {
	db *sql.DB
}

// NewPostgresStorage создаёт новое PostgreSQL-хранилище метрик.
func NewPostgresStorage(db *sql.DB) *PostgresStorage {
	return &PostgresStorage{db: db}
}

// UpdateGauge сохраняет значение gauge-метрики.
// Если метрика уже существует, её значение перезаписывается.
func (p *PostgresStorage) UpdateGauge(ctx context.Context, metricName string, value float64) error {
	return retryDBOperation(ctx, func() error {
		_, err := p.db.ExecContext(
			ctx,
			`INSERT INTO metrics (id, type, value)
		 	 VALUES ($1, 'gauge', $2)
		 	 ON CONFLICT (id, type)
		 	 DO UPDATE SET value = EXCLUDED.value`,
			metricName,
			value,
		)

		if err != nil {
			return fmt.Errorf("exec upsert gauge query: %w", err)
		}

		return nil
	})
}

// UpdateCounter увеличивает значение counter-метрики на delta.
// Если метрика ещё не существует, она создаётся.
func (p *PostgresStorage) UpdateCounter(ctx context.Context, metricName string, delta int64) error {
	return retryDBOperation(ctx, func() error {
		_, err := p.db.ExecContext(
			ctx,
			`INSERT INTO metrics (id, type, total)
		 	 VALUES ($1, 'counter', $2)
		 	 ON CONFLICT (id, type)
		 	 DO UPDATE SET total = metrics.total + EXCLUDED.total`,
			metricName,
			delta,
		)

		if err != nil {
			return fmt.Errorf("exec upsert counter query: %w", err)
		}

		return nil
	})
}

// UpdateMetrics пакетно обновляет метрики в рамках одной транзакции.
func (p *PostgresStorage) UpdateMetrics(ctx context.Context, metrics []model.Metrics) error {
	return retryDBOperation(ctx, func() error {
		tx, err := p.db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin tx: %w", err)
		}

		defer func() {
			_ = tx.Rollback()
		}()

		for _, metric := range metrics {
			switch metric.MType {
			case model.Gauge:
				_, err = tx.ExecContext(
					ctx,
					`INSERT INTO metrics (id, type, value)
				 	 VALUES ($1, 'gauge', $2)
				 	 ON CONFLICT (id, type)
				 	 DO UPDATE SET value = EXCLUDED.value`,
					metric.ID,
					*metric.Value,
				)
				if err != nil {
					return fmt.Errorf("update gauge %q: %w", metric.ID, err)
				}

			case model.Counter:
				_, err = tx.ExecContext(
					ctx,
					`INSERT INTO metrics (id, type, total)
				 	 VALUES ($1, 'counter', $2)
				 	 ON CONFLICT (id, type)
				 	 DO UPDATE SET total = metrics.total + EXCLUDED.total`,
					metric.ID,
					*metric.Delta,
				)
				if err != nil {
					return fmt.Errorf("update counter %q: %w", metric.ID, err)
				}
			}
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit tx: %w", err)
		}

		return nil
	})
}

// GetGauge возвращает значение gauge-метрики.
// Если метрика не найдена, возвращается ErrMetricNotFound.
func (p *PostgresStorage) GetGauge(ctx context.Context, metricName string) (float64, error) {
	var value float64

	err := retryDBOperation(ctx, func() error {
		err := p.db.QueryRowContext(
			ctx,
			`SELECT value
		 	 FROM metrics
		 	 WHERE id = $1 AND type = 'gauge'`,
			metricName,
		).Scan(&value)

		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrMetricNotFound
			}
			return fmt.Errorf("query gauge: %w", err)
		}

		return nil
	})

	if err != nil {
		return 0, err
	}

	return value, nil
}

// GetCounter возвращает значение counter-метрики.
// Если метрика не найдена, возвращается ErrMetricNotFound.
func (p *PostgresStorage) GetCounter(ctx context.Context, metricName string) (int64, error) {
	var total int64

	err := retryDBOperation(ctx, func() error {
		err := p.db.QueryRowContext(
			ctx,
			`SELECT total
		 	 FROM metrics
		 	 WHERE id = $1 AND type = 'counter'`,
			metricName,
		).Scan(&total)

		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrMetricNotFound
			}
			return fmt.Errorf("query counter: %w", err)
		}

		return nil
	})

	if err != nil {
		return 0, err
	}

	return total, nil
}

// Snapshot возвращает снимок всех сохранённых метрик.
// Результат разделяется на две map: gauges и counters.
func (p *PostgresStorage) Snapshot(ctx context.Context) (map[string]float64, map[string]int64, error) {
	var (
		gauges   map[string]float64
		counters map[string]int64
	)

	err := retryDBOperation(ctx, func() error {
		rows, err := p.db.QueryContext(
			ctx,
			`SELECT id, type, total, value
		 	 FROM metrics`,
		)

		if err != nil {
			return fmt.Errorf("query snapshot: %w", err)
		}

		defer rows.Close()

		localGauges := make(map[string]float64)
		localCounters := make(map[string]int64)

		for rows.Next() {
			var (
				id    string
				mType string
				total sql.NullInt64
				value sql.NullFloat64
			)

			if err := rows.Scan(&id, &mType, &total, &value); err != nil {
				return fmt.Errorf("scan snapshot row: %w", err)
			}

			switch mType {
			case "gauge":
				if value.Valid {
					localGauges[id] = value.Float64
				}
			case "counter":
				if total.Valid {
					localCounters[id] = total.Int64
				}
			}
		}

		if err := rows.Err(); err != nil {
			return fmt.Errorf("iterate snapshot rows: %w", err)
		}

		gauges = localGauges
		counters = localCounters

		return nil
	})

	if err != nil {
		return nil, nil, err
	}

	return gauges, counters, nil
}

func retryDBOperation(ctx context.Context, op func() error) error {
	retryDelays := []time.Duration{
		time.Second * 1,
		time.Second * 3,
		time.Second * 5,
	}

	var lastErr error

	for attempt := 0; attempt <= len(retryDelays); attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		err := op()
		if err == nil {
			return nil
		}

		lastErr = err

		if !isRetriablePGError(err) || attempt == len(retryDelays) {
			return lastErr
		}

		time.Sleep(retryDelays[attempt])
	}

	return lastErr
}

func isRetriablePGError(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgerrcode.IsConnectionException(pgErr.Code)
}
