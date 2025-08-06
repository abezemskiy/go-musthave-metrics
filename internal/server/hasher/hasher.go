package hasher

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"go.uber.org/zap"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/repositories"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/logger"
)

var key string

// SetKey - устанавливает секретный ключ для подписи данных.
func SetKey(k string) {
	key = k
}

// GetKey - возвращает секретный ключ для подписи данных.
func GetKey() string {
	return key
}

// HashMiddleware - middleware для проверки подписи и подписи данных, если установлен ключ.
func HashMiddleware(handler http.Handler) http.HandlerFunc {
	logFn := func(res http.ResponseWriter, req *http.Request) {
		if k := GetKey(); k == "" {
			handler.ServeHTTP(res, req)
			return
		}

		// О необходимости такого поведения понял из тестов
		noneHash := req.Header.Get("Hash")
		if noneHash == "none" {
			handler.ServeHTTP(res, req)
			return
		}
		reqHash := req.Header.Get("HashSHA256")
		if reqHash == "" {
			handler.ServeHTTP(res, req)
			return
		}

		// Проверяю подпись----------------------------------------
		body, err := io.ReadAll(req.Body)
		if err != nil {
			logger.ServerLog.Error("read body into string error: ", zap.String("address", req.URL.String()), zap.String("error: ", error.Error(err)))
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		logger.ServerLog.Debug("geting body from agent ", zap.String("body: ", fmt.Sprintf("%x", body)))

		// Восстанавливаем тело запроса, чтобы другие хендлеры могли его использовать
		req.Body = io.NopCloser(bytes.NewReader(body))

		// проверка подписи в случае непустого тела запроса
		if len(body) != 0 {
			ok, err := repositories.CheckHash(body, reqHash, GetKey())
			if err != nil {
				logger.ServerLog.Error("checking hash error", zap.String("address", req.URL.String()), zap.String("error: ", error.Error(err)))

				res.WriteHeader(http.StatusInternalServerError)
				return
			}
			if !ok {
				logger.ServerLog.Error("hashs is not equal ", zap.String("address", req.URL.String()))
				res.WriteHeader(http.StatusBadRequest)
				return
			}
		}

		// Подписываю ответ сервера в случае, если задан ключ---------------------------------------------
		// Устанавливаю мидлварь для получения тела ответа сервера
		var writer = repositories.NewHashWriter(res, GetKey())
		handler.ServeHTTP(writer, req)
	}
	return logFn
}
