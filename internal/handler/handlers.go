package handler

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tladugin/yaProject.git/internal/logger"
	"io"
	"log"

	"github.com/tladugin/yaProject.git/internal/models"
	"github.com/tladugin/yaProject.git/internal/repository"

	"net/http"
	"strconv"
	"strings"
)

// Конструкторы для создания обработчиков

// NewServerSync создает сервер с синхронным бэкапом
func NewServerSync(s *repository.MemStorage, p *repository.Producer) *ServerSync {
	return &ServerSync{
		storage:  s,
		producer: p,
	}
}

// NewServer создает базовый сервер без бэкапа
func NewServer(s *repository.MemStorage) *Server {
	return &Server{
		storage: s,
	}
}

// Структуры обработчиков

// Server - базовый обработчик для in-memory хранилища
type Server struct {
	storage *repository.MemStorage
}

// ServerPing - обработчик для проверки соединения с БД
type ServerPing struct {
	storage     *repository.MemStorage
	databaseDSN *string
}

// ServerDB - обработчик для работы с PostgreSQL
type ServerDB struct {
	storage        *repository.MemStorage
	connectionPool *pgxpool.Pool
	flagKey        *string
}

// NewServerDB создает обработчик для работы с базой данных
func NewServerDB(s *repository.MemStorage, p *pgxpool.Pool, k *string) *ServerDB {
	return &ServerDB{
		storage:        s,
		connectionPool: p,
		flagKey:        k,
	}
}

// NewServerPingDB создает обработчик для проверки доступности БД
func NewServerPingDB(s *repository.MemStorage, c *string) *ServerPing {
	return &ServerPing{
		storage:     s,
		databaseDSN: c,
	}
}

// ServerSync - обработчик с синхронным бэкапом в файл
type ServerSync struct {
	storage  *repository.MemStorage
	producer *repository.Producer
}

// MainPage отображает главную страницу со списком всех метрик
func (s *Server) MainPage(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "text/html; charset=utf-8")
	res.Header().Set("Content-Encoding", "gzip")
	res.Header().Set("Accept-Encoding", "gzip")

	// Генерация HTML страницы со списком метрик
	fmt.Fprint(res, "<html><body><ul>")

	// Отображение gauge метрик
	for m := range s.storage.GaugeSlice() {
		fmt.Fprintf(res, "<li>%s: %v</li>", s.storage.GaugeSlice()[m].Name, s.storage.GaugeSlice()[m].Value)
	}

	// Отображение counter метрик
	for m := range s.storage.CounterSlice() {
		fmt.Fprintf(res, "<li>%s: %v</li>", s.storage.CounterSlice()[m].Name, s.storage.CounterSlice()[m].Value)
	}

	fmt.Fprint(res, "</ul></body></html>")
}

// Обработчики для работы с PostgreSQL

// updateGaugePostgres обновляет gauge метрику в PostgreSQL
func (s *ServerDB) updateGaugePostgres(ctx context.Context, name string, value float64) error {
	_, err := s.connectionPool.Exec(ctx,
		`INSERT INTO gauge_metrics (name, value) VALUES ($1, $2) 
		 ON CONFLICT (name) DO UPDATE SET value = EXCLUDED.value`,
		name, value)
	if err != nil {
		return fmt.Errorf("error updating gauge_metrics: %v", err)
	}
	return nil
}

// updateCounterPostgres обновляет counter метрику в PostgreSQL
func (s *ServerDB) updateCounterPostgres(ctx context.Context, name string, delta int64) error {
	_, err := s.connectionPool.Exec(ctx,
		`INSERT INTO counter_metrics (name, value) VALUES ($1, $2) 
		 ON CONFLICT (name) DO UPDATE SET value = counter_metrics.value + EXCLUDED.value`,
		name, delta)
	if err != nil {
		return fmt.Errorf("error updating counter_metrics: %v", err)
	}
	return nil
}

