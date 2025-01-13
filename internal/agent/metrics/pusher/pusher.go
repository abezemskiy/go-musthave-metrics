package pusher

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/compress"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/hasher"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/logger"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/metrics/builder"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/metrics/config"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/storage"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/repositories"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/tools/ipgetter"
)

// Push - отправляет метрику на сервер в JSON формате и возвращает ошибку при неудаче.
func PushJSON(address, action, typeMetric, nameMetric, valueMetric string, client *resty.Client) error {
	metric, err := builder.Build(typeMetric, nameMetric, valueMetric)
	if err != nil {
		logger.AgentLog.Error("Build metric error", zap.String("error", error.Error(err)))
		return err
	}

	// сериализую полученную струтктуру с метриками в json-представление  в виде слайса байт
	var bufEncode bytes.Buffer
	enc := json.NewEncoder(&bufEncode)
	if err := enc.Encode(metric); err != nil {
		logger.AgentLog.Error("Encode message error", zap.String("error", error.Error(err)))
		return err
	}

	// Сжатие данных для передачи
	compressBody, err := compress.Compress(bufEncode.Bytes())
	if err != nil {
		logger.AgentLog.Error("Fail to comperess push data ", zap.String("error", error.Error(err)))
		return err
	}

	// Шифрование сжатых данных если установлен путь к публичному ключу
	crypto := config.GetCryptoGrapher()
	if crypto.PublicKeyIsSet() {
		compressBody, err = crypto.Encrypt(compressBody)
		if err != nil {
			logger.AgentLog.Error("fail to encode compressed data ", zap.String("error", error.Error(err)))
			return err
		}
	}

	// Подписываю данные отправляемые на сервер
	// Делаю не через middleware, чтобы агент подписывал именно нескомпресированный ответ
	hash, err := repositories.CalkHash(bufEncode.Bytes(), hasher.GetKey())
	if err != nil {
		logger.AgentLog.Error("Fail to calc hash ", zap.String("error", error.Error(err)))
		return err
	}
	logger.AgentLog.Debug("body and hash for forwarding to server ", zap.String("body", fmt.Sprintf("%x", bufEncode.Bytes())),
		zap.String("hash", hash), zap.String("key", hasher.GetKey()))

	// получаю ip адрес хоста для передачи на сервер в заголовке X-Real-IP
	hostAddress, err := ipgetter.Get()
	if err != nil {
		logger.AgentLog.Error("Fail to get host ip address ", zap.String("error", error.Error(err)))
		return err
	}

	url := fmt.Sprintf("%s/%s", address, action)
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Content-Encoding", "gzip").
		SetHeader("Accept-Encoding", "gzip").
		SetHeader("HashSHA256", hash).
		SetHeader("X-Real-IP", hostAddress).
		SetBody(compressBody).
		Post(url)

	if err != nil {
		logger.AgentLog.Error("Push json metric to server error ", zap.String("error", error.Error(err)))
		return err
	}

	logger.AgentLog.Debug("Get answer from server", zap.String("Content-Encoding", resp.Header().Get("Content-Encoding")),
		zap.String("statusCode", fmt.Sprintf("%d", resp.StatusCode())),
		zap.String("Content-Type", resp.Header().Get("Content-Type")),
		zap.String("HashSHA256", resp.Header().Get("HashSHA256")),
		zap.String("Content-Encoding", fmt.Sprint(resp.Header().Values("Content-Encoding"))))

	if resp.StatusCode() != http.StatusOK {
		logger.AgentLog.Error("Geting status is not 200 ", zap.String("statusCode", fmt.Sprintf("%d", resp.StatusCode())))
		return fmt.Errorf("status code is: %d %w", resp.StatusCode(), errors.New(resp.String()))
	}

	contentEncoding := resp.Header().Get("Content-Encoding")
	if strings.Contains(contentEncoding, "gzip") {
		logger.AgentLog.Debug("Get compress answer data in PushJSON function", zap.String("Content-Encoding", contentEncoding))
	} else {
		logger.AgentLog.Debug("Get uncompress answer data in PushJSON function", zap.String("Content-Encoding", contentEncoding))
	}

	responceMetric := resp.Body()
	if !bytes.Equal(bufEncode.Bytes(), responceMetric) {
		return fmt.Errorf("answer metric from server not equal pushing metric: get %d, want %d", responceMetric, bufEncode.Bytes())
	}

	// Десериализую данные полученные от сервера, в основном для дебага
	var resJSON repositories.Metric
	buRes := bytes.NewBuffer(responceMetric)
	dec := json.NewDecoder(buRes)
	if err := dec.Decode(&resJSON); err != nil {
		logger.AgentLog.Error("decode decompress data from server error ", zap.String("error", error.Error(err)))
		return err
	}
	logger.AgentLog.Debug(fmt.Sprintf("decode metric from server %s", resJSON.String()))

	logger.AgentLog.Debug(fmt.Sprintf("Success push metric in JSON format: typeMetric - %s, nameMetric - %s, valueMetric - %s", typeMetric, nameMetric, valueMetric))
	return nil
}

