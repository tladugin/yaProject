package repository

import (
	"encoding/json"
	"github.com/tladugin/yaProject.git/internal/logger"
	"github.com/tladugin/yaProject.git/internal/models"
	"os"
	"strings"
	"testing"
	"time"
)

// TestMain инициализирует логгер для всех тестов
func TestMain(m *testing.M) {
	// Инициализируем логгер для тестов
	var err error
	logger.Sugar, err = logger.InitLogger()
	if err != nil {
		// В тестах можно использовать nil логгер или заглушку
		logger.Sugar = nil
	}

	code := m.Run()
	os.Exit(code)
}

// TestNewConsumer тестирует создание Consumer
func TestNewConsumer(t *testing.T) {
	// Создаем временный файл
	tmpfile, err := os.CreateTemp("", "test_consumer")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	// Записываем тестовые данные - используем правильные теги JSON из models.Metrics
	testData := `{"id":"test","type":"gauge","value":1.5}` + "\n"
	if _, err := tmpfile.WriteString(testData); err != nil {
		t.Fatalf("Failed to write test data: %v", err)
	}
	tmpfile.Close()

	// Тестируем создание Consumer
	consumer, err := NewConsumer(tmpfile.Name())
	if err != nil {
		t.Fatalf("NewConsumer failed: %v", err)
	}
	defer consumer.Close()

	if consumer.file == nil {
		t.Error("Consumer file should not be nil")
	}
	if consumer.reader == nil {
		t.Error("Consumer reader should not be nil")
	}
}

// TestConsumer_ReadEvent тестирует чтение событий
func TestConsumer_ReadEvent(t *testing.T) {
	// Создаем временный файл с тестовыми данными
	tmpfile, err := os.CreateTemp("", "test_read_event")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	// Записываем несколько метрик с правильными именами полей JSON
	// Смотрим на теги json в models.Metrics структуре
	metrics := []string{
		`{"id":"gauge1","type":"gauge","value":1.5}` + "\n",
		`{"id":"counter1","type":"counter","delta":10}` + "\n",
	}

	for _, metric := range metrics {
		if _, err := tmpfile.WriteString(metric); err != nil {
			t.Fatalf("Failed to write test data: %v", err)
		}
	}
	tmpfile.Close()

	consumer, err := NewConsumer(tmpfile.Name())
	if err != nil {
		t.Fatalf("NewConsumer failed: %v", err)
	}
	defer consumer.Close()

	// Читаем первую метрику
	event1, err := consumer.ReadEvent()
	if err != nil {
		t.Fatalf("ReadEvent failed: %v", err)
	}

	if event1.ID != "gauge1" || event1.MType != "gauge" || *event1.Value != 1.5 {
		t.Errorf("First event mismatch: got ID=%s, Type=%s, Value=%v", event1.ID, event1.MType, *event1.Value)
	}

	// Читаем вторую метрику
	event2, err := consumer.ReadEvent()
	if err != nil {
		t.Fatalf("ReadEvent failed: %v", err)
	}

	if event2.ID != "counter1" || event2.MType != "counter" || *event2.Delta != 10 {
		t.Errorf("Second event mismatch: got ID=%s, Type=%s, Delta=%v", event2.ID, event2.MType, *event2.Delta)
	}
}

// TestConsumer_ReadEvent_EOF тестирует чтение при достижении конца файла
func TestConsumer_ReadEvent_EOF(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "test_eof")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	// Записываем одну метрику
	testData := `{"id":"test","type":"gauge","value":1.0}` + "\n"
	if _, err := tmpfile.WriteString(testData); err != nil {
		t.Fatalf("Failed to write test data: %v", err)
	}
	tmpfile.Close()

	consumer, err := NewConsumer(tmpfile.Name())
	if err != nil {
		t.Fatalf("NewConsumer failed: %v", err)
	}
	defer consumer.Close()

	// Читаем первую метрику
	event, err := consumer.ReadEvent()
	if err != nil {
		t.Fatalf("First ReadEvent should succeed: %v", err)
	}

	if event.ID != "test" {
		t.Errorf("Expected event ID 'test', got '%s'", event.ID)
	}

	// Вторая попытка чтения должна вернуть ошибку
	_, err = consumer.ReadEvent()
	if err == nil {
		t.Error("Second ReadEvent should return error on EOF")
	}
}