// PostUpdatePostgres обрабатывает обновление метрик через JSON для PostgreSQL
func (s *ServerDB) PostUpdatePostgres(res http.ResponseWriter, req *http.Request) {
	ctx := context.Background()
	res.Header().Set("Content-Type", "application/json")
	res.Header().Set("Content-Encoding", "gzip")
	res.Header().Set("Accept-Encoding", "gzip")

	var metric models.Metrics
	if err := json.NewDecoder(req.Body).Decode(&metric); err != nil {
		http.Error(res, "Invalid JSON format", http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	// Валидация и обработка метрик в зависимости от типа
	switch metric.MType {
	case "gauge":
		if metric.Value == nil {
			http.Error(res, "Gauge value is required", http.StatusBadRequest)
			return
		}
		err := s.updateGaugePostgres(ctx, metric.ID, *metric.Value)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

	case "counter":
		if metric.Delta == nil {
			http.Error(res, "Counter delta is required", http.StatusBadRequest)
			return
		}
		err := s.updateCounterPostgres(ctx, metric.ID, *metric.Delta)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}

	default:
		http.Error(res, "Invalid metric type", http.StatusBadRequest)
		return
	}

	// Возвращаем обновленную метрику
	json.NewEncoder(res).Encode(metric)
}

// getGauge получает gauge метрику из PostgreSQL
func (s *ServerDB) getGauge(ctx context.Context, name string) (models.Metrics, error) {
	var value float64
	err := s.connectionPool.QueryRow(ctx,
		"SELECT value FROM gauge_metrics WHERE name = $1", name).Scan(&value)

	if err != nil {
		return models.Metrics{}, fmt.Errorf("error querying gauge_metrics: %v", err)
	}

	return models.Metrics{
		ID:    name,
		MType: "gauge",
		Value: &value,
	}, nil
}

// getCounter получает counter метрику из PostgreSQL
func (s *ServerDB) getCounter(ctx context.Context, name string) (models.Metrics, error) {
	var value int64
	err := s.connectionPool.QueryRow(ctx,
		"SELECT value FROM counter_metrics WHERE name = $1", name).Scan(&value)

	if err != nil {
		return models.Metrics{}, fmt.Errorf("error querying counter_metrics: %v", err)
	}

	return models.Metrics{
		ID:    name,
		MType: "counter",
		Delta: &value,
	}, nil
}

// PostValue обрабатывает запрос на получение значения метрики для PostgreSQL
func (s *ServerDB) PostValue(res http.ResponseWriter, req *http.Request) {
	ctx := context.Background()
	res.Header().Set("Content-Type", "application/json")
	res.Header().Set("Content-Encoding", "gzip")
	res.Header().Set("Accept-Encoding", "gzip")

	var metric models.Metrics
	if err := json.NewDecoder(req.Body).Decode(&metric); err != nil {
		http.Error(res, "Invalid JSON format", http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	if metric.ID == "" {
		http.Error(res, "Metric ID is required", http.StatusBadRequest)
		return
	}

	var result models.Metrics
	var err error

	// Получение метрики в зависимости от типа
	switch metric.MType {
	case "gauge":
		result, err = s.getGauge(ctx, metric.ID)
	case "counter":
		result, err = s.getCounter(ctx, metric.ID)
	default:
		http.Error(res, "Invalid metric type", http.StatusBadRequest)
		return
	}

	// Обработка ошибок при получении метрики
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(res, "Metric not found", http.StatusNotFound)
		} else {
			http.Error(res, err.Error(), http.StatusNotFound)
		}
		return
	}

	json.NewEncoder(res).Encode(result)
}

// UpdatesGaugesBatchPostgres обрабатывает пакетное обновление метрик для PostgreSQL
func (s *ServerDB) UpdatesGaugesBatchPostgres(res http.ResponseWriter, req *http.Request) {
	ctx := context.Background()
	res.Header().Set("Content-Type", "application/json")
	res.Header().Set("Content-Encoding", "gzip")
	res.Header().Set("Accept-Encoding", "gzip")

	// Читаем тело запроса один раз
	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(res, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	// Проверка хеша (если ключ установлен)
	if req.Header.Get("HashSHA256") != "" && s.flagKey != nil && *s.flagKey != "" {
		bytesKey := []byte(*s.flagKey)
		hash := sha256.Sum256(append(bytesKey, bodyBytes...))
		hashHeaderServer := hex.EncodeToString(hash[:])

		if hashHeaderServer != req.Header.Get("HashSHA256") {
			res.Header().Set("HashSHA256", hashHeaderServer)
			http.Error(res, "Invalid hash header", http.StatusBadRequest)
			return
		}
		res.Header().Set("HashSHA256", hashHeaderServer)
	}

	// Декодируем JSON из сохранённых байтов
	var metrics []models.Metrics
	if err := json.Unmarshal(bodyBytes, &metrics); err != nil {
		logger.Sugar.Info("could not decode metrics json")
		http.Error(res, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	// Собираем названия метрик для аудита
	var metricNames []string
	for _, metric := range metrics {
		metricNames = append(metricNames, metric.ID)
	}

	// Начинаем транзакцию для атомарности операций
	tx, err := s.connectionPool.Begin(ctx)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(ctx)

	// Подготавливаем statement для пакетного обновления gauge метрик
	stmtGauge, err := tx.Prepare(ctx, "batch_update_gauge",
		`INSERT INTO gauge_metrics (name, value) VALUES ($1, $2) 
		 ON CONFLICT (name) DO UPDATE SET value = EXCLUDED.value`)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	// Подготавливаем statement для пакетного обновления counter метрик
	stmtCounter, err := tx.Prepare(ctx, "batch_update_counter",
		`INSERT INTO counter_metrics (name, value) VALUES ($1, $2) 
		 ON CONFLICT (name) DO UPDATE SET value = counter_metrics.value + EXCLUDED.value`)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	// Выполняем пакетное обновление метрик
	for _, value := range metrics {
		var err error
		switch value.MType {
		case "gauge":
			_, err = tx.Exec(ctx, stmtGauge.SQL, value.ID, value.Value)
		case "counter":
			_, err = tx.Exec(ctx, stmtCounter.SQL, value.ID, value.Delta)
		default:
			http.Error(res, "Unknown metric type", http.StatusBadRequest)
			return
		}

		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Фиксируем транзакцию
	if err := tx.Commit(ctx); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}

	// Добавляем данные для аудита в контекст и сохраняем обновленный запрос
	ip := getIPAddress(req)
	updatedReq := WithAuditData(req, metricNames, ip)

	// Сохраняем обновленный контекст в оригинальный запрос
	*req = *updatedReq

	res.WriteHeader(http.StatusOK)
}

// UpdatesGaugesBatch обрабатывает пакетное обновление метрик для in-memory хранилища
func (s *Server) UpdatesGaugesBatch(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "application/json")
	res.Header().Set("Content-Encoding", "gzip")
	res.Header().Set("Accept-Encoding", "gzip")

	// Читаем тело запроса один раз
	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(res, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	// Декодируем JSON из сохранённых байтов
	var metrics []models.Metrics
	if err := json.Unmarshal(bodyBytes, &metrics); err != nil {
		logger.Sugar.Info("could not decode metrics json")
		http.Error(res, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	// Собираем названия метрик для аудита
	var metricNames []string
	for _, value := range metrics {
		metricNames = append(metricNames, value.ID)
	}

	// Выполняем пакетное обновление в in-memory хранилище
	for _, value := range metrics {
		switch value.MType {
		case "gauge":
			s.storage.AddGauge(value.ID, *value.Value)
		case "counter":
			s.storage.AddCounter(value.ID, *value.Delta)
		default:
			http.Error(res, "Unknown metric type", http.StatusBadRequest)
			return
		}
	}

	// Добавляем данные для аудита в контекст и сохраняем обновленный запрос
	ip := getIPAddress(req)
	updatedReq := WithAuditData(req, metricNames, ip)

	// Сохраняем обновленный контекст в оригинальный запрос
	*req = *updatedReq

	res.WriteHeader(http.StatusOK)
}

// PostUpdateSyncBackup обрабатывает обновление метрик с синхронным бэкапом в файл
func (s *ServerSync) PostUpdateSyncBackup(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "application/json")
	res.Header().Set("Content-Encoding", "gzip")
	res.Header().Set("Accept-Encoding", "gzip")

	var decodedMetrics models.Metrics
	var encodedMetrics models.Metrics
	encoder := json.NewEncoder(res)
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&decodedMetrics)
	if err != nil {
		http.Error(res, "Wrong decoding", http.StatusNotAcceptable)
		return
	}
	if decodedMetrics.ID == "" {
		http.Error(res, "Wrong metric ID", http.StatusNotAcceptable)
		return
	}
	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			http.Error(res, "Error closing body", http.StatusInternalServerError)
		}
	}(req.Body)

	// Обработка метрик в зависимости от типа
	switch decodedMetrics.MType {
	case "gauge":
		if decodedMetrics.Value == nil {
			http.Error(res, "No gauge value", http.StatusNotAcceptable)
			return
		}
		// Обновление в памяти и запись в бэкап
		s.storage.AddGauge(decodedMetrics.ID, *decodedMetrics.Value)

		encodedMetrics.ID = decodedMetrics.ID
		encodedMetrics.MType = "gauge"
		encodedMetrics.Value = decodedMetrics.Value

		// Синхронная запись в файл бэкапа
		if err = s.producer.WriteEvent(&encodedMetrics); err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
		}

		err = encoder.Encode(encodedMetrics)

	case "counter":
		if decodedMetrics.Delta == nil {
			http.Error(res, "No counter delta", http.StatusNotAcceptable)
			return
		}
		s.storage.AddCounter(decodedMetrics.ID, *decodedMetrics.Delta)

		encodedMetrics.ID = decodedMetrics.ID
		encodedMetrics.MType = "counter"
		encodedMetrics.Delta = decodedMetrics.Delta

		// Синхронная запись в файл бэкапа
		if err = s.producer.WriteEvent(&encodedMetrics); err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
		}

		err = encoder.Encode(encodedMetrics)

	default:
		http.Error(res, "Wrong metric type", http.StatusNotAcceptable)
		return
	}
}