// Push отправляет метрику на сервер и возвращает ошибку при неудаче.
func Push(address, action, typemetric, namemetric, valuemetric string, client *resty.Client) error {
	url := fmt.Sprintf("%s/%s/%s/%s/%s", address, action, typemetric, namemetric, valuemetric)

	// получаю ip адрес хоста для передачи на сервер в заголовке X-Real-IP
	hostAddress, err := ipgetter.Get()
	if err != nil {
		logger.AgentLog.Error("Fail to get host ip address ", zap.String("error", error.Error(err)))
		return err
	}

	resp, err := client.R().
		SetHeader("Content-Type", "text/plain").
		SetHeader("X-Real-IP", hostAddress).
		Post(url)

	if err != nil {
		return fmt.Errorf("error with post: %s, %w", url, err)
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("received non-200 response status: %d for url: %s", resp.StatusCode(), url)
	}
	return nil
}

// PushAll - отправляет все собранные метрики на сервер, поочередно отправляя каждую метрику по отдельности.
func PushAll(address, action string, metrics *storage.MetricsStats, client *resty.Client) {
	metrics.Lock()
	defer metrics.Unlock()

	for _, metricName := range storage.AllMetrics {
		typeMetric, value, err := metrics.GetMetricString(metricName)
		if err != nil {
			logger.AgentLog.Error(fmt.Sprintf("Failed to get metric %s: %v\n", typeMetric, err), zap.String("action", "push metrics"))
			continue
		}
		er := PushJSON(address, action, typeMetric, metricName, value, client)
		if er != nil {
			logger.AgentLog.Error(fmt.Sprintf("Failed to push metric %s: %v\n", typeMetric, er), zap.String("action", "push metrics"))
		}
	}
}