// TestNewProducer тестирует создание Producer
func TestNewProducer(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "test_producer")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	producer, err := NewProducer(tmpfile.Name())
	if err != nil {
		t.Fatalf("NewProducer failed: %v", err)
	}
	defer producer.Close()

	if producer.file == nil {
		t.Error("Producer file should not be nil")
	}
	if producer.writer == nil {
		t.Error("Producer writer should not be nil")
	}
}

// TestProducer_WriteEvent тестирует запись событий
func TestProducer_WriteEvent(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "test_write_event")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	producer, err := NewProducer(tmpfile.Name())
	if err != nil {
		t.Fatalf("NewProducer failed: %v", err)
	}

	// Тестовая метрика
	value := 2.5
	event := &models.Metrics{
		ID:    "test_gauge",
		MType: "gauge",
		Value: &value,
	}

	// Записываем событие
	err = producer.WriteEvent(event)
	if err != nil {
		t.Fatalf("WriteEvent failed: %v", err)
	}

	// Закрываем producer чтобы сбросить буфер
	producer.Close()

	// Проверяем что данные записаны
	data, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if len(data) == 0 {
		t.Error("File should not be empty after WriteEvent")
	}

	// Проверяем JSON валидность и правильность полей
	var decodedEvent models.Metrics
	if err := json.Unmarshal(data, &decodedEvent); err != nil {
		t.Errorf("Written data is not valid JSON: %v, data: %s", err, string(data))
	}

	if decodedEvent.ID != "test_gauge" {
		t.Errorf("Decoded event ID mismatch: %s", decodedEvent.ID)
	}
	if decodedEvent.MType != "gauge" {
		t.Errorf("Decoded event type mismatch: %s", decodedEvent.MType)
	}
}

// TestProducer_WriteEvent_CheckJSONFields тестирует что JSON содержит правильные поля
func TestProducer_WriteEvent_CheckJSONFields(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "test_json_fields")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	producer, err := NewProducer(tmpfile.Name())
	if err != nil {
		t.Fatalf("NewProducer failed: %v", err)
	}

	// Тестовая метрика
	value := 3.14
	delta := int64(100)
	event := &models.Metrics{
		ID:    "test_metric",
		MType: "gauge",
		Value: &value,
		Delta: &delta,
	}

	// Записываем событие
	err = producer.WriteEvent(event)
	if err != nil {
		t.Fatalf("WriteEvent failed: %v", err)
	}
	producer.Close()

	// Читаем записанные данные
	data, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	jsonStr := string(data)

	// Проверяем наличие ожидаемых полей в JSON
	if !strings.Contains(jsonStr, `"id":"test_metric"`) {
		t.Errorf("JSON should contain id field: %s", jsonStr)
	}
	if !strings.Contains(jsonStr, `"type":"gauge"`) {
		t.Errorf("JSON should contain type field: %s", jsonStr)
	}
	if !strings.Contains(jsonStr, `"value":3.14`) {
		t.Errorf("JSON should contain value field: %s", jsonStr)
	}
}

// TestRestoreFromBackup тестирует восстановление из бэкапа
func TestRestoreFromBackup(t *testing.T) {
	// Создаем временный файл бэкапа
	tmpfile, err := os.CreateTemp("", "test_restore")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	// Записываем тестовые данные с правильными именами полей JSON
	backupData := `{"id":"gauge_restore","type":"gauge","value":3.14}` + "\n" +
		`{"id":"counter_restore","type":"counter","delta":42}` + "\n"

	if _, err := tmpfile.WriteString(backupData); err != nil {
		t.Fatalf("Failed to write backup data: %v", err)
	}
	tmpfile.Close()

	storage := NewMemStorage()
	RestoreFromBackup(storage, tmpfile.Name())

	// Проверяем восстановленные gauge метрики
	foundGauge := false
	for _, gauge := range storage.GaugeSlice() {
		if gauge.Name == "gauge_restore" && gauge.Value == 3.14 {
			foundGauge = true
			break
		}
	}
	if !foundGauge {
		t.Error("Gauge metric not restored correctly")
	}

	// Проверяем восстановленные counter метрики
	foundCounter := false
	for _, counter := range storage.CounterSlice() {
		if counter.Name == "counter_restore" && counter.Value == 42 {
			foundCounter = true
			break
		}
	}
	if !foundCounter {
		t.Error("Counter metric not restored correctly")
	}
}

// TestRestoreFromBackup_EmptyFile тестирует восстановление из пустого файла
func TestRestoreFromBackup_EmptyFile(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "test_restore_empty")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())
	tmpfile.Close()

	storage := NewMemStorage()
	// Не должно паниковать
	RestoreFromBackup(storage, tmpfile.Name())

	if len(storage.GaugeSlice()) != 0 || len(storage.CounterSlice()) != 0 {
		t.Error("Storage should be empty after restoring from empty file")
	}
}

