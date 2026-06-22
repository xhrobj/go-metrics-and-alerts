package migrations

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"

	// Подключает поддержку миграций из файловой системы.
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// RunMigrations применяет SQL-миграции к подключенной базе данных PostgreSQL.
func RunMigrations(db *sql.DB) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("create migration driver: %w", err)
	}

	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working dir: %w", err)
	}

	path := fmt.Sprintf("file://%s/migrations", wd)

	m, err := migrate.NewWithDatabaseInstance(
		path,
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("create migrate instance: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("run migrations: %w", err)
	}

	return nil
}
