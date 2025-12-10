package agent

import (
	"bytes"
	"compress/gzip"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/tladugin/yaProject.git/internal/models"
	"github.com/tladugin/yaProject.git/internal/repository"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

// SendMetric отправляет одиночную метрику на сервер
func SendMetric(URL string, metricType string, storage *repository.MemStorage, i int, key string, localIP string) error {
	// 1. Подготовка метрики
	var metric models.Metrics

	switch metricType {
	case "gauge":
		metric = models.Metrics{
			MType: "gauge",
			ID:    storage.GaugeSlice()[i].Name,
			Value: &storage.GaugeSlice()[i].Value,
		}
	case "counter":
		metric = models.Metrics{
			MType: "counter",
			ID:    storage.CounterSlice()[i].Name,
			Delta: &storage.CounterSlice()[i].Value,
		}
	default:
		return fmt.Errorf("unknown metric type: %s", metricType)
	}

	// 2. Сериализация в JSON
	jsonData, err := json.Marshal(metric)
	if err != nil {
		return fmt.Errorf("json marshal error: %w", err)
	}

	// 3. Сжатие данных
	buf, err := repository.CompressData(jsonData)
	if err != nil {
		return fmt.Errorf("compress data error: %w", err)
	}

	// 4. Нормализация URL (добавление протокола если отсутствует)
	if !strings.HasPrefix(URL, "http://") && !strings.HasPrefix(URL, "https://") {
		URL = "http://" + URL
	}

	// 5. Создание и настройка запроса
	req, err := http.NewRequest("POST", URL, buf)
	if err != nil {
		return fmt.Errorf("request creation failed: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept-Encoding", "gzip")

	// Добавляем заголовок X-Real-IP
	if localIP != "" {
		req.Header.Set("X-Real-IP", localIP)
	}

	// 5.1 Проверяем наличие ключа, если он есть, отправляем в заголовке хеш
	if key != "" {
		bytesBuf := buf.Bytes()
		bytesKey := []byte(key)
		hash := sha256.Sum256(append(bytesKey, bytesBuf...))
		hashHeader := hex.EncodeToString(hash[:])
		req.Header.Set("HashSHA256", hashHeader)
	}

	// 6. Отправка запроса
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("failed to close response body: %v", err)
		}
	}()

	// 7. Проверка ответа сервера
	if resp.StatusCode != http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read error response: %w", err)
		}
		return fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// SendMetricsBatch отправляет пачку метрик на сервер
func SendMetricsBatch(URL string, metricType string, storage *repository.MemStorage, batchSize int, key string, pollCounter int64, FlagCryptoKey string, localIP string) error {
	// 1. Подготовка URL
	if !strings.HasPrefix(URL, "http://") && !strings.HasPrefix(URL, "https://") {
		URL = "http://" + URL
	}

	// 2. Подготовка метрик
	var metrics []models.Metrics
	switch metricType {
	case "gauge":
		if len(storage.GaugeSlice()) == 0 {
			return nil // Нет метрик для отправки
		}

		for i := 0; i < batchSize; i++ {
			value := storage.GaugeSlice()[i].Value // Создаем копию значения
			metrics = append(metrics, models.Metrics{
				MType: "gauge",
				ID:    storage.GaugeSlice()[i].Name,
				Value: &value,
			})
		}
	case "counter":
		if len(storage.CounterSlice()) == 0 {
			return nil // Нет метрик для отправки
		}

		for i := 0; i < batchSize; i++ {
			delta := pollCounter
			metrics = append(metrics, models.Metrics{
				MType: "counter",
				ID:    storage.CounterSlice()[i].Name,
				Delta: &delta,
			})
		}
	default:
		return fmt.Errorf("unknown metric type: %s", metricType)
	}

	// 3. Сериализация в JSON
	jsonData, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("json marshal error: %w", err)
	}

	// 4. Сжатие данных
	compressedData, err := compressData(jsonData)
	if err != nil {
		return fmt.Errorf("compress data error: %w", err)
	}

	// 4.1 Шифрование данных при наличии ключа шифрования
	if FlagCryptoKey != "" {
		var publicKey *rsa.PublicKey

		if publicKey, err = LoadPublicKey(FlagCryptoKey); err != nil {
			return fmt.Errorf("load public key error: %w", err)
		}
		fmt.Println("Using public key")

		compressedData, err = EncryptData(compressedData, publicKey)
		if err != nil {
			return fmt.Errorf("encrypt data error: %w", err)
		}
	}

	// 5. Создание запроса
	req, err := http.NewRequest("POST", URL, bytes.NewReader(compressedData))
	if err != nil {
		return fmt.Errorf("request creation failed: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept-Encoding", "gzip")

	// Добавляем заголовок X-Real-IP
	if localIP != "" {
		req.Header.Set("X-Real-IP", localIP)
	}

	// 6. Добавление хеша, если есть ключ
	if key != "" {
		hash := sha256.Sum256(append([]byte(key), jsonData...)) // Хешируем исходные данные
		req.Header.Set("HashSHA256", hex.EncodeToString(hash[:]))
	}

	// 7. Отправка запроса
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// 8. Проверка ответа сервера
	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read error response: %w", err)
		}
		return fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// SendWithRetry отправляет метрики с повторными попытками при временных ошибках
