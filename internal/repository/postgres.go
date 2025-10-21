package repository

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"log"
	"os"
	"time"
)

// UpdateGauge обновляет или создает метрику типа gauge в PostgreSQL
func UpdateGauge(pool *pgxpool.Pool, ctx context.Context, name string, value float64) error {
	_, err := pool.Exec(ctx,
		`INSERT INTO gauges (name, value)	
		 VALUES ($1, $2)	
		 ON CONFLICT (name) DO UPDATE 
		 SET value = EXCLUDED.value, updated_at = NOW()`,
		name, value)
	return err
}

// UpdateCounter обновляет или создает метрику типа counter в PostgreSQL
// При конфликте имени увеличивает существующее значение
func UpdateCounter(pool *pgxpool.Pool, ctx context.Context, name string, value int64) error {
	_, err := pool.Exec(ctx,
		`INSERT INTO counters (name, value) 
		 VALUES ($1, $2) 
		 ON CONFLICT (name) DO UPDATE 
		 SET value = counters.value + EXCLUDED.value, updated_at = NOW()`,
		name, value)
	return err
}

// GetGauge получает значение метрики типа gauge из PostgreSQL по имени
func GetGauge(pool *pgxpool.Pool, ctx context.Context, name string) (float64, error) {
	// Проверка входных параметров
	if pool == nil {
		return 0, fmt.Errorf("database pool is nil")
	}

	if name == "" {
		return 0, fmt.Errorf("metric name cannot be empty")
	}

	// Выполнение запроса
	var value float64
	err := pool.QueryRow(ctx, `SELECT value FROM gauges WHERE name = $1`, name).Scan(&value)

	// Обработка ошибок
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return 0, fmt.Errorf("gauge '%s' not found", name)
	case err != nil:
		return 0, fmt.Errorf("failed to get gauge '%s': %w", name, err)
	}

	return value, nil
}

// GetCounter получает значение метрики типа counter из PostgreSQL по имени
func GetCounter(pool *pgxpool.Pool, ctx context.Context, name string) (int64, error) {
	if pool == nil {
		return 0, fmt.Errorf("database pool is nil")
	}

	if name == "" {
		return 0, fmt.Errorf("metric name cannot be empty")
	}

	var value int64
	err := pool.QueryRow(ctx, `SELECT value FROM counters WHERE name = $1`, name).Scan(&value)

	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return 0, fmt.Errorf("counter '%s' not found", name)
	case err != nil:
		return 0, fmt.Errorf("failed to get counter '%s': %w", name, err)
	}

	return value, nil
}

// GetConnection создает и настраивает пул соединений с PostgreSQL
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
		// Пример настройки параметров пула соединений
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

	// Проверяем соединение с базой данных
	if err := pool.Ping(initCtx); err != nil {
		pool.Close()
		return nil, nil, nil, fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}

	// Возвращаем новый контекст для использования в вызывающем коде
	ctx, cancelFunc := context.WithCancel(context.Background())

	return pool, ctx, cancelFunc, nil
}

// applyMigrations применяет миграции базы данных для создания необходимых таблиц
func applyMigrations(db *pgxpool.Pool, ctx context.Context) error {
	// Проверяем существование таблицы миграций
	var exists bool
	err := db.QueryRow(ctx,
		`SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_name = 'migrations'
		)`).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check migrations table: %w", err)
	}

	if !exists {
		// Создаем таблицу для отслеживания примененных миграций
		log.Println("creating migrations table")
		_, err = db.Exec(ctx,
			`CREATE TABLE migrations (
				id SERIAL PRIMARY KEY, 
				name VARCHAR(255) NOT NULL, 
				applied_at TIMESTAMP NOT NULL DEFAULT NOW()
			)`)
		if err != nil {
			return fmt.Errorf("failed to create migrations table: %w", err)
		}
		log.Println("migrations table created")
	} else {
		log.Println("migrations table already exists")
	}

	// Применяем начальную миграцию для создания таблиц метрик
	var applied bool
	err = db.QueryRow(ctx,
		"SELECT EXISTS (SELECT 1 FROM migrations WHERE name = '000001_create_metrics_table')").Scan(&applied)
	if err != nil {
		return fmt.Errorf("failed to check initial migration: %w", err)
	}

	if !applied {
		// Проверяем существование файла миграции
		path := "migrations/000001_create_metrics_table.up.sql"
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return fmt.Errorf("файл миграции не найден: %s", path)
		}

		// Читаем SQL из файла миграции
		migrationSQL, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read migration file: %w", err)
		}

		// Выполняем миграцию в транзакции для обеспечения атомарности
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

		// Выполняем SQL миграции
		if _, err := tx.Exec(ctx, string(migrationSQL)); err != nil {
			return fmt.Errorf("failed to execute migration: %w", err)
		}

		// Записываем факт применения миграции
		if _, err := tx.Exec(ctx,
			"INSERT INTO migrations (name) VALUES ('migrations/000001_create_metrics_table.up.sql')"); err != nil {
			return fmt.Errorf("failed to record migration: %w", err)
		}

		// Фиксируем транзакцию
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("failed to commit migration: %w", err)
		}
	}
	log.Println("migration commited")
	return nil
}

// NewPostgresRepository создает новый репозиторий для работы с PostgreSQL
// Инициализирует соединение и применяет необходимые миграции
func NewPostgresRepository(databaseDSN string) (*pgxpool.Pool, context.Context, error) {
	// Получаем соединение с базой данных
	pool, ctx, cancel, err := GetConnection(databaseDSN)
	if err != nil {
		log.Println(err)
	}
	defer cancel()
	defer pool.Close()

	// Применяем миграции базы данных
	if err := applyMigrations(pool, ctx); err != nil {
		return nil, ctx, fmt.Errorf("failed to apply migrations: %w", err)
	}

	return pool, ctx, nil
}