// PushBatch - отправляет батч метрик на сервер.
func PushBatch(address, action string, metricsSlice []repositories.Metric, client *resty.Client) error {

	// сериализую полученную слайс с метриками в json-представление  в виде слайса байт
	var bufEncode bytes.Buffer
	enc := json.NewEncoder(&bufEncode)
	if err := enc.Encode(metricsSlice); err != nil {
		logger.AgentLog.Error("Encode message error", zap.String("error", error.Error(err)))
		return err
	}

	// Сжатие данных для передачи
	compressBody, err := compress.Compress(bufEncode.Bytes())
	if err != nil {
		logger.AgentLog.Error("Fail to comperess push data ", zap.String("error", error.Error(err)))
		return err
	}

	// Шифрование сжатых данных если установлен путь к публичному ключу
	crypto := config.GetCryptoGrapher()
	if crypto.PublicKeyIsSet() {
		compressBody, err = crypto.Encrypt(compressBody)
		if err != nil {
			logger.AgentLog.Error("fail to encode compressed data ", zap.String("error", error.Error(err)))
			return err
		}
	}

	// Создаю контекст с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), config.GetContextTimeout())
	defer cancel()

	// Подписываю данные отправляемые на сервер
	// Делаю не через middleware, чтобы агент подписывал именно нескомпресированный ответ
	hash, err := repositories.CalkHash(bufEncode.Bytes(), hasher.GetKey())
	if err != nil {
		logger.AgentLog.Error("Fail to calc hash ", zap.String("error", error.Error(err)))
		return err
	}
	logger.AgentLog.Debug("body and hash for forwarding to server ", zap.String("body", fmt.Sprintf("%x", bufEncode.Bytes())),
		zap.String("hash", hash), zap.String("key", hasher.GetKey()))

	// получаю ip адрес хоста для передачи на сервер в заголовке X-Real-IP
	hostAddress, err := ipgetter.Get()
	if err != nil {
		logger.AgentLog.Error("Fail to get host ip address ", zap.String("error", error.Error(err)))
		return err
	}

	url := fmt.Sprintf("%s/%s", address, action)
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Content-Encoding", "gzip").
		SetHeader("Accept-Encoding", "gzip").
		SetHeader("HashSHA256", hash).
		SetHeader("X-Real-IP", hostAddress).
		SetBody(compressBody).
		SetContext(ctx).
		Post(url)

	if err != nil {
		logger.AgentLog.Error("Push batch json metrics to server error ", zap.String("error", error.Error(err)))
		return err
	}

	logger.AgentLog.Debug("Get answer from server", zap.String("Content-Encoding", resp.Header().Get("Content-Encoding")),
		zap.String("statusCode", fmt.Sprintf("%d", resp.StatusCode())),
		zap.String("Content-Type", resp.Header().Get("Content-Type")),
		zap.String("HashSHA256", resp.Header().Get("HashSHA256")),
		zap.String("Content-Encoding", fmt.Sprint(resp.Header().Values("Content-Encoding"))))

	if resp.StatusCode() != http.StatusOK {
		logger.AgentLog.Error("Geting status is not 200 ", zap.String("statusCode", fmt.Sprintf("%d", resp.StatusCode())))
		return fmt.Errorf("status code is: %d %w", resp.StatusCode(), errors.New(resp.String()))
	}
	contentEncoding := resp.Header().Get("Content-Encoding")
	if strings.Contains(contentEncoding, "gzip") {
		logger.AgentLog.Debug("Get compress answer data in PushBatch function", zap.String("Content-Encoding", contentEncoding))
	} else {
		logger.AgentLog.Debug("Get uncompress answer data in PushBatch function", zap.String("Content-Encoding", contentEncoding))
	}

	responceMetrics := resp.Body()
	if !bytes.Equal(bufEncode.Bytes(), responceMetrics) {
		return fmt.Errorf("answer metric from server not equal pushing metric: get %d, want %d", responceMetrics, bufEncode.Bytes())
	}

	// Десериализую данные полученные от сервера, в основном для дебага
	var resJSON []repositories.Metric
	buRes := bytes.NewBuffer(responceMetrics)
	dec := json.NewDecoder(buRes)
	if err := dec.Decode(&resJSON); err != nil {
		logger.AgentLog.Error("decode decompress data from server error ", zap.String("error", error.Error(err)))
		return err
	}

	logger.AgentLog.Debug("Success push batch metrics in JSON format")
	return nil
}

// PrepareAndPushBatch - строит батч метрик и вызывает функцию для отправки батча на сервер в рамках одной передачи.
func PrepareAndPushBatch(address, action string, metrics *storage.MetricsStats, client *resty.Client) error {
	if metrics == nil {
		return fmt.Errorf("metrics is not initialize")
	}
	if client == nil {
		return fmt.Errorf("resty client is not initialize")
	}

	metrics.Lock()
	defer metrics.Unlock()

	// создаю слайс с метриками для отправки батчем
	metricsSlice := builder.BuildSlice(metrics)

	err := PushBatch(address, action, metricsSlice, client)
	if err != nil {
		logger.AgentLog.Error("Failed to push batch metrics", zap.String("action", "push metrics"), zap.String("error", error.Error(err)))
		return err
	}
	return nil
}
