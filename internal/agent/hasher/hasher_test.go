package hasher

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetKey(t *testing.T) {
	{
		SetKey("")
		assert.Equal(t, "", key)
	}
	{
		SetKey("secret key")
		assert.Equal(t, "secret key", key)
	}
}

func TestGetKey(t *testing.T) {
	{
		key = ""
		assert.Equal(t, "", GetKey())
	}
	{
		key = "secret key"
		assert.Equal(t, "secret key", GetKey())
	}
}

func TestVerifyHashMiddleware(t *testing.T) {
	// ключ не задан, подпись не проверяется
	{
		SetKey("")
		err := VerifyHashMiddleware(nil, nil)
		assert.NoError(t, err)
	}
	// подпись не проверяется, если статус ответа сервера не равен StatusOK
	{
		httpR := &http.Response{
			StatusCode: 400,
		}

		responce := resty.Response{
			RawResponse: httpR,
		}
		err := VerifyHashMiddleware(nil, &responce)
		assert.NoError(t, err)
	}
	// отсутствует хэш в заголовке ответа
	{
		// устанавливаю секретный коюч
		SetKey("secret key")
		httpH := make(http.Header, 0)
		httpH.Add("HashSHA256", "")

		httpR := &http.Response{
			StatusCode: 200,
			Header:     httpH,
		}

		responce := resty.Response{
			RawResponse: httpR,
		}
		err := VerifyHashMiddleware(nil, &responce)
		assert.Error(t, err)
	}
	// тест с успешной проверкой подписи
	{
		// устанавливаю секретный коюч
		SetKey("secret key")

		// устанавливаю тело ответа от сервера
		body := []byte("my test information for hashing and checking")
		// подписываю тело алгоритмом HMAC, используя SHA-256
		h := hmac.New(sha256.New, []byte(GetKey()))
		_, err := h.Write(body)
		require.NoError(t, err)
		hash := h.Sum(nil)

		httpH := make(http.Header, 0)
		httpH.Add("HashSHA256", hex.EncodeToString(hash[:]))
		httpR := &http.Response{
			StatusCode: 200,
			Header:     httpH,
		}

		responce := resty.Response{
			RawResponse: httpR,
		}
		responce.SetBody(body)
		err = VerifyHashMiddleware(nil, &responce)
		assert.NoError(t, err)
	}
	// тест с неуспешной проверкой подписи, разные секретные ключи
	{
		// устанавливаю секретный коюч
		SetKey("secret key")

		// устанавливаю тело ответа от сервера
		body := []byte("my test information for hashing and checking")
		// подписываю тело алгоритмом HMAC, используя SHA-256
		h := hmac.New(sha256.New, []byte("wrong secret key"))
		_, err := h.Write(body)
		require.NoError(t, err)
		hash := h.Sum(nil)

		httpH := make(http.Header, 0)
		httpH.Add("HashSHA256", hex.EncodeToString(hash[:]))
		httpR := &http.Response{
			StatusCode: 200,
			Header:     httpH,
		}

		responce := resty.Response{
			RawResponse: httpR,
		}
		responce.SetBody(body)
		err = VerifyHashMiddleware(nil, &responce)
		assert.Error(t, err)
	}
	// тест с неправильным форматом хэша
	{
		// устанавливаю секретный коюч
		SetKey("secret key")

		// устанавливаю тело ответа от сервера
		body := []byte("my test information for hashing and checking")
		// подписываю тело алгоритмом HMAC, используя SHA-256
		h := hmac.New(sha256.New, []byte(GetKey()))
		_, err := h.Write(body)
		require.NoError(t, err)
		hash := h.Sum(nil)

		httpH := make(http.Header, 0)
		httpH.Add("HashSHA256", string(hash))
		httpR := &http.Response{
			StatusCode: 200,
			Header:     httpH,
		}

		responce := resty.Response{
			RawResponse: httpR,
		}
		responce.SetBody(body)
		err = VerifyHashMiddleware(nil, &responce)
		assert.Error(t, err)
	}
}
