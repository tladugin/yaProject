package repository

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tladugin/yaProject.git/internal/logger"
	models "github.com/tladugin/yaProject.git/internal/models"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type Consumer struct {
	file *os.File
	// добавляем reader в Consumer
	reader *bufio.Reader
}

func NewConsumer(filename string) (*Consumer, error) {
	file, err := os.OpenFile(filename, os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}

	return &Consumer{
		file: file,
		// создаём новый Reader
		reader: bufio.NewReader(file),
	}, nil
}
func (c *Consumer) Close() error {
	return c.file.Close()
}

func (c *Consumer) ReadEvent() (*models.Metrics, error) {
	mutex.Lock()
	defer mutex.Unlock()
	data, err := c.reader.ReadBytes('\n')

	if err != nil {
		return nil, err
	}

	// преобразуем данные из JSON-представления в структуру
	event := models.Metrics{}
	err = json.Unmarshal(data, &event)
	if err != nil {
		return nil, err
	}

	return &event, nil
}

type Producer struct {
	file *os.File
	// добавляем Writer в Producer
	writer *bufio.Writer
}

func NewProducer(filename string) (*Producer, error) {
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {

		return nil, fmt.Errorf("error opening file: %w", err)
	}

	return &Producer{
		file: file,
		// создаём новый Writer
		writer: bufio.NewWriter(file),
	}, nil
}
func (p *Producer) Close() error {
	return p.file.Close()
}

func (p *Producer) WriteEvent(event *models.Metrics) error {
	mutex.Lock()
	defer mutex.Unlock()
	data, err := json.Marshal(&event)
	if err != nil {
		return err
	}

	// записываем событие в буфер
	if _, err := p.writer.Write(data); err != nil {
		return err
	}

	// добавляем перенос строки
	if err := p.writer.WriteByte('\n'); err != nil {
		return err
	}

	// записываем буфер в файл
	return p.writer.Flush()
}
func RestoreFromBackup(storage *MemStorage, flagFileStoragePath string) {
	consumer, err := NewConsumer(flagFileStoragePath)
	if err != nil {
		logger.Sugar.Error("Failed to create consumer: ", err)
		return
	}
	defer func(consumer *Consumer) {
		err = consumer.Close()
		if err != nil {
			logger.Sugar.Error("Failed to close consumer: ", err)
		}
	}(consumer)

	event, err := consumer.ReadEvent()
	if err != nil {
		logger.Sugar.Error("Failed to read event: ", err)
		return
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
	for m := range storage.GaugeSlice() {
		logger.Sugar.Infoln(storage.GaugeSlice()[m].Name, storage.GaugeSlice()[m].Value)
	}
	for m := range storage.CounterSlice() {
		logger.Sugar.Infoln(storage.CounterSlice()[m].Name, storage.CounterSlice()[m].Value)
	}
}

func RunPeriodicBackup(storage *MemStorage, producer *Producer, interval time.Duration, stop <-chan struct{}, flagFileStoragePath string) {

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := performBackup(storage, producer, flagFileStoragePath); err != nil {
				logger.Sugar.Error("Periodic backup failed: ", err)
			} else {
				logger.Sugar.Info("Periodic backup complete")
			}
		case <-stop:
			return
		}
	}
}

/*
func RunPeriodicPostgresBackup(storage *repository.MemStorage, pool *pgxpool.Pool, interval time.Duration, stop <-chan struct{}) {

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := performBackupToPostgres(storage, pool, sugar); err != nil {
					sugar.Error("Periodic backup to postgres failed: ", err)
				} else {
					sugar.Info("Periodic backup to postgres complete")
				}
			case <-stop:
				return
			}
		}
	}
*/
func RunFinalBackup(storage *MemStorage, producer *Producer, stop <-chan struct{}, wg *sync.WaitGroup, flagFileStoragePath string) {
	defer wg.Done()
	<-stop

	logger.Sugar.Info("Starting final backup...")

	if err := performBackup(storage, producer, flagFileStoragePath); err != nil {
		logger.Sugar.Error("Final backup failed: ", err)
	}
	logger.Sugar.Info("Final backup completed")
}

func performBackup(storage *MemStorage, producer *Producer, flagFileStoragePath string) error {
	var producerM sync.Mutex

	producerM.Lock()
	defer producerM.Unlock()

	err := producer.Close()
	if err != nil {
		logger.Sugar.Error("Failed to close producer: ", err)
	}
	oldBackup := flagFileStoragePath + "_old"
	if err := os.Remove(oldBackup); err != nil && !os.IsNotExist(err) {
		log.Println("No old backup file: ", err)
	}

	err = os.Rename(flagFileStoragePath, oldBackup)
	if err != nil {
		log.Fatal(err)
	}
	if err := os.Remove(flagFileStoragePath); err != nil && !os.IsNotExist(err) {
		return err
	}

	// Создаем новый продюсер (файл будет создан заново)
	newProducer, err := NewProducer(flagFileStoragePath)
	if err != nil {
		return err
	}

	// Бэкап gauge метрик
	for _, gauge := range storage.GaugeSlice() {
		backup := models.Metrics{
			ID:    gauge.Name,
			MType: "gauge",
			Value: &gauge.Value,
		}
		if err := newProducer.WriteEvent(&backup); err != nil {
			return err
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
			return err
		}
	}

	return nil
}

func performBackupToPostgres(storage *MemStorage, dbPool *pgxpool.Pool) error {
	var mu sync.Mutex
	ctx := context.Background()

	mu.Lock()
	defer mu.Unlock()

	// Начинаем транзакцию
	tx, err := dbPool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Очищаем существующие данные (опционально, можно пропустить если нужно накапливать)
	if _, err := tx.Exec(ctx, "TRUNCATE TABLE gauges, counters"); err != nil {
		return fmt.Errorf("failed to truncate tables: %w", err)
	}

	// Бэкап gauge метрик
	for _, gauge := range storage.GaugeSlice() {
		_, err := tx.Exec(ctx,
			`INSERT INTO gauges (name, value) 
			VALUES ($1, $2)
			ON CONFLICT (name) DO UPDATE 
			SET value = EXCLUDED.value, updated_at = NOW()`,
			gauge.Name,
			gauge.Value,
		)
		if err != nil {
			return fmt.Errorf("failed to backup gauge %s: %w", gauge.Name, err)
		}
	}

	// Бэкап counter метрик
	for _, counter := range storage.CounterSlice() {
		_, err := tx.Exec(ctx,
			`INSERT INTO counters (name, value) 
			VALUES ($1, $2)
			ON CONFLICT (name) DO UPDATE 
			SET value = counters.value + EXCLUDED.value, updated_at = NOW()`,
			counter.Name,
			counter.Value,
		)
		if err != nil {
			return fmt.Errorf("failed to backup counter %s: %w", counter.Name, err)
		}
	}

	// Фиксируем транзакцию
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	logger.Sugar.Info("Successfully backed up metrics to PostgreSQL")
	return nil
}

func WaitForShutdown(stop chan<- struct{}) {
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	sig := <-shutdown
	logger.Sugar.Infof("Received signal: %v. Shutting down...", sig)
	close(stop)

}