// PostUpdate обрабатывает обновление метрик без бэкапа (асинхронный режим)
func (s *Server) PostUpdate(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "application/json")
	res.Header().Set("Content-Encoding", "gzip")
	res.Header().Set("Accept-Encoding", "gzip")

	var decodedMetrics models.Metrics
	var encodedMetrics models.Metrics
	encoder := json.NewEncoder(res)
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&decodedMetrics)
	if err != nil {
		http.Error(res, "Wrong decoding", http.StatusNotAcceptable)
		return
	}
	if decodedMetrics.ID == "" {
		http.Error(res, "Wrong metric ID", http.StatusNotAcceptable)
		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			http.Error(res, "Error closing body", http.StatusInternalServerError)
		}
	}(req.Body)

	switch decodedMetrics.MType {
	case "gauge":
		if decodedMetrics.Value == nil {
			http.Error(res, "No gauge value", http.StatusNotAcceptable)
			return
		}
		s.storage.AddGauge(decodedMetrics.ID, *decodedMetrics.Value)

		encodedMetrics.ID = decodedMetrics.ID
		encodedMetrics.MType = "gauge"
		encodedMetrics.Value = decodedMetrics.Value

		err := encoder.Encode(encodedMetrics)
		if err != nil {
			return
		}

	case "counter":
		if decodedMetrics.Delta == nil {
			http.Error(res, "No counter delta", http.StatusNotAcceptable)
			return
		}
		s.storage.AddCounter(decodedMetrics.ID, *decodedMetrics.Delta)

		encodedMetrics.ID = decodedMetrics.ID
		encodedMetrics.MType = "counter"
		encodedMetrics.Delta = decodedMetrics.Delta

		encoder.Encode(encodedMetrics)

	default:
		http.Error(res, "Wrong metric type", http.StatusNotAcceptable)
		return
	}
}

