package compress

import (
	"net/http"
	"slices"
	"strings"

	"go.uber.org/zap"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/repositories"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/logger"
)

var contentTypes = []string{
	"application/json",
	"text/html",
	"",
}

func GzipMiddleware(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// по умолчанию устанавливаем оригинальный http.ResponseWriter как тот,
		// который будем передавать следующей функции
		ow := w

		// проверяем, что клиент умеет получать от сервера сжатые данные в формате gzip
		acceptEncoding := r.Header.Get("Accept-Encoding")
		supportsGzip := strings.Contains(acceptEncoding, "gzip")
		contentType := r.Header.Get("Content-Type")
		if supportsGzip && slices.Contains(contentTypes, contentType) {
			logger.ServerLog.Debug("client accept encoding, compress answer data", zap.String("Accept-Encoding", acceptEncoding),
				zap.String("Content-Type", contentType))
			// оборачиваем оригинальный http.ResponseWriter новым с поддержкой сжатия
			cw := repositories.NewCompressWriter(w)
			// меняем оригинальный http.ResponseWriter на новый
			ow = cw
			// не забываем отправить клиенту все сжатые данные после завершения middleware
			defer cw.Close()
		}

		// проверяем, что клиент отправил серверу сжатые данные в формате gzip
		contentEncoding := r.Header.Get("Content-Encoding")
		sendsGzip := strings.Contains(contentEncoding, "gzip")
		if sendsGzip {
			logger.ServerLog.Debug("client push encoding data, needed decompress", zap.String("Content-Encoding", contentEncoding))
			// оборачиваем тело запроса в io.Reader с поддержкой декомпрессии
			cr, err := repositories.NewCompressReader(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			// меняем тело запроса на новое
			r.Body = cr
			defer cr.Close()
		}

		// передаём управление хендлеру
		h.ServeHTTP(ow, r)
	}
}