func SendWithRetry(url string, storage *repository.MemStorage, key string, pollCounter int64, FlagCryptoKey string, localIP string) error {
	maxRetries := 3
	retryDelays := []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second}
	var lastErr error

	// Цикл повторных попыток
	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Задержка перед повторной попыткой (кроме первой)
		if attempt > 0 {
			time.Sleep(retryDelays[attempt-1])
		}

		// Отправка gauge метрик
		errG := SendMetricsBatch(url, "gauge", storage, len(storage.GaugeSlice()), key, pollCounter, FlagCryptoKey, localIP)
		if errG != nil {
			lastErr = errG
		}

		// Отправка counter метрик
		errC := SendMetricsBatch(url, "counter", storage, len(storage.CounterSlice()), key, pollCounter, FlagCryptoKey, localIP)
		if errC == nil {
			return nil // Успешная отправка
		}

		lastErr = errC

		// Проверяем, стоит ли повторять запрос
		if !isRetriableError(errG) || !isRetriableError(errC) {
			break // Не повторяем для неустранимых ошибок
		}
	}

	return fmt.Errorf("after %d attempts: %w", maxRetries, lastErr)
}

// GetLocalIP возвращает локальный IP-адрес агента
func GetLocalIP() (string, error) {
	// Попытка 1: Получение IP через интерфейсы сети
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", fmt.Errorf("failed to get network interfaces: %w", err)
	}

	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if ok && !ipNet.IP.IsLoopback() && ipNet.IP.To4() != nil {
			return ipNet.IP.String(), nil
		}
	}

	// Попытка 2: DNS запрос для получения внешнего IP
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		// Попытка 3: Используем hostname
		hostname, err := os.Hostname()
		if err != nil {
			return "", fmt.Errorf("failed to get any IP address: %w", err)
		}

		addrs, err := net.LookupIP(hostname)
		if err != nil {
			return "", fmt.Errorf("failed to lookup IP from hostname: %w", err)
		}

		for _, addr := range addrs {
			if addr.To4() != nil && !addr.IsLoopback() {
				return addr.String(), nil
			}
		}
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String(), nil
}

// GetLocalIPWithFallback возвращает локальный IP с fallback значением
func GetLocalIPWithFallback() string {
	ip, err := GetLocalIP()
	if err != nil {
		log.Printf("Warning: Failed to get local IP: %v", err)
		return "127.0.0.1" // fallback на localhost
	}
	return ip
}

// Остальные функции остаются без изменений

// LoadPublicKey загружает публичный ключ из файла
func LoadPublicKey(keyPath string) (*rsa.PublicKey, error) {
	// Читаем файл с ключом
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key file %s: %w", keyPath, err)
	}

	// Декодируем PEM блок
	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block from public key file %s: no PEM data found", keyPath)
	}

	// Проверяем тип PEM блока
	if block.Type != "PUBLIC KEY" && block.Type != "RSA PUBLIC KEY" {
		return nil, fmt.Errorf("unexpected PEM block type %q in public key file %s, expected PUBLIC KEY or RSA PUBLIC KEY", block.Type, keyPath)
	}

	// Парсим публичный ключ
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		// Пробуем альтернативный формат RSA ключа
		pub, err = x509.ParsePKCS1PublicKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse public key from file %s: %w", keyPath, err)
		}
	}

	// Приводим к типу *rsa.PublicKey
	publicKey, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("key in file %s is not an RSA public key, got %T", keyPath, pub)
	}

	return publicKey, nil
}

// EncryptData шифрует данные с помощью публичного ключа
func EncryptData(data []byte, publicKey *rsa.PublicKey) ([]byte, error) {
	if publicKey == nil {
		return data, nil // Если ключ не загружен, возвращаем исходные данные
	}

	// Шифруем данные с помощью RSA-OAEP
	encrypted, err := rsa.EncryptOAEP(
		sha256.New(),
		rand.Reader,
		publicKey,
		data,
		nil,
	)
	if err != nil {
		return nil, err
	}

	return encrypted, nil
}

// compressData сжимает данные с использованием gzip
func compressData(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(data); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// isRetriableError проверяет, является ли ошибка временной и можно ли повторить запрос
func isRetriableError(err error) bool {
	// Считаем ошибку временной, если это:
	// - ошибка сети/соединения
	// - таймаут
	// - 5xx ошибка сервера
	var netErr net.Error
	return errors.As(err, &netErr)
}