// TestPerformBackup тестирует выполнение бэкапа
func TestPerformBackup(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "test_perform")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())
	defer func() {
		// Очищаем старый бэкап если он создался
		os.Remove(tmpfile.Name() + "_old")
	}()

	storage := NewMemStorage()
	producer, err := NewProducer(tmpfile.Name())
	if err != nil {
		t.Fatalf("NewProducer failed: %v", err)
	}

	// Добавляем тестовые данные
	storage.AddGauge("backup_gauge", 7.77)
	storage.AddCounter("backup_counter", 777)

	// Выполняем бэкап
	err = performBackup(storage, producer, tmpfile.Name())
	if err != nil {
		t.Fatalf("performBackup failed: %v", err)
	}

	// Проверяем что файл создан
	info, err := os.Stat(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to stat backup file: %v", err)
	}

	if info.Size() == 0 {
		t.Error("Backup file should not be empty")
	}
}

// TestMemStorage_Integration тестирует интеграцию Consumer/Producer с MemStorage
func TestMemStorage_Integration(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "test_integration")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())
	defer func() {
		os.Remove(tmpfile.Name() + "_old")
	}()

	storage := NewMemStorage()
	producer, err := NewProducer(tmpfile.Name())
	if err != nil {
		t.Fatalf("NewProducer failed: %v", err)
	}

	// Добавляем метрики
	storage.AddGauge("integration_gauge", 5.55)
	storage.AddCounter("integration_counter", 555)

	// Выполняем бэкап
	err = performBackup(storage, producer, tmpfile.Name())
	if err != nil {
		t.Fatalf("performBackup failed: %v", err)
	}
	producer.Close()

	// Создаем новое хранилище и восстанавливаем из бэкапа
	newStorage := NewMemStorage()
	RestoreFromBackup(newStorage, tmpfile.Name())

	// Проверяем восстановленные данные
	gaugeFound := false
	for _, gauge := range newStorage.GaugeSlice() {
		if gauge.Name == "integration_gauge" && gauge.Value == 5.55 {
			gaugeFound = true
			break
		}
	}
	if !gaugeFound {
		t.Error("Gauge metric not restored in integration test")
	}

	counterFound := false
	for _, counter := range newStorage.CounterSlice() {
		if counter.Name == "integration_counter" && counter.Value == 555 {
			counterFound = true
			break
		}
	}
	if !counterFound {
		t.Error("Counter metric not restored in integration test")
	}
}

// TestWaitForShutdown тестирует ожидание сигнала завершения
func TestWaitForShutdown(t *testing.T) {
	stop := make(chan struct{})

	// Запускаем WaitForShutdown в отдельной горутине
	go WaitForShutdown(stop)

	// Даем время на настройку signal.Notify
	time.Sleep(100 * time.Millisecond)

	// Проверяем что канал еще не закрыт
	select {
	case <-stop:
		t.Error("Stop channel should not be closed yet")
	default:
		// Ожидаемое поведение
	}
}

// TestBackupWithEmptyStorage тестирует бэкап пустого хранилища
func TestBackupWithEmptyStorage(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "test_empty_storage")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())
	defer func() {
		os.Remove(tmpfile.Name() + "_old")
	}()

	storage := NewMemStorage()
	producer, err := NewProducer(tmpfile.Name())
	if err != nil {
		t.Fatalf("NewProducer failed: %v", err)
	}

	// Выполняем бэкап пустого хранилища
	err = performBackup(storage, producer, tmpfile.Name())
	if err != nil {
		t.Fatalf("performBackup should not fail with empty storage: %v", err)
	}

	// Файл должен быть создан, но может быть пустым
	_, err = os.Stat(tmpfile.Name())
	if err != nil {
		t.Fatalf("Backup file should be created: %v", err)
	}
}

// TestConsumerClose тестирует закрытие Consumer
func TestConsumerClose(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "test_close")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	consumer, err := NewConsumer(tmpfile.Name())
	if err != nil {
		t.Fatalf("NewConsumer failed: %v", err)
	}

	// Закрытие должно работать без ошибок
	err = consumer.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

// TestProducerClose тестирует закрытие Producer
func TestProducerClose(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "test_producer_close")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	producer, err := NewProducer(tmpfile.Name())
	if err != nil {
		t.Fatalf("NewProducer failed: %v", err)
	}

	// Закрытие должно работать без ошибок
	err = producer.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}
