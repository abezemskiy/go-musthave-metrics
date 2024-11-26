package encrypt

import (
	"bytes"
	"io"
	mathRand "math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/tools/encryption"
)

func TestSetCryptoGrapher(t *testing.T) {
	crypto := encryption.Initialize("/path/to/public/key", "/path/to/private/key")
	SetCryptoGrapher(crypto)

	assert.Equal(t, crypto.PublicKeyIsSet(), cryptoGrapher.PublicKeyIsSet())
}

func TestMiddleware(t *testing.T) {
	// функция для очистки файлов с ключами
	removeFile := func(file string) {
		err := os.Remove(file)
		require.NoError(t, err)
	}

	randomData := func(rnd *mathRand.Rand, n int) []byte {
		b := make([]byte, n)
		_, err := rnd.Read(b)
		require.NoError(t, err)
		return b
	}

	testHandler := func() http.HandlerFunc {
		fn := func(res http.ResponseWriter, req *http.Request) {
			res.WriteHeader(200)
			body, err := io.ReadAll(req.Body)
			require.NoError(t, err)
			_, err = res.Write(body)
			require.NoError(t, err)
		}
		return fn
	}

	// генерация ключей
	pathKeys := "."
	err := encryption.GenerateKeys(pathKeys)
	require.NoError(t, err)

	// Создаю струткуру с ключами шифрования
	SetCryptoGrapher(encryption.Initialize(pathKeys+"/public_key.pem", pathKeys+"/private_key.pem"))

	// Success decryption test------------------------------
	rnd := mathRand.New(mathRand.NewSource(103))
	body := randomData(rnd, 256)
	ecryptedData, err := cryptoGrapher.Encrypt(body)
	require.NoError(t, err)

	type want struct {
		decryptedData []byte
		statusCode    int
	}

	tests := []struct {
		name    string
		request string
		data    []byte
		want    want
	}{
		{
			name:    "Success decryption",
			request: "/test",
			data:    ecryptedData,
			want: want{
				decryptedData: body,
				statusCode:    200,
			},
		},
		{
			name:    "Empty data",
			request: "/test",
			data:    nil,
			want: want{
				decryptedData: nil,
				statusCode:    500,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Post("/test", Middleware(testHandler()))

			request := httptest.NewRequest(http.MethodPost, tt.request, bytes.NewReader(tt.data))
			w := httptest.NewRecorder()
			r.ServeHTTP(w, request)

			res := w.Result()
			defer res.Body.Close() // Закрываем тело ответа
			// проверяем код ответа
			assert.Equal(t, tt.want.statusCode, res.StatusCode)

			if tt.want.statusCode == 200 {
				getBody, err := io.ReadAll(res.Body)
				require.NoError(t, err)
				defer res.Body.Close()

				assert.Equal(t, tt.want.decryptedData, getBody)
			}
		})
	}

	defer removeFile(pathKeys + "/private_key.pem")
	defer removeFile(pathKeys + "/public_key.pem")
}
