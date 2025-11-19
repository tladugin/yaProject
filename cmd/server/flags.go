package main

import (
	"flag"
	"os"
	"strings"
)

type flags struct {
	flagRunAddr         string
	flagStoreInterval   string
	flagFileStoragePath string
	flagRestore         bool
	flagDatabaseDSN     string
	flagKey             string
	flagAuditFile       string
	flagAuditURL        string
	flagUsePprof        bool
	flagCryptoKey       string
}

// "host=localhost user=postgres password=543218 dbname=metrics sslmode=disable"

// parseFlags обрабатывает аргументы командной строки
// и сохраняет их значения в соответствующих переменных
func parseFlags() flags {
	const defaultDSN = ""
	var f flags
	// регистрируем переменную flagRunAddr
	// как аргумент -a со значением :8080 по умолчанию
	flag.StringVar(&f.flagRunAddr, "a", "localhost:8080", "address and port to run server")

	flag.StringVar(&f.flagStoreInterval, "i", "300", "saving server data interval")
	flag.StringVar(&f.flagFileStoragePath, "f", "server_backup", "path for server backup file")
	flag.BoolVar(&f.flagRestore, "r", false, "restore server data")
	flag.StringVar(&f.flagDatabaseDSN, "d", defaultDSN, "database DSN")
	flag.StringVar(&f.flagKey, "k", "", "key")
	flag.StringVar(&f.flagAuditFile, "audit-file", "", "path for server audit file")
	flag.StringVar(&f.flagAuditURL, "audit-url", "", "audit URL")
	flag.BoolVar(&f.flagUsePprof, "pprof", false, "use benchmark")
	flag.StringVar(&f.flagCryptoKey, "crypto-key", "", "path to private key for decryption")
	// парсим переданные серверу аргументы в зарегистрированные переменные
	flag.Parse()

	envRunAddr, ok := os.LookupEnv("ADDRESS")
	if ok && strings.TrimSpace(envRunAddr) != "" {
		f.flagRunAddr = envRunAddr
	}

	envStoreInterval, ok := os.LookupEnv("STORE_INTERVAL")
	if ok && strings.TrimSpace(envStoreInterval) != "" {
		f.flagStoreInterval = envStoreInterval
	}

	envFileStoragePath, ok := os.LookupEnv("FILE_STORAGE_PATH")
	if ok && strings.TrimSpace(envFileStoragePath) != "" {
		f.flagFileStoragePath = envFileStoragePath
	}

	envRestoreKey, ok := os.LookupEnv("RESTORE")
	if ok && strings.TrimSpace(envRestoreKey) != "" {
		f.flagRestore = true
	}

	envConnectString, ok := os.LookupEnv("DATABASE_DSN")
	if ok && strings.TrimSpace(envConnectString) != "" {
		f.flagDatabaseDSN = envConnectString
	}

	envKey, ok := os.LookupEnv("KEY")
	if ok && strings.TrimSpace(envKey) != "" {
		f.flagKey = envKey
	}
	envAuditFile, ok := os.LookupEnv("AUDIT_FILE")
	if ok && strings.TrimSpace(envAuditFile) != "" {
		f.flagAuditFile = envAuditFile
	}
	envAuditURL, ok := os.LookupEnv("AUDIT_URL")
	if ok && strings.TrimSpace(envAuditURL) != "" {
		f.flagAuditURL = envAuditURL
	}
	if envCryptoKey := os.Getenv("CRYPTO_KEY"); envCryptoKey != "" {
		f.flagCryptoKey = envCryptoKey
	}
	return f
}
