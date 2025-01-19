package hasher

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/logger"
)

var key string

// SetKey - устанавливает секретный ключ для подписи и расшифровки данных.
func SetKey(k string) {
	key = k
}

// GetKey - получает установленный секретный ключ.
func GetKey() string {
	return key
}

// VerifyHashMiddleware - проверяет хэш тела ответа
func VerifyHashMiddleware(_ *resty.Client, resp *resty.Response) error {
	// Если ключ не задан, то проверять подпись данных не нужно
	if k := GetKey(); k == "" {
		return nil
	}
	// Если ответ сервера
	if resp.StatusCode() != http.StatusOK {
		return nil
	}

	// Получаем тело ответа в виде байтов
	bodyBytes := resp.Body()

	// Извлекаем хэш из заголовка ответа
	serverHash := resp.Header().Get("HashSHA256")
	if serverHash == "" {
		return errors.New("missing HashSHA256 header in the response")
	}
	// Логирование заголовка
	logger.AgentLog.Debug("Received HashSHA256 header and body", zap.String("header", serverHash), zap.String("body", fmt.Sprintf("%x", bodyBytes)))

	serverHashBytes, err := hex.DecodeString(serverHash)

	if err != nil {
		return err
	}

	// подписываем алгоритмом HMAC, используя SHA-256
	h := hmac.New(sha256.New, []byte(GetKey()))
	_, err = h.Write(bodyBytes)
	if err != nil {
		return err
	}
	hash := h.Sum(nil)

	// проверяю хэши
	if !hmac.Equal(hash, serverHashBytes) {
		err := fmt.Errorf("want %x, get %x", hash, serverHashBytes)
		logger.AgentLog.Error("hashs is not equal ", zap.String("error: ", error.Error(err)))
		return err
	}

	return nil
}
