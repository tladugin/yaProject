package grpc

import (
	"context"
	"fmt"
	proto2 "github.com/tladugin/yaProject.git/internal/proto"
	"github.com/tladugin/yaProject.git/internal/repository"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"log"
	"net"
)

// MetricsServer реализует gRPC сервис для метрик
type MetricsServer struct {
	proto2.UnimplementedMetricsServer
	storage *repository.MemStorage
}

// NewMetricsServer создает новый gRPC сервер для метрик
func NewMetricsServer(storage *repository.MemStorage) *MetricsServer {
	return &MetricsServer{
		storage: storage,
	}
}

// UpdateMetrics обновляет метрики на сервере
func (s *MetricsServer) UpdateMetrics(ctx context.Context, req *proto2.UpdateMetricsRequest) (*proto2.UpdateMetricsResponse, error) {
	for _, metric := range req.GetMetrics() {
		switch metric.GetType() {
		case proto2.Metric_GAUGE:
			// Проверяем, что значение не nil
			if metric.Value != 0 {
				// Получаем значение через GetValue()
				value := metric.GetValue()
				s.storage.AddGauge(metric.GetId(), value)
			}
		case proto2.Metric_COUNTER:
			// Проверяем, что значение не nil
			if metric.Delta != 0 {
				// Получаем значение через GetDelta()
				delta := metric.GetDelta()
				s.storage.AddCounter(metric.GetId(), delta)
			}
		default:
			return nil, status.Errorf(codes.InvalidArgument, "unknown metric type: %v", metric.GetType())
		}
	}

	return &proto2.UpdateMetricsResponse{}, nil
}

// IPInterceptor проверяет IP-адрес клиента
func IPInterceptor(trustedSubnet string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Если подсеть не задана, пропускаем проверку
		if trustedSubnet == "" {
			return handler(ctx, req)
		}

		// Получаем метаданные
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.PermissionDenied, "metadata not provided")
		}

		// Получаем IP из метаданных
		ipValues := md.Get("x-real-ip")
		if len(ipValues) == 0 {
			return nil, status.Errorf(codes.PermissionDenied, "x-real-ip header not provided")
		}

		clientIP := ipValues[0]

		// Проверяем IP
		allowed, err := checkIPInSubnet(clientIP, trustedSubnet)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to check IP: %v", err)
		}

		if !allowed {
			return nil, status.Errorf(codes.PermissionDenied, "access denied for IP: %s", clientIP)
		}

		return handler(ctx, req)
	}
}

// checkIPInSubnet проверяет вхождение IP в подсеть
func checkIPInSubnet(ipStr, cidr string) (bool, error) {
	if cidr == "" {
		return true, nil
	}

	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return false, fmt.Errorf("invalid CIDR: %w", err)
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false, fmt.Errorf("invalid IP: %s", ipStr)
	}

	return ipnet.Contains(ip), nil
}

// RunGRPCServer запускает gRPC сервер
func RunGRPCServer(storage *repository.MemStorage, address string, trustedSubnet string) error {
	lis, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	// Создаем gRPC сервер с интерцептором
	s := grpc.NewServer(
		grpc.UnaryInterceptor(IPInterceptor(trustedSubnet)),
	)

	// Регистрируем сервис
	server := NewMetricsServer(storage)
	proto2.RegisterMetricsServer(s, server)

	log.Printf("gRPC server listening on %s", address)

	if err := s.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve: %w", err)
	}

	return nil
}
