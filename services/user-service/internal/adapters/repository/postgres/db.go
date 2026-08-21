package postgres

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/bishtpramod19/ecommerce-platform/services/user-service/internal/config"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

// createDatabaseIfNotExists connects to the default 'postgres' database
// and creates our application database if it doesn't exist.
// This makes the service fully self-contained — no manual DB creation needed.
func createDatabaseIfNotExists(cfg *config.Config) error {
	// Connect to default postgres database (always exists)
	defaultDSN := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=postgres sslmode=disable",
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBUser,
		cfg.DBPassword,
	)

	db, err := sql.Open("postgres", defaultDSN)
	if err != nil {
		return fmt.Errorf("error connecting to default database: %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return fmt.Errorf("error pinging default database: %w", err)
	}

	// Create database if not exists
	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", cfg.DBName))
	if err != nil {
		// "already exists" is not an error — service may restart multiple times
		if strings.Contains(err.Error(), "already exists") {
			log.Printf("Database %s already exists, skipping creation", cfg.DBName)
			return nil
		}
		return fmt.Errorf("error creating database %s: %w", cfg.DBName, err)
	}

	log.Printf("Database %s created successfully", cfg.DBName)
	return nil
}

func NewPostgresDB(cfg *config.Config) (*sql.DB, error) {

	// Create database if not exists (self-contained setup)
	if err := createDatabaseIfNotExists(cfg); err != nil {
		return nil, fmt.Errorf("error ensuring database exists: %w", err)
	}

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBName,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)

	log.Println("Successfully connected to PostgreSQL")

	if err := runMigrations(db); err != nil {
		return nil, fmt.Errorf("error running migrations: %w", err)
	}

	return db, nil
}

func runMigrations(db *sql.DB) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("error creating migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("error creating migrate instance: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("error running migrations: %w", err)
	}

	log.Println("Migrations ran successfully")
	return nil
}
