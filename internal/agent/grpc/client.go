package grpc

import (
	"context"
	"fmt"
	"github.com/tladugin/yaProject.git/internal/proto"

	"github.com/tladugin/yaProject.git/internal/repository"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"log"
	"time"
)

// GRPCClient представляет gRPC клиент для отправки метрик
type GRPCClient struct {
	client  proto.MetricsClient
	conn    *grpc.ClientConn
	localIP string
	timeout time.Duration
}

// NewGRPCClient создает нового gRPC клиента
func NewGRPCClient(serverAddr string, localIP string, timeout time.Duration) (*GRPCClient, error) {
	// Устанавливаем соединение с сервером
	conn, err := grpc.NewClient(
		serverAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1024*1024*10)), // 10MB
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC server: %w", err)
	}

	client := proto.NewMetricsClient(conn)

	return &GRPCClient{
		client:  client,
		conn:    conn,
		localIP: localIP,
		timeout: timeout,
	}, nil
}

// Close закрывает соединение
func (c *GRPCClient) Close() error {
	return c.conn.Close()
}

// SendMetrics отправляет метрики на сервер
func (c *GRPCClient) SendMetrics(storage *repository.MemStorage, pollCounter int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	// Добавляем IP в метаданные
	md := metadata.New(map[string]string{"x-real-ip": c.localIP})
	ctx = metadata.NewOutgoingContext(ctx, md)

	// Подготавливаем запрос
	req := &proto.UpdateMetricsRequest{}

	// Добавляем gauge метрики
	for _, gauge := range storage.GaugeSlice() {
		metric := &proto.Metric{
			Id:    gauge.Name,
			Type:  proto.Metric_GAUGE,
			Value: gauge.Value,
		}
		req.Metrics = append(req.Metrics, metric)
	}

	// Добавляем counter метрики
	for _, counter := range storage.CounterSlice() {
		delta := pollCounter
		if counter.Name == "PollCount" {
			delta = int64(counter.Value)
		}

		metric := &proto.Metric{
			Id:    counter.Name,
			Type:  proto.Metric_COUNTER,
			Delta: delta,
		}
		req.Metrics = append(req.Metrics, metric)
	}

	// Отправляем запрос
	_, err := c.client.UpdateMetrics(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to send metrics: %w", err)
	}

	log.Printf("Successfully sent %d metrics via gRPC", len(req.Metrics))
	return nil
}

// SendWithRetry отправляет метрики с повторными попытками
func (c *GRPCClient) SendWithRetry(storage *repository.MemStorage, pollCounter int64, maxRetries int) error {
	var lastErr error

	for i := 0; i <= maxRetries; i++ {
		if i > 0 {
			// Экспоненциальная задержка: 1s, 2s, 4s, ...
			delay := time.Duration(1<<uint(i-1)) * time.Second
			log.Printf("Retry attempt %d/%d after %v", i, maxRetries, delay)
			time.Sleep(delay)
		}

		err := c.SendMetrics(storage, pollCounter)
		if err == nil {
			return nil
		}

		lastErr = err
		log.Printf("Attempt %d failed: %v", i+1, err)
	}

	return fmt.Errorf("after %d attempts: %w", maxRetries, lastErr)
}
