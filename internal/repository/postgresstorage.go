package repository

import (
	"database/sql"
	"errors"
	"fmt"
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
func (p *PostgresStorage) UpdateGauge(metricName string, value float64) error {
	_, err := p.db.Exec(
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
}

// UpdateCounter увеличивает значение counter-метрики на delta.
// Если метрика ещё не существует, она создаётся.
func (p *PostgresStorage) UpdateCounter(metricName string, delta int64) error {
	_, err := p.db.Exec(
		`INSERT INTO metrics (id, type, total)
		 VALUES ($1, 'counter', $2)
		 ON CONFLICT (id, type)
		 DO UPDATE SET total = metrics.total + EXCLUDED.delta`,
		metricName,
		delta,
	)

	if err != nil {
		return fmt.Errorf("exec upsert counter query: %w", err)
	}

	return nil
}

// GetGauge возвращает значение gauge-метрики.
// Если метрика не найдена, возвращается ErrMetricNotFound.
func (p *PostgresStorage) GetGauge(metricName string) (float64, error) {
	var value float64

	err := p.db.QueryRow(
		`SELECT value
		 FROM metrics
		 WHERE id = $1 AND type = 'gauge'`,
		metricName,
	).Scan(&value)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrMetricNotFound
		}
		return 0, fmt.Errorf("query gauge: %w", err)
	}

	return value, nil
}

// GetCounter возвращает значение counter-метрики.
// Если метрика не найдена, возвращается ErrMetricNotFound.
func (p *PostgresStorage) GetCounter(metricName string) (int64, error) {
	var total int64

	err := p.db.QueryRow(
		`SELECT total
		 FROM metrics
		 WHERE id = $1 AND type = 'counter'`,
		metricName,
	).Scan(&total)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrMetricNotFound
		}
		return 0, fmt.Errorf("query counter: %w", err)
	}

	return total, nil
}

// Snapshot возвращает снимок всех сохранённых метрик.
// Результат разделяется на две map: gauges и counters.
func (p *PostgresStorage) Snapshot() (map[string]float64, map[string]int64, error) {
	rows, err := p.db.Query(
		`SELECT id, type, total, value
		 FROM metrics`,
	)

	if err != nil {
		return nil, nil, fmt.Errorf("query snapshot: %w", err)
	}

	defer rows.Close()

	gauges := make(map[string]float64)
	counters := make(map[string]int64)

	for rows.Next() {
		var (
			id    string
			mType string
			total sql.NullInt64
			value sql.NullFloat64
		)

		if err := rows.Scan(&id, &mType, &total, &value); err != nil {
			return nil, nil, fmt.Errorf("scan snapshot row: %w", err)
		}

		switch mType {
		case "gauge":
			if value.Valid {
				gauges[id] = value.Float64
			}
		case "counter":
			if total.Valid {
				counters[id] = total.Int64
			}
		}
	}

	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate snapshot rows: %w", err)
	}

	return gauges, counters, nil
}