// PostValue обрабатывает запрос на получение значения метрики для in-memory хранилища
func (s *Server) PostValue(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "application/json")
	res.Header().Set("Content-Encoding", "gzip")
	res.Header().Set("Accept-Encoding", "gzip")

	var decodedMetrics models.Metrics
	var encodedMetrics models.Metrics
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&decodedMetrics)
	if err != nil {
		http.Error(res, "Wrong decoding", http.StatusNotAcceptable)
		return
	}
	if decodedMetrics.ID == "" {
		http.Error(res, "Wrong metric ID", http.StatusNotAcceptable)
		return
	}
	defer req.Body.Close()
	encoder := json.NewEncoder(res)

	// Поиск и возврат метрики в зависимости от типа
	switch decodedMetrics.MType {
	case "gauge":
		getCheck := false
		for i, m := range s.storage.GaugeSlice() {
			if m.Name == decodedMetrics.ID {
				getCheck = true
				encodedMetrics.MType = "gauge"
				encodedMetrics.ID = s.storage.GaugeSlice()[i].Name
				encodedMetrics.Value = &s.storage.GaugeSlice()[i].Value
				encoder.Encode(encodedMetrics)
			}
		}
		if !getCheck {
			http.Error(res, "No metric found", http.StatusNotFound)
			return
		}
	case "counter":
		getCheck := false
		for i, m := range s.storage.CounterSlice() {
			if m.Name == decodedMetrics.ID {
				getCheck = true
				encodedMetrics.MType = "counter"
				encodedMetrics.ID = s.storage.CounterSlice()[i].Name
				encodedMetrics.Delta = &s.storage.CounterSlice()[i].Value
				encoder.Encode(encodedMetrics)
			}
		}
		if !getCheck {
			http.Error(res, "No metric found", http.StatusNotFound)
			return
		}
	default:
		http.Error(res, "Wrong metric type", http.StatusNotAcceptable)
		return
	}
}

