package server

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"
)

var privateKey *rsa.PrivateKey

// LoadPrivateKey загружает приватный ключ из файла
func LoadPrivateKey(keyPath string) error {
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return err
	}

	block, _ := pem.Decode(keyData)
	if block == nil {
		return fmt.Errorf("failed to parse PEM block containing the private key")
	}

	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		// Попробуем PKCS8
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return err
		}
		var ok bool
		privateKey, ok = key.(*rsa.PrivateKey)
		if !ok {
			return fmt.Errorf("not an RSA private key")
		}
	} else {
		privateKey = priv
	}

	return nil
}

// DecryptData расшифровывает данные с помощью приватного ключа
func DecryptData(encryptedData []byte) ([]byte, error) {
	if privateKey == nil {
		return encryptedData, nil // Если ключ не загружен, возвращаем исходные данные
	}

	// Расшифровываем данные с помощью RSA-OAEP
	decrypted, err := rsa.DecryptOAEP(
		sha256.New(),
		rand.Reader,
		privateKey,
		encryptedData,
		nil,
	)
	if err != nil {
		return nil, err
	}

	return decrypted, nil
}

// DecryptMiddleware для использования с chi router
func DecryptMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Пропускаем GET запросы и запросы без тела
		if r.Method == http.MethodGet || r.Body == nil || r.ContentLength == 0 {
			next.ServeHTTP(w, r)
			return
		}

		// Читаем тело запроса
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// Расшифровываем данные
		decryptedData, err := DecryptData(bodyBytes)
		if err != nil {
			http.Error(w, "Failed to decrypt data", http.StatusBadRequest)
			return
		}

		// Заменяем тело запроса
		r.Body = io.NopCloser(bytes.NewReader(decryptedData))
		r.ContentLength = int64(len(decryptedData))

		next.ServeHTTP(w, r)
	})
}
