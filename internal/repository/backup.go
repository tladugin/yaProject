package repository

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tladugin/yaProject.git/internal/logger"
	models "github.com/tladugin/yaProject.git/internal/models"
)

// Consumer отвечает за чтение данных из файла бэкапа
type Consumer struct {
	file   *os.File
	reader *bufio.Reader
}

// NewConsumer создает новый экземпляр Consumer для чтения бэкапов
func NewConsumer(filename string) (*Consumer, error) {
	file, err := os.OpenFile(filename, os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}

	return &Consumer{
		file:   file,
		reader: bufio.NewReader(file),
	}, nil
}

// Close закрывает файл потребителя
func (c *Consumer) Close() error {
	return c.file.Close()
}

// ReadEvent читает одну запись метрики из файла
func (c *Consumer) ReadEvent() (*models.Metrics, error) {
	mutex.Lock()
	defer mutex.Unlock()

	data, err := c.reader.ReadBytes('\n')
	if err != nil {
		return nil, err
	}

	event := models.Metrics{}
	err = json.Unmarshal(data, &event)
	if err != nil {
		return nil, err
	}

	return &event, nil
}

// Producer отвечает за запись данных в файл бэкапа
type Producer struct {
	file   *os.File
	writer *bufio.Writer
}

// NewProducer создает новый экземпляр Producer для записи бэкапов
func NewProducer(filename string) (*Producer, error) {
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}

	return &Producer{
		file:   file,
		writer: bufio.NewWriter(file),
	}, nil
}

// Close закрывает файл продюсера
func (p *Producer) Close() error {
	return p.file.Close()
}

// WriteEvent записывает одну запись метрики в файл
func (p *Producer) WriteEvent(event *models.Metrics) error {
	mutex.Lock()
	defer mutex.Unlock()

	data, err := json.Marshal(&event)
	if err != nil {
		return err
	}

	if _, err := p.writer.Write(data); err != nil {
		return err
	}

	if err := p.writer.WriteByte('\n'); err != nil {
		return err
	}

	return p.writer.Flush()
}

// RestoreFromBackup восстанавливает данные хранилища из файла бэкапа
func RestoreFromBackup(storage *MemStorage, flagFileStoragePath string) error {
	consumer, err := NewConsumer(flagFileStoragePath)
	if err != nil {
		return fmt.Errorf("failed to create consumer: %w", err)
	}
	defer consumer.Close()

	event, err := consumer.ReadEvent()
	if err != nil {
		return fmt.Errorf("failed to read event: %w", err)
	}

	for event != nil {
		switch event.MType {
		case "gauge":
			storage.AddGauge(event.ID, *event.Value)
		case "counter":
			storage.AddCounter(event.ID, *event.Delta)
		}

		event, err = consumer.ReadEvent()
		if err != nil {
			logger.Sugar.Info("Backup restore completed")
			break
		}
	}

	return nil
}

// RunPeriodicBackupWithContext запускает периодическое создание бэкапов с контекстом
func RunPeriodicBackupWithContext(ctx context.Context, storage *MemStorage, producer *Producer, interval time.Duration, filename string) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Sugar.Info("Periodic backup stopped")
			return
		case <-ticker.C:
			if err := performBackup(storage, producer, filename); err != nil {
				logger.Sugar.Errorw("Periodic backup failed", "error", err)
			} else {
				logger.Sugar.Debug("Periodic backup complete")
			}
		}
	}
}

// RunFinalBackupWithContext выполняет финальный бэкап при завершении работы приложения
func RunFinalBackupWithContext(ctx context.Context, storage *MemStorage, producer *Producer, filename string) {
	<-ctx.Done()
	logger.Sugar.Info("Starting final backup...")

	if err := performBackup(storage, producer, filename); err != nil {
		logger.Sugar.Errorw("Final backup failed", "error", err)
	} else {
		logger.Sugar.Info("Final backup completed")
	}
}

// SaveBackup сохраняет бэкап (упрощенная версия performBackup)
func SaveBackup(storage *MemStorage, producer *Producer, filename string) error {
	return performBackup(storage, producer, filename)
}

// performBackup выполняет фактическое создание бэкапа
func performBackup(storage *MemStorage, producer *Producer, flagFileStoragePath string) error {
	var producerM sync.Mutex

	producerM.Lock()
	defer producerM.Unlock()

	// Закрытие текущего продюсера
	if err := producer.Close(); err != nil {
		logger.Sugar.Errorw("Failed to close producer", "error", err)
	}

	// Удаление старого бэкапа если существует
	oldBackup := flagFileStoragePath + "_old"
	if err := os.Remove(oldBackup); err != nil && !os.IsNotExist(err) {
		logger.Sugar.Debugw("No old backup file to remove", "error", err)
	}

	// Переименование текущего бэкапа в старый
	if err := os.Rename(flagFileStoragePath, oldBackup); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to rename backup: %w", err)
	}

	// Создание нового продюсера
	newProducer, err := NewProducer(flagFileStoragePath)
	if err != nil {
		return fmt.Errorf("failed to create new producer: %w", err)
	}

	// Бэкап gauge метрик
	for _, gauge := range storage.GaugeSlice() {
		backup := models.Metrics{
			ID:    gauge.Name,
			MType: "gauge",
			Value: &gauge.Value,
		}
		if err := newProducer.WriteEvent(&backup); err != nil {
			return fmt.Errorf("failed to write gauge metric: %w", err)
		}
	}

	// Бэкап counter метрик
	for _, counter := range storage.CounterSlice() {
		backup := models.Metrics{
			ID:    counter.Name,
			MType: "counter",
			Delta: &counter.Value,
		}
		if err := newProducer.WriteEvent(&backup); err != nil {
			return fmt.Errorf("failed to write counter metric: %w", err)
		}
	}

	return nil
}

// performBackupToPostgres выполняет бэкап метрик в PostgreSQL
func performBackupToPostgres(storage *MemStorage, dbPool *pgxpool.Pool) error {
	var mu sync.Mutex
	ctx := context.Background()

	mu.Lock()
	defer mu.Unlock()

	tx, err := dbPool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Очищаем существующие данные
	if _, err := tx.Exec(ctx, "TRUNCATE TABLE gauges, counters"); err != nil {
		return fmt.Errorf("failed to truncate tables: %w", err)
	}

	// Бэкап gauge метрик
	for _, gauge := range storage.GaugeSlice() {
		_, err := tx.Exec(ctx,
			`INSERT INTO gauges (name, value) VALUES ($1, $2)
			 ON CONFLICT (name) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()`,
			gauge.Name, gauge.Value,
		)
		if err != nil {
			return fmt.Errorf("failed to backup gauge %s: %w", gauge.Name, err)
		}
	}

	// Бэкап counter метрик
	for _, counter := range storage.CounterSlice() {
		_, err := tx.Exec(ctx,
			`INSERT INTO counters (name, value) VALUES ($1, $2)
			 ON CONFLICT (name) DO UPDATE SET value = counters.value + EXCLUDED.value, updated_at = NOW()`,
			counter.Name, counter.Value,
		)
		if err != nil {
			return fmt.Errorf("failed to backup counter %s: %w", counter.Name, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	logger.Sugar.Info("Successfully backed up metrics to PostgreSQL")
	return nil
}
