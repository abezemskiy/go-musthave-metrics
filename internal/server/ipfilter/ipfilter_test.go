package ipfilter

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

func TestSetTrustedSubnet(t *testing.T) {
	trustedSubnet = ""
	SetTrustedSubnet("192.168.0.0/24")
	assert.Equal(t, "192.168.0.0/24", trustedSubnet)
}

func TestGetTrustedSubnet(t *testing.T) {
	trustedSubnet = "192.168.17.0/24"
	assert.Equal(t, "192.168.17.0/24", getTrustedSubnet())
}

func TestMiddleware(t *testing.T) {
	testHandler := func() http.HandlerFunc {
		fn := func(res http.ResponseWriter, req *http.Request) {
			res.WriteHeader(200)
		}
		return fn
	}

	tests := []struct {
		name     string
		subNet   string
		realIP   string
		wantCode int
	}{
		{
			name:     "in trusted",
			subNet:   "192.168.0.0/24",
			realIP:   "192.168.0.235",
			wantCode: 200,
		},
		{
			name:     "not in trusted",
			subNet:   "192.168.1.0/24",
			realIP:   "192.168.0.235",
			wantCode: 403,
		}, {
			name:     "wrong subNet",
			subNet:   "wrong.sub.net",
			realIP:   "192.168.0.235",
			wantCode: 500,
		},
		{
			name:     "empty subNet",
			subNet:   "",
			realIP:   "192.168.0.235",
			wantCode: 200,
		},
		{
			name:     "wrong real ip",
			subNet:   "192.168.14.0/16",
			realIP:   "wrong.real.ip",
			wantCode: 500,
		},
		{
			name:     "empty real ip",
			subNet:   "192.168.14.0/16",
			realIP:   "",
			wantCode: 403,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetTrustedSubnet(tt.subNet)

			r := chi.NewRouter()
			r.Post("/test", Middleware(testHandler()))

			request := httptest.NewRequest(http.MethodPost, "/test", nil)
			w := httptest.NewRecorder()
			request.Header.Add("X-Real-IP", tt.realIP)
			r.ServeHTTP(w, request)

			res := w.Result()
			defer res.Body.Close() // Закрываем тело ответа
			// проверяем код ответа
			assert.Equal(t, tt.wantCode, res.StatusCode)
		})
	}
}
