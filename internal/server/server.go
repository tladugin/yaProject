package server

import (
	"context"
	"fmt"
	"github.com/go-chi/chi/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/tladugin/yaProject.git/internal/handler"
	"github.com/tladugin/yaProject.git/internal/logger"
	"github.com/tladugin/yaProject.git/internal/repository"
	"log"
	"net"
	"net/http"
	"sync"
)

// Глобальные переменные для информации о сборке
var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

// RunHTTPServer запускает HTTP сервер для работы с метриками
func RunHTTPServer(
	storage *repository.MemStorage,
	producer *repository.Producer,
	ctx context.Context, // ИЗМЕНЕНО: контекст вместо канала
	wg *sync.WaitGroup,
	flagStoreInterval int,
	flagRunAddr *string,
	flagDatabaseDSN *string,
	flagKey *string,
	flagAuditFile *string,
	flagAuditURL *string,
	ipChecker *IPChecker,
) {
	defer wg.Done()

	// Создаем менеджер аудита для отслеживания операций
	auditManager := NewAuditManager(true)
	defer auditManager.Close()

	// Инициализируем наблюдатели аудита (файловый и/или HTTP)
	initAuditObservers(auditManager, flagAuditFile, flagAuditURL)

	// Создаем обработчики для разных режимов работы
	s := handler.NewServer(storage)                   // Базовый обработчик
	sSync := handler.NewServerSync(storage, producer) // Синхронный обработчик с бэкапом

	var db *handler.ServerDB     // Обработчик для работы с БД
	var ping *handler.ServerPing // Обработчик для проверки соединения

	// Инициализация работы с PostgreSQL если указан DSN
	if *flagDatabaseDSN != "" {
		// Проверка и применение миграций базы данных
		p, _, err := repository.NewPostgresRepository(*flagDatabaseDSN)
		if err != nil {
			logger.Sugar.Error("Failed to initialize storage: %v", err.Error())
		}
		defer p.Close()

		// Установка соединения с базой данных
		pool, _, _, err := repository.GetConnection(*flagDatabaseDSN)
		if err != nil {
			logger.Sugar.Error("Failed to get connection!: %v", err.Error())
		}
		defer pool.Close()

		// Создание обработчиков для работы с БД
		ping = handler.NewServerPingDB(storage, flagDatabaseDSN) // Обработчик проверки доступности БД
		db = handler.NewServerDB(storage, pool, flagKey)         // Основной обработчик операций с БД
	}

	// Настройка маршрутизатора
	r := chi.NewRouter()

	// Регистрация middleware компонентов
	if ipChecker != nil {
		r.Use(IPCheckMiddleware(ipChecker))
	}
	r.Use(
		DecryptMiddleware,                   // Расшифровывание запросов
		repository.GzipMiddleware,           // Сжатие ответов
		logger.LoggingAnswer(logger.Sugar),  // Логирование ответов
		logger.LoggingRequest(logger.Sugar), // Логирование запросов
		AuditMiddleware(auditManager),       // Аудит операций
	)

	// Определение маршрутов приложения
	r.Route("/", func(r chi.Router) {

		// Маршруты для работы с базой данных (если подключена)
		if *flagDatabaseDSN != "" {
			r.Get("/ping", ping.GetPing)                       // Проверка доступности БД
			r.Post("/update", db.PostUpdatePostgres)           // Обновление метрик
			r.Post("/update/", db.PostUpdatePostgres)          // Альтернативный путь обновления
			r.Post("/value", db.PostValue)                     // Получение значения метрики
			r.Post("/value/", db.PostValue)                    // Альтернативный путь получения
			r.Post("/updates", db.UpdatesGaugesBatchPostgres)  // Пакетное обновление метрик
			r.Post("/updates/", db.UpdatesGaugesBatchPostgres) // Альтернативный путь пакетного обновления

		} else {
			// Маршруты для работы с in-memory хранилищем
			r.Get("/", s.MainPage)                                   // Главная страница
			r.Get("/value/{metric}/{name}", s.GetHandler)            // Получение метрики через URL параметры
			r.Post("/update/{metric}/{name}/{value}", s.PostHandler) // Обновление через URL параметры
			r.Post("/updates", s.UpdatesGaugesBatch)                 // Пакетное обновление gauge метрик
			r.Post("/updates/", s.UpdatesGaugesBatch)                // Альтернативный путь пакетного обновления

			// Выбор режима бэкапа в зависимости от интервала
			if flagStoreInterval == 0 {
				logger.Sugar.Info("Running in sync backup mode")
				r.Post("/update", sSync.PostUpdateSyncBackup)  // Синхронный бэкап после каждого обновления
				r.Post("/update/", sSync.PostUpdateSyncBackup) // Альтернативный путь синхронного обновления
			} else {
				logger.Sugar.Info("Running in async backup mode")
				r.Post("/update", s.PostUpdate)  // Асинхронный бэкап (по расписанию)
				r.Post("/update/", s.PostUpdate) // Альтернативный путь асинхронного обновления
			}

			r.Post("/value", s.PostValue)  // Получение значения через POST
			r.Post("/value/", s.PostValue) // Альтернативный путь получения
		}
	})

	// Настройка HTTP сервера
	server := &http.Server{
		Addr:    *flagRunAddr,
		Handler: r,
	}

	// Горутина для graceful shutdown
	go func() {
		<-ctx.Done() // ИЗМЕНЕНО: ждем отмены контекста
		logger.Sugar.Info("Shutting down HTTP server...")
		if err := server.Close(); err != nil {
			logger.Sugar.Error("HTTP server shutdown error: ", err)
		}
	}()

	// Запуск сервера
	logger.Sugar.Infof("Starting server on %s", *flagRunAddr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Sugar.Error("Server failed: ", err)
	}
}

