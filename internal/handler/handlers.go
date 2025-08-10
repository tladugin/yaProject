package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tladugin/yaProject.git/internal/model"
	"github.com/tladugin/yaProject.git/internal/repository"
	"log"

	"io"

	"net/http"
	"strconv"
	"strings"
)

func NewServerSync(s *repository.MemStorage, p *Producer) *ServerSync {
	return &ServerSync{
		storage:  s,
		producer: p,
	}

}

func NewServer(s *repository.MemStorage) *Server {
	return &Server{
		storage: s,
	}

}

type Server struct {
	storage *repository.MemStorage
}

type ServerPing struct {
	storage     *repository.MemStorage
	databaseDSN *string
}
type ServerDB struct {
	storage        *repository.MemStorage
	connectionPool *pgxpool.Pool
}

func NewServerDB(s *repository.MemStorage, p *pgxpool.Pool) *ServerDB {
	return &ServerDB{
		storage:        s,
		connectionPool: p,
	}

}

func NewServerPingDB(s *repository.MemStorage, c *string) *ServerPing {
	return &ServerPing{
		storage:     s,
		databaseDSN: c,
	}

}

type ServerSync struct {
	storage  *repository.MemStorage
	producer *Producer
}

func (s *Server) MainPage(res http.ResponseWriter, req *http.Request) {

	res.Header().Set("Content-Type", "text/html; charset=utf-8")
	res.Header().Set("Content-Encoding", "gzip")
	res.Header().Set("Accept-Encoding", "gzip")

	fmt.Fprint(res, "<html><body><ul>")
	for m := range s.storage.GaugeSlice() {

		fmt.Fprintf(res, "<li>%s: %v</li>", s.storage.GaugeSlice()[m].Name, s.storage.GaugeSlice()[m].Value)

		//res.Header().Set("Content-Type", "application/json")
		//json.NewEncoder(res).Encode(s.storage.GaugeSlice()[m])
		//res.WriteHeader(http.StatusOK)
	}

	for m := range s.storage.CounterSlice() {

		fmt.Fprintf(res, "<li>%s: %v</li>", s.storage.CounterSlice()[m].Name, s.storage.CounterSlice()[m].Value)

		//res.Header().Set("Content-Type", "application/json")
		//json.NewEncoder(res).Encode(s.storage.CounterSlice()[m])
		//res.WriteHeader(http.StatusOK)

	}
	fmt.Fprint(res, "</ul></body></html>")
}

// postgres handlers! --->
func (s *ServerDB) updateGaugePostgres(ctx context.Context, name string, value float64) error {

	_, err := s.connectionPool.Exec(ctx,
		`INSERT INTO gauge_metrics (name, value) VALUES ($1, $2) ON CONFLICT (name) DO UPDATE SET value = EXCLUDED.value`, name, value)
	if err != nil {
		println("update gauges error: " + err.Error())
	}
	return err
}
func (s *ServerDB) updateCounterPostgres(ctx context.Context, name string, delta int64) error {
	_, err := s.connectionPool.Exec(ctx,
		`INSERT INTO counter_metrics (name, value) VALUES ($1, $2) ON CONFLICT (name) DO UPDATE SET value = counter_metrics.value + EXCLUDED.value`, name, delta)
	return err
}

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

	/*if metric.ID == "" {
		http.Error(res, "Metric ID is required", http.StatusBadRequest)
		return
	}

	*/

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
func (s *ServerDB) getGauge(ctx context.Context, name string) (models.Metrics, error) {
	var value float64
	err := s.connectionPool.QueryRow(ctx,
		"SELECT value FROM gauge_metrics WHERE name = $1", name).Scan(&value)

	if err != nil {
		return models.Metrics{}, err
	}

	return models.Metrics{
		ID:    name,
		MType: "gauge",
		Value: &value,
	}, nil
}

