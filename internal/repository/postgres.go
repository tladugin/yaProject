package repository

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"log"
	"os"
	"time"
)

func GetConnection(databaseDSN string) (*pgxpool.Pool, context.Context, context.CancelFunc, error) {
	// Создаем контекст с таймаутом для инициализации подключения
	initCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	//initCtx := context.Background()
	// Парсим конфигурацию пула соединений
	poolConfig, err := pgxpool.ParseConfig(databaseDSN)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to parse PostgreSQL DSN: %w", err)
	}

	/*
		poolConfig.MinConns = 2
		poolConfig.MaxConns = 10
		poolConfig.MaxConnLifetime = 1 * time.Hour
		poolConfig.MaxConnIdleTime = 30 * time.Minute
		poolConfig.HealthCheckPeriod = 1 * time.Minute

	*/

	// Создаем пул соединений
	pool, err := pgxpool.NewWithConfig(initCtx, poolConfig)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Проверяем соединение
	if err := pool.Ping(initCtx); err != nil {
		pool.Close()
		return nil, nil, nil, fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}

	// Применяем миграции
	/*if err := applyMigrations(pool, initCtx); err != nil {
		pool.Close()
		return nil, nil, nil, fmt.Errorf("failed to apply migrations: %w", err)
	}

	*/

	// Возвращаем новый контекст для использования в вызывающем коде
	ctx, cancelFunc := context.WithCancel(context.Background())

	return pool, ctx, cancelFunc, nil
}
func applyMigrations(db *pgxpool.Pool, ctx context.Context) error {
	// Проверяем существование таблицы миграций
	var exists bool
	err := db.QueryRow(ctx, "SELECT EXISTS (SELECT FROM information_schema.tables 	WHERE table_name = 'migrations'	)").Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check migrations table: %w", err)

	}
	if !exists {
		// Создаем таблицу
		log.Println("creating migrations table")
		_, err = db.Exec(ctx, "CREATE TABLE migrations (	id SERIAL PRIMARY KEY, name VARCHAR(255) NOT NULL, applied_at TIMESTAMP NOT NULL DEFAULT NOW())")
		if err != nil {
			return fmt.Errorf("failed to create migrations table: %w", err)
		}
		log.Println("migrations table created")
	} else {
		log.Println("migrations table already exists")
	}

	// Применяем начальную миграцию
	var applied bool
	err = db.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM migrations WHERE name = '000001_create_metrics_table')").Scan(&applied)
	if err != nil {
		return fmt.Errorf("failed to check initial migration: %w", err)
	}

	if !applied {

		path := "migrations/000001_create_metrics_table.up.sql"
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return fmt.Errorf("файл миграции не найден: %s", path)
		}

		// Читаем SQL из файла миграции
		migrationSQL, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read migration file: %w", err)
		}

		// Выполняем в транзакции
		tx, err := db.Begin(ctx)
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}
		defer func(tx pgx.Tx, ctx context.Context) {
			err := tx.Rollback(ctx)
			if err != nil {
				log.Printf("failed to rollback transaction: %s", err)
			}
		}(tx, ctx)

		if _, err := tx.Exec(ctx, string(migrationSQL)); err != nil {
			return fmt.Errorf("failed to execute migration: %w", err)
		}

		if _, err := tx.Exec(ctx, "INSERT INTO migrations (name) VALUES ('migrations/000001_create_metrics_table.up.sql')"); err != nil {
			return fmt.Errorf("failed to record migration: %w", err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("failed to commit migration: %w", err)
		}
	}
	log.Println("migration commited")
	return nil
}
func NewPostgresRepository(databaseDSN string) (*pgxpool.Pool, context.Context, error) {
	pool, ctx, cancel, err := GetConnection(databaseDSN)
	if err != nil {
		log.Println(err)
	}
	defer cancel()
	defer pool.Close()
	if err := applyMigrations(pool, ctx); err != nil {
		return nil, ctx, fmt.Errorf("failed to apply migrations: %w", err)
	}

	return pool, ctx, nil
}
