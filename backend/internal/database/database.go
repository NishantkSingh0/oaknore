package database

import (
	"fmt"
	"log"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"github.com/oaknore/pms3/internal/config"
)

// New opens a PostgreSQL connection pool and verifies connectivity.
func New(cfg config.DatabaseConfig) (*sqlx.DB, error) {
	db, err := sqlx.Open("postgres", cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(30 * time.Minute)

	// Verify connection with backoff
	const maxAttempts = 10
	for i := 1; i <= maxAttempts; i++ {
		if err = db.Ping(); err == nil {
			break
		}
		if i == maxAttempts {
			return nil, fmt.Errorf("ping db after %d attempts: %w", maxAttempts, err)
		}
		log.Printf("db not ready (attempt %d/%d), retrying in 2s…", i, maxAttempts)
		time.Sleep(2 * time.Second)
	}

	log.Println("database connected")
	return db, nil
}

// RunMigrations applies all pending UP migrations from the migrations/ directory.
// dsn must be a postgres:// URL — key=value DSNs are not accepted by golang-migrate.
func RunMigrations(cfg config.DatabaseConfig) error {
	// Build a proper postgres:// URL regardless of what was passed in.
	url := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name, cfg.SSLMode,
	)
	m, err := migrate.New("file://migrations", url)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	defer m.Close()

	if err = m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("run migrations: %w", err)
	}
	log.Println("migrations applied")
	return nil
}
