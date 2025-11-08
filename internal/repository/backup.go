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

// Consumer отвечает за чтение данных из файла бэкапа
type Consumer struct {
	file   *os.File      // Файл для чтения
	reader *bufio.Reader // Буферизованный reader для эффективного чтения
}

// NewConsumer создает новый экземпляр Consumer для чтения бэкапов
func NewConsumer(filename string) (*Consumer, error) {
	file, err := os.OpenFile(filename, os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}

	return &Consumer{
		file:   file,
		reader: bufio.NewReader(file), // Инициализация буферизованного чтения
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

	// Чтение данных до символа новой строки
	data, err := c.reader.ReadBytes('\n')
	if err != nil {
		return nil, err
	}

	// Декодирование JSON данных в структуру Metrics
	event := models.Metrics{}
	err = json.Unmarshal(data, &event)
	if err != nil {
		return nil, err
	}

	return &event, nil
}

// Producer отвечает за запись данных в файл бэкапа
type Producer struct {
	file   *os.File      // Файл для записи
	writer *bufio.Writer // Буферизованный writer для эффективной записи
}

// NewProducer создает новый экземпляр Producer для записи бэкапов
func NewProducer(filename string) (*Producer, error) {
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}

	return &Producer{
		file:   file,
		writer: bufio.NewWriter(file), // Инициализация буферизованной записи
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

	// Кодирование структуры в JSON
	data, err := json.Marshal(&event)
	if err != nil {
		return err
	}

	// Запись события в буфер
	if _, err := p.writer.Write(data); err != nil {
		return err
	}

	// Добавление переноса строки для разделения записей
	if err := p.writer.WriteByte('\n'); err != nil {
		return err
	}

	// Сброс буфера в файл
	return p.writer.Flush()
}

// RestoreFromBackup восстанавливает данные хранилища из файла бэкапа
func RestoreFromBackup(storage *MemStorage, flagFileStoragePath string) {
	// Создание потребителя для чтения бэкапа
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

	// Чтение первой записи
	event, err := consumer.ReadEvent()
	if err != nil {
		logger.Sugar.Error("Failed to read event: ", err)
		return
	}

	// Цикл чтения всех записей из бэкапа
	for event != nil {
		// Восстановление метрик в зависимости от типа
		switch event.MType {
		case "gauge":
			storage.AddGauge(event.ID, *event.Value)
		case "counter":
			storage.AddCounter(event.ID, *event.Delta)
		}

		// Чтение следующей записи
		event, err = consumer.ReadEvent()
		if err != nil {
			logger.Sugar.Info("Backup restore completed")
			break
		}
	}

	// Логирование восстановленных gauge метрик
	for m := range storage.GaugeSlice() {
		logger.Sugar.Infoln(storage.GaugeSlice()[m].Name, storage.GaugeSlice()[m].Value)
	}

	// Логирование восстановленных counter метрик
	for m := range storage.CounterSlice() {
		logger.Sugar.Infoln(storage.CounterSlice()[m].Name, storage.CounterSlice()[m].Value)
	}
}

// RunPeriodicBackup запускает периодическое создание бэкапов с заданным интервалом
func RunPeriodicBackup(storage *MemStorage, producer *Producer, interval time.Duration, stop <-chan struct{}, flagFileStoragePath string) {
	// Создание тикера для периодического выполнения
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Основной цикл периодического бэкапа
	for {
		select {
		case <-ticker.C:
			// Выполнение бэкапа по таймеру
			if err := performBackup(storage, producer, flagFileStoragePath); err != nil {
				logger.Sugar.Error("Periodic backup failed: ", err)
			} else {
				logger.Sugar.Info("Periodic backup complete")
			}
		case <-stop:
			// Завершение при получении сигнала остановки
			return
		}
	}
}

// RunFinalBackup выполняет финальный бэкап при завершении работы приложения
func RunFinalBackup(storage *MemStorage, producer *Producer, stop <-chan struct{}, wg *sync.WaitGroup, flagFileStoragePath string) {
	defer wg.Done()
	<-stop // Ожидание сигнала остановки

	logger.Sugar.Info("Starting final backup...")

	// Выполнение финального бэкапа
	if err := performBackup(storage, producer, flagFileStoragePath); err != nil {
		logger.Sugar.Error("Final backup failed: ", err)
	}
	logger.Sugar.Info("Final backup completed")
}

// performBackup выполняет фактическое создание бэкапа в файловую систему
func performBackup(storage *MemStorage, producer *Producer, flagFileStoragePath string) error {
	var producerM sync.Mutex

	producerM.Lock()
	defer producerM.Unlock()

	// Закрытие текущего продюсера
	err := producer.Close()
	if err != nil {
		logger.Sugar.Error("Failed to close producer: ", err)
	}

	// Удаление старого бэкапа если существует
	oldBackup := flagFileStoragePath + "_old"
	if err := os.Remove(oldBackup); err != nil && !os.IsNotExist(err) {
		log.Println("No old backup file: ", err)
	}

	// Переименование текущего бэкапа в старый
	err = os.Rename(flagFileStoragePath, oldBackup)
	if err != nil {
		log.Fatal(err)
	}

	// Удаление текущего файла бэкапа
	if err := os.Remove(flagFileStoragePath); err != nil && !os.IsNotExist(err) {
		return err
	}

	// Создание нового продюсера (файл будет создан заново)
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

// performBackupToPostgres выполняет бэкап метрик в PostgreSQL базу данных
func performBackupToPostgres(storage *MemStorage, dbPool *pgxpool.Pool) error {
	var mu sync.Mutex
	ctx := context.Background()

	mu.Lock()
	defer mu.Unlock()

	// Начинаем транзакцию для атомарности операций
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

// WaitForShutdown ожидает сигналов завершения работы приложения
func WaitForShutdown(stop chan<- struct{}) {
	// Канал для получения сигналов ОС
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Ожидание сигнала завершения
	sig := <-shutdown
	logger.Sugar.Infof("Received signal: %v. Shutting down...", sig)
	close(stop) // Отправка сигнала остановки всем горутинам
}
