package encrypt

import (
	"bytes"
	"io"
	"net/http"

	"go.uber.org/zap"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/logger"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/tools/encryption"
)

// переменная, которая хранит структуру шифрования и расшифровки
var cryptoGrapher encryption.Cryptographer

// SetCryptoGrapher - функция для установки структуры шифрования и расшифровки данных
func SetCryptoGrapher(c *encryption.Cryptographer) {
	cryptoGrapher = *c
}

// Middleware - мидлварь, которая расшифровывает данные от агента.
func Middleware(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// если установлен адрес к приватному ключу предполагается, что используется шифрование данных
		if cryptoGrapher.PrivateKeyIsSet() {
			// Чтение зашифрованного тела запроса
			encryptedData, err := io.ReadAll(r.Body)
			if err != nil {
				logger.ServerLog.Error("read encrypted body error", zap.String("error", error.Error(err)))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			defer r.Body.Close()

			decryptedData, err := cryptoGrapher.Decrypt(encryptedData)
			if err != nil {
				logger.ServerLog.Error("decrypt data error", zap.String("error", error.Error(err)))
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