func (s *ServerDB) getCounter(ctx context.Context, name string) (models.Metrics, error) {
	var value int64
	err := s.connectionPool.QueryRow(ctx,
		"SELECT value FROM counter_metrics WHERE name = $1", name).Scan(&value)

	if err != nil {
		return models.Metrics{}, err
	}

	return models.Metrics{
		ID:    name,
		MType: "counter",
		Delta: &value,
	}, nil
}

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

	switch metric.MType {
	case "gauge":
		result, err = s.getGauge(ctx, metric.ID)
	case "counter":
		result, err = s.getCounter(ctx, metric.ID)
	default:
		http.Error(res, "Invalid metric type", http.StatusBadRequest)
		return
	}

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(res, "Metric not found", http.StatusNotFound)
		}
		return
	}

	json.NewEncoder(res).Encode(result)
}
func (s *ServerDB) UpdatesGaugesBatchPostgres(res http.ResponseWriter, req *http.Request) {
	ctx := context.Background()
	res.Header().Set("Content-Type", "application/json")
	res.Header().Set("Content-Encoding", "gzip")
	res.Header().Set("Accept-Encoding", "gzip")

	var metrics []models.Metrics

	if err := json.NewDecoder(req.Body).Decode(&metrics); err != nil {
		http.Error(res, "Invalid JSON format", http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	// Начинаем транзакцию
	tx, err := s.connectionPool.Begin(ctx)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
	defer tx.Rollback(ctx)

	// Подготавливаем statement для пакетного обновления
	stmtGauge, err := tx.Prepare(ctx, "batch_update_gauge",
		`INSERT INTO gauge_metrics (name, value) VALUES ($1, $2) ON CONFLICT (name) DO UPDATE SET value = EXCLUDED.value`)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}

	stmtCounter, err := tx.Prepare(ctx, "batch_update_counter",
		`INSERT INTO counter_metrics (name, value) VALUES ($1, $2) ON CONFLICT (name) DO UPDATE SET value = counter_metrics.value + EXCLUDED.value`)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}

	// Выполняем пакетное обновление
	for _, value := range metrics {
		switch value.MType {
		case "gauge":
			_, err := tx.Exec(ctx, stmtGauge.SQL, value.ID, value.Value)
			if err != nil {
				http.Error(res, err.Error(), http.StatusInternalServerError)
			}
		case "counter":
			_, err := tx.Exec(ctx, stmtCounter.SQL, value.ID, value.Delta)
			if err != nil {
				http.Error(res, err.Error(), http.StatusInternalServerError)
			}
		}

	}

	// Фиксируем транзакцию
	if err := tx.Commit(ctx); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}

}

// postgres handlers! ^----
func (s *ServerSync) PostUpdateSyncBackup(res http.ResponseWriter, req *http.Request) {
	//metricCounter += 1

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
	switch decodedMetrics.MType {
	case "gauge":
		//println(decodedMetrics.ID)
		if decodedMetrics.Value == nil {
			http.Error(res, "No gauge value", http.StatusNotAcceptable)
			return
		}
		s.storage.AddGauge(decodedMetrics.ID, *decodedMetrics.Value)

		encodedMetrics.ID = decodedMetrics.ID
		encodedMetrics.MType = "gauge"
		encodedMetrics.Value = decodedMetrics.Value

		if err = s.producer.WriteEvent(&encodedMetrics); err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
		}

		//fmt.Println(encodedMetrics.ID, encodedMetrics.MType, encodedMetrics.Value)

		err = encoder.Encode(encodedMetrics)
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

		//fmt.Println(encodedMetrics.ID, encodedMetrics.MType, encodedMetrics.Delta)

		if err = s.producer.WriteEvent(&encodedMetrics); err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
		}

		err = encoder.Encode(encodedMetrics)
		if err != nil {
			return
		}

	default:
		http.Error(res, "Wrong metric type", http.StatusNotAcceptable)
		return

	}

}
func (s *Server) PostUpdate(res http.ResponseWriter, req *http.Request) {
	//metricCounter += 1

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
		//println(decodedMetrics.ID)
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

		err := encoder.Encode(encodedMetrics)
		if err != nil {
			return
		}

	default:
		http.Error(res, "Wrong metric type", http.StatusNotAcceptable)
		return

	}

}
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