// printBuildInfo выводит информацию о сборке
func PrintBuildInfo() {
	// Устанавливаем "N/A" если значения не заданы
	if buildVersion == "" {
		buildVersion = "N/A"
	}
	if buildDate == "" {
		buildDate = "N/A"
	}
	if buildCommit == "" {
		buildCommit = "N/A"
	}

	// Вывод в формате согласно требованиям
	log.Printf("Build version: %s", buildVersion)
	log.Printf("Build date: %s", buildDate)
	log.Printf("Build commit: %s", buildCommit)
}

// IPChecker проверяет IP-адрес на вхождение в доверенную подсеть
type IPChecker struct {
	trustedSubnet *net.IPNet
}

// NewIPChecker создает новый IPChecker
func NewIPChecker(cidr string) (*IPChecker, error) {
	if cidr == "" {
		return &IPChecker{}, nil
	}

	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, fmt.Errorf("invalid CIDR notation: %w", err)
	}

	return &IPChecker{
		trustedSubnet: ipnet,
	}, nil
}

// IsAllowed проверяет, разрешен ли IP-адрес
func (c *IPChecker) IsAllowed(ipStr string) (bool, error) {
	// Если подсеть не задана, разрешаем все
	if c.trustedSubnet == nil {
		return true, nil
	}

	// Парсим IP-адрес
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false, fmt.Errorf("invalid IP address: %s", ipStr)
	}

	// Проверяем вхождение в подсеть
	return c.trustedSubnet.Contains(ip), nil
}

// GetClientIP извлекает IP-адрес клиента из заголовка X-Real-IP
func GetClientIP(r *http.Request) string {
	// Получаем IP из заголовка X-Real-IP
	realIP := r.Header.Get("X-Real-IP")

	// Проверяем, что IP не пустой
	if realIP == "" {
		return ""
	}

	// Возвращаем IP из заголовка
	return realIP
}

// IPCheckMiddleware проверяет IP-адрес клиента
func IPCheckMiddleware(ipChecker *IPChecker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Получаем IP-адрес клиента
			clientIP := GetClientIP(r)

			// Проверяем доступ
			allowed, err := ipChecker.IsAllowed(clientIP)
			if err != nil {
				log.Printf("Failed to check IP: %v (IP: %s)", err, clientIP)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			if !allowed {
				log.Printf("Access denied for IP: %s to path: %s", clientIP, r.URL.Path)
				http.Error(w, "Access denied", http.StatusForbidden)
				return
			}

			log.Printf("IP check passed for IP: %s to path: %s", clientIP, r.URL.Path)
			next.ServeHTTP(w, r)
		})
	}
}
