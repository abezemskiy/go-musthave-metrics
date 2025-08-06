package hasher

import (
	"bytes"
	"errors"
	mathRand "math/rand"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/repositories"
)

func TestSetKey(t *testing.T) {
	key = "secret key"
	newKey := "new secret key"
	SetKey(newKey)
	assert.Equal(t, newKey, key)
}

func TestGetKey(t *testing.T) {
	key = "second key"
	assert.Equal(t, key, GetKey())
}

type errorReader struct{}

func (e *errorReader) Read(_ []byte) (int, error) {
	return 0, errors.New("simulated connection error")
}

func TestHashMiddleware(t *testing.T) {
	// функция для генерации тестового тела
	randomData := func(rnd *mathRand.Rand, n int) []byte {
		b := make([]byte, n)
		_, err := rnd.Read(b)
		require.NoError(t, err)
		return b
	}

	testHandler := func() http.HandlerFunc {
		fn := func(res http.ResponseWriter, _ *http.Request) {
			res.WriteHeader(200)
		}
		return fn
	}

	{
		// данные для "successful hashing"
		rnd := mathRand.New(mathRand.NewSource(113))
		testBody1 := randomData(rnd, 256)
		key1 := "secret key for test1"
		hash1, err := repositories.CalkHash(testBody1, key1)
		require.NoError(t, err)

		type want struct {
			hash       string
			statusCode int
		}
		type header struct {
			key   string
			value string
		}
		tests := []struct {
			name   string
			data   []byte
			key    string
			want   want
			header []header
		}{
			{
				name: "successful hashing",
				data: testBody1,
				key:  key1,
				want: want{
					hash:       hash1,
					statusCode: 200,
				},
				header: []header{{key: "Hash", value: "exist"}, {key: "HashSHA256", value: hash1}},
			},
			{
				name: "don't set key",
				data: testBody1,
				key:  "",
				want: want{
					hash:       hash1,
					statusCode: 200,
				},
				header: []header{{key: "Hash", value: "exist"}, {key: "HashSHA256", value: "wrong hash"}},
			},
			{
				name: "don't set key",
				data: testBody1,
				key:  "wrong key",
				want: want{
					hash:       hash1,
					statusCode: 200,
				},
				header: []header{{key: "Hash", value: "none"}, {key: "HashSHA256", value: "wrong hash"}},
			},
			{
				name: "don't set key",
				data: testBody1,
				key:  "wrong key",
				want: want{
					hash:       hash1,
					statusCode: 200,
				},
				header: []header{{key: "Hash", value: "exist"}, {key: "HashSHA256", value: ""}},
			},
			{
				name: "failure hashing#1",
				data: testBody1,
				key:  key1,
				want: want{
					hash:       hash1,
					statusCode: 500,
				},
				header: []header{{key: "Hash", value: "exist"}, {key: "HashSHA256", value: "wrong hash"}},
			},
			{
				name: "failure hashing#2",
				data: testBody1,
				key:  "wrong key",
				want: want{
					hash:       hash1,
					statusCode: 400,
				},
				header: []header{{key: "Hash", value: "exist"}, {key: "HashSHA256", value: hash1}},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				r := chi.NewRouter()
				r.Post("/test", HashMiddleware(testHandler()))

				SetKey(tt.key)

				request := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(tt.data))
				for _, h := range tt.header {
					request.Header.Set(h.key, h.value)
				}

				w := httptest.NewRecorder()
				r.ServeHTTP(w, request)

				res := w.Result()
				defer res.Body.Close() // Закрываем тело ответа
				// проверяем код ответа
				assert.Equal(t, tt.want.statusCode, res.StatusCode)
			})
		}
	}
	// тест с ошибкой при чтении запроса
	{
		r := chi.NewRouter()
		r.Post("/test", HashMiddleware(testHandler()))

		SetKey("wrong secret key")

		request := httptest.NewRequest(http.MethodPost, "/test", &errorReader{})
		request.Header.Set("Hash", "exist")
		request.Header.Set("HashSHA256", "wrong hash")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, request)

		res := w.Result()
		defer res.Body.Close() // Закрываем тело ответа
		// проверяем код ответа
		assert.Equal(t, 500, res.StatusCode)
	}
}