func (s *Server) PostHandler(res http.ResponseWriter, req *http.Request) {

	//res.Header().Set("Content-Encoding", "gzip")
	//res.Header().Set("Accept-Encoding", "gzip")

	//params := chi.URLParam(req, "URL")
	parts := strings.Split(req.URL.Path, "/")
	if len(parts) != 5 {
		http.Error(res, "Invalid URL", http.StatusBadRequest)
	}
	metric := chi.URLParam(req, "metric")
	//println(metric)
	name := chi.URLParam(req, "name")
	//println(name)
	value := chi.URLParam(req, "value")
	//println(value)

	switch metric {

	case "gauge":
		//if partFloat, err := strconv. ParseFloat(v, 64); err == nil { fmt. Printf("%T, %v\n", s, s)}
		partFloat, Error := strconv.ParseFloat(value, 64)
		//fmt.Printf("%f", partFloat)
		if Error != nil {
			http.Error(res, "Invalid metric value", http.StatusBadRequest)
			return
		}

		s.storage.AddGauge(name, partFloat)
		//println(s.storage.GaugeSlice())
		//res.WriteHeader(http.StatusOK)

	case "counter":
		partInt, Error := strconv.ParseInt(value, 0, 64)
		if Error != nil {
			http.Error(res, "Invalid metric value", http.StatusBadRequest)
			return
		}

		s.storage.AddCounter(name, partInt)
		//res.WriteHeader(http.StatusOK)

		//fmt.Println(s.storage.CounterSlice())
	default:
		http.Error(res, "Invalid metric value", http.StatusBadRequest)
	}

}
func (s *Server) GetHandler(res http.ResponseWriter, req *http.Request) {

	//res.Header().Set("Content-Encoding", "gzip")
	//res.Header().Set("Accept-Encoding", "gzip")

	metric := chi.URLParam(req, "metric")
	//println(metric)
	name := chi.URLParam(req, "name")
	//println(name)

	switch metric {
	case "gauge":
		getCheck := false
		for i, m := range s.storage.GaugeSlice() {
			if m.Name == name {
				getCheck = true
				//fmt.Fprintf(res, "%s %f", s.storage.GaugeSlice()[i].Name, s.storage.GaugeSlice()[i].Value)
				fmt.Fprint(res, s.storage.GaugeSlice()[i].Value)
				//fmt.Println(s.storage.GaugeSlice()[i].Value)
				//res.WriteHeader(http.StatusOK)

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
				//println(m.Name)
				//println(m.Value)
				//res.Header().Set("Content-Type", "application/json")
				//json.NewEncoder(res).Encode(s.storage.GaugeSlice()[i])
				//fmt.Fprintf(res, "%s %d", s.storage.CounterSlice()[i].Name, s.storage.CounterSlice()[i].Value)
				fmt.Fprint(res, s.storage.CounterSlice()[i].Value)

				//res.WriteHeader(http.StatusOK)

			}
		}
		if !getCheck {
			http.Error(res, "No metric found", http.StatusNotFound)
		}
	default:
		http.Error(res, "Invalid metric type", http.StatusNotFound)

	}
}
func (s *ServerPing) GetPing(res http.ResponseWriter, req *http.Request) {

	pool, _, cancel, err := repository.GetConnection(*s.databaseDSN)

	if err != nil {
		http.Error(res, "Connection error", http.StatusInternalServerError)
		log.Fatalf("Connection error: %v", err)
	} else {
		res.WriteHeader(http.StatusOK)

	}
	defer cancel()
	defer pool.Close()

}
