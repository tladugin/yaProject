package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/tladugin/yaProject.git/internal/agent"
	"github.com/tladugin/yaProject.git/internal/logger"
)

func main() {
	// Инициализация логгера
	sugar, err := logger.InitLogger()
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := sugar.Sync(); err != nil {
			log.Fatal(err)
		}
	}()

	// Парсинг флагов
	flags := agent.ParseFlags()

	// Создание и запуск агента
	newAgent := agent.NewAgent(flags, sugar)

	// Запуск агента
	if err := newAgent.Start(); err != nil {
		sugar.Fatal("Failed to start agent:", err)
	}

	// Ожидание сигнала завершения
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)
	<-shutdown

	// Остановка агента
	newAgent.Stop()
	sugar.Infoln("Shutting down...")
}
