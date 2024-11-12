package encrypt

import (
	"bytes"
	"io"
	"net/http"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/tools/encryption"
)

// cryptoKey - переменна, которая хранит адрес к приватному ключу для расшифровки данных от агента.
var cryptoKey string

// SetCryptoKey - функция для установки пути к приватному ключу сервера
func SetCryptoKey(key string) {
	cryptoKey = key
}

// Middleware - мидлварь, которая расшифровывает данные от агента.
func Middleware(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// если установлен адрес к приватному ключу предполагается, что используется шифрование данных
		if cryptoKey != "" {
			// Чтение зашифрованного тела запроса
			encryptedData, err := io.ReadAll(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			defer r.Body.Close()

			decryptedData, err := encryption.DecryptData(cryptoKey, encryptedData)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			// Создание нового тела запроса с расшифрованными данными
			r.Body = io.NopCloser(bytes.NewReader(decryptedData))
		}
		// передаём управление хендлеру
		h.ServeHTTP(w, r)
	}
}
