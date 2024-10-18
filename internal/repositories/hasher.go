package repositories

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"

	"go.uber.org/zap"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/logger"
)

func CalkHash(body []byte, key string) (string, error) {
	// подписываем алгоритмом HMAC, используя SHA-256
	h := hmac.New(sha256.New, []byte(key))
	_, err := h.Write(body)
	if err != nil {
		return "", err
	}
	hash := h.Sum(nil)

	// Преобразуем хэш в строку
	hashStr := hex.EncodeToString(hash[:])
	return hashStr, nil
}

func CheckHash(body []byte, wantHash, key string) error {
	logger.ServerLog.Debug("getting body and hash to check in CheckHash", zap.String("body", fmt.Sprintf("%x", body)), zap.String("hash", wantHash),
		zap.String("key", key))

	reqHashBytes, err := hex.DecodeString(wantHash)
	if err != nil {
		return err
	}

	// подписываем алгоритмом HMAC, используя SHA-256
	h := hmac.New(sha256.New, []byte(key))
	_, err = h.Write(body)
	if err != nil {
		return err
	}
	hash := h.Sum(nil)

	if !hmac.Equal(hash, reqHashBytes) {
		return fmt.Errorf("hashs is not equal, want %x, get %x", reqHashBytes, hash)
	}
	return nil
}

// HashWriter реализует интерфейс http.ResponseWriter и позволяет прозрачно для сервера
// получить тело ответа для последующей подписи, если на сервере задан ключ
type HashWriter struct {
	w   http.ResponseWriter
	key string
}

func NewHashWriter(w http.ResponseWriter, key string) *HashWriter {
	return &HashWriter{
		w:   w,
		key: key,
	}
}

func (h *HashWriter) Header() http.Header {
	return h.w.Header()
}

func (h *HashWriter) Write(p []byte) (int, error) {
	hash, err := CalkHash(p, h.key)
	if err != nil {
		return 0, err
	}
	// Устанавливаю заголовок о подписи данных и результат подписи хэша
	h.w.Header().Set("HashSHA256", hash)

	logger.ServerLog.Debug("calculated hash in Write method", zap.String("hash", hash), zap.String("size of p", fmt.Sprintf("%d", len(p))),
		zap.String("body", fmt.Sprintf("%x", p)))

	return h.w.Write(p)
}

func (h *HashWriter) WriteHeader(statusCode int) {
	h.w.WriteHeader(statusCode)
}