// PostHandler обрабатывает обновление метрик через URL параметры
func (s *Server) PostHandler(res http.ResponseWriter, req *http.Request) {
	// Разбор URL для получения параметров метрики
	parts := strings.Split(req.URL.Path, "/")
	if len(parts) != 5 {
		http.Error(res, "Invalid URL", http.StatusBadRequest)
	}
	metric := chi.URLParam(req, "metric")
	name := chi.URLParam(req, "name")
	value := chi.URLParam(req, "value")

	// Обработка метрик в зависимости от типа
	switch metric {
	case "gauge":
		partFloat, Error := strconv.ParseFloat(value, 64)
		if Error != nil {
			http.Error(res, "Invalid metric value", http.StatusBadRequest)
			return
		}
		s.storage.AddGauge(name, partFloat)

	case "counter":
		partInt, Error := strconv.ParseInt(value, 0, 64)
		if Error != nil {
			http.Error(res, "Invalid metric value", http.StatusBadRequest)
			return
		}
		s.storage.AddCounter(name, partInt)

	default:
		http.Error(res, "Invalid metric value", http.StatusBadRequest)
	}
}

// GetHandler обрабатывает получение метрик через URL параметры
func (s *Server) GetHandler(res http.ResponseWriter, req *http.Request) {
	metric := chi.URLParam(req, "metric")
	name := chi.URLParam(req, "name")

	// Поиск и возврат метрики в зависимости от типа
	switch metric {
	case "gauge":
		getCheck := false
		for i, m := range s.storage.GaugeSlice() {
			if m.Name == name {
				getCheck = true
				fmt.Fprint(res, s.storage.GaugeSlice()[i].Value)
			}
		}
		if !getCheck {
			http.Error(res, "No metric found", http.StatusNotFound)
		}
	case "counter":
		getCheck := false
		for i, m := range s.storage.CounterSlice() {
			if m.Name == name {
				getCheck = true
				fmt.Fprint(res, s.storage.CounterSlice()[i].Value)
			}
		}
		if !getCheck {
			http.Error(res, "No metric found", http.StatusNotFound)
		}
	default:
		http.Error(res, "Invalid metric type", http.StatusNotFound)
	}
}

// GetPing проверяет доступность базы данных
func (s *ServerPing) GetPing(res http.ResponseWriter, req *http.Request) {
	pool, _, cancel, err := repository.GetConnection(*s.databaseDSN)

	if err != nil {
		log.Printf("Connection error: %v", err)
		http.Error(res, "Connection error", http.StatusInternalServerError)

	} else {
		res.WriteHeader(http.StatusOK)
	}
	defer cancel()
	defer pool.Close()
}
