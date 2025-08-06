package pusher

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"syscall"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-resty/resty/v2"
	"github.com/golang/mock/gomock"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/errors/checker"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/mocks"
	agentStorage "github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/storage"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/repositories"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/compress"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/handlers"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/storage"
)

func TestPush(t *testing.T) {
	stor := storage.NewDefaultMemStorage()

	type args struct {
		action      string
		typeMetric  string
		nameMetric  string
		valueMetric string
		client      *resty.Client
	}
	tests := []struct {
		name     string
		args     args
		wantStor *storage.MemStorage
		wantErr  bool
	}{
		{
			name: "Count #1",
			args: args{
				action:      "update",
				typeMetric:  "counter",
				nameMetric:  "counter1",
				valueMetric: "4",
				client:      resty.New(),
			},
			wantStor: storage.NewMemStorage(nil, map[string]int64{"counter1": 4}),
			wantErr:  false,
		},
		{
			name: "Count error #1",
			args: args{
				action:      "update",
				typeMetric:  "wrangtype",
				nameMetric:  "counter1",
				valueMetric: "4",
				client:      resty.New(),
			},
			wantStor: storage.NewMemStorage(nil, map[string]int64{"counter1": 4}),
			wantErr:  true,
		},
		{
			name: "Gauge #1",
			args: args{
				action:      "update",
				typeMetric:  "gauge",
				nameMetric:  "gauge1",
				valueMetric: "3.14",
				client:      resty.New(),
			},
			wantStor: storage.NewMemStorage(map[string]float64{"gauge1": 3.14}, map[string]int64{"counter1": 4}),
			wantErr:  false,
		},
		{
			name: "Gauge error #1",
			args: args{
				action:      "update",
				typeMetric:  "wrangtype",
				nameMetric:  "gauge1",
				valueMetric: "3.14",
				client:      resty.New(),
			},
			wantStor: storage.NewMemStorage(map[string]float64{"gauge1": 3.14}, map[string]int64{"counter1": 4}),
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Post("/update/{metricType}/{metricName}/{metricValue}", func(res http.ResponseWriter, req *http.Request) {
				handlers.UpdateMetrics(res, req, stor)
			})

			// Создаём тестовый сервер
			ts := httptest.NewServer(r)
			defer ts.Close()

			if err := Push(ts.URL, tt.args.action, tt.args.typeMetric, tt.args.nameMetric, tt.args.valueMetric, tt.args.client); (err != nil) != tt.wantErr {
				t.Errorf("PushJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
			wantAll, err := tt.wantStor.GetAllMetrics(context.Background())
			require.NoError(t, err)
			getAll, errGet := stor.GetAllMetrics(context.Background())
			require.NoError(t, errGet)
			assert.Equal(t, wantAll, getAll)

			wantAllSlice, errWantSlice := tt.wantStor.GetAllMetricsSlice(context.Background())
			require.NoError(t, errWantSlice)
			getAllSlice, errGetSlice := stor.GetAllMetricsSlice(context.Background())
			require.NoError(t, errGetSlice)
			assert.Equal(t, wantAllSlice, getAllSlice)
		})
	}
}

func TestPushJSON(t *testing.T) {
	{
		stor := storage.NewDefaultMemStorage()

		type args struct {
			action      string
			typeMetric  string
			nameMetric  string
			valueMetric string
			client      *resty.Client
		}
		tests := []struct {
			name     string
			args     args
			wantStor *storage.MemStorage
			wantErr  bool
		}{
			{
				name: "Count #1",
				args: args{
					action:      "update",
					typeMetric:  "counter",
					nameMetric:  "counter1",
					valueMetric: "4",
					client:      resty.New(),
				},
				wantStor: storage.NewMemStorage(nil, map[string]int64{"counter1": 4}),
				wantErr:  false,
			},
			{
				name: "Count error #1",
				args: args{
					action:      "update",
					typeMetric:  "wrangtype",
					nameMetric:  "counter1",
					valueMetric: "4",
					client:      resty.New(),
				},
				wantStor: storage.NewMemStorage(nil, map[string]int64{"counter1": 4}),
				wantErr:  true,
			},
			{
				name: "Gauge #1",
				args: args{
					action:      "update",
					typeMetric:  "gauge",
					nameMetric:  "gauge1",
					valueMetric: "3.14",
					client:      resty.New(),
				},
				wantStor: storage.NewMemStorage(map[string]float64{"gauge1": 3.14}, map[string]int64{"counter1": 4}),
				wantErr:  false,
			},
			{
				name: "Gauge error #1",
				args: args{
					action:      "update",
					typeMetric:  "wrangtype",
					nameMetric:  "gauge1",
					valueMetric: "3.14",
					client:      resty.New(),
				},
				wantStor: storage.NewMemStorage(map[string]float64{"gauge1": 3.14}, map[string]int64{"counter1": 4}),
				wantErr:  true,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				r := chi.NewRouter()
				r.Post("/update", compress.GzipMiddleware(handlers.UpdateMetricsJSONHandler(stor)))

				// Создаём тестовый сервер
				ts := httptest.NewServer(r)
				defer ts.Close()

				if err := PushJSON(ts.URL, tt.args.action, tt.args.typeMetric, tt.args.nameMetric, tt.args.valueMetric, tt.args.client); (err != nil) != tt.wantErr {
					t.Errorf("PushJSON() error = %v, wantErr %v", err, tt.wantErr)
				}
				wantAll, err := tt.wantStor.GetAllMetrics(context.Background())
				require.NoError(t, err)
				getAll, errGet := stor.GetAllMetrics(context.Background())
				require.NoError(t, errGet)
				assert.Equal(t, wantAll, getAll)

				wantAllSlice, errWantSlice := tt.wantStor.GetAllMetricsSlice(context.Background())
				require.NoError(t, errWantSlice)
				getAllSlice, errGetSlice := stor.GetAllMetricsSlice(context.Background())
				require.NoError(t, errGetSlice)
				assert.Equal(t, wantAllSlice, getAllSlice)
			})
		}
	}
	// Проверяю какую ошибку получает хэндлер агента в случае retryable ошибки на стороне сервера
	{
		deltaPointer := func(delta int64) *int64 {
			return &delta
		}
		// создаём контроллер
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := mocks.NewMockServerRepo(ctrl)

		connectionRefusedMetric := repositories.Metric{
			ID:    "connectionRefused",
			MType: "counter",
			Delta: deltaPointer(123),
		}

		connectionRefusedError := &net.OpError{
			Op:  "dial",
			Net: "tcp",
			Err: &os.SyscallError{
				Syscall: "connect",
				Err:     syscall.ECONNREFUSED,
			},
		}
		m.EXPECT().AddCounter(gomock.Any(), connectionRefusedMetric.ID, gomock.Any()).Return(connectionRefusedError)

		ConnectionExceptionMetric := repositories.Metric{
			ID:    "ConnectionException",
			MType: "counter",
			Delta: deltaPointer(123),
		}
		ConnectionExceptionError := &pgconn.PgError{
			Code:    pgerrcode.ConnectionException,
			Message: "connection exception",
		}
		m.EXPECT().AddCounter(gomock.Any(), ConnectionExceptionMetric.ID, gomock.Any()).Return(ConnectionExceptionError)

		EACCESMetric := repositories.Metric{
			ID:    "EACCES",
			MType: "counter",
			Delta: deltaPointer(123),
		}
		EACCESError := syscall.EACCES
		m.EXPECT().AddCounter(gomock.Any(), EACCESMetric.ID, gomock.Any()).Return(EACCESError)

		type args struct {
			ctx    context.Context
			metric repositories.Metric
			client *resty.Client
		}
		tests := []struct {
			name               string
			args               args
			wantErr            bool
			checkErrorFunction func(error) bool
		}{
			{
				name: "connection refused",
				args: args{
					ctx:    context.Background(),
					metric: connectionRefusedMetric,
					client: resty.New(),
				},
				wantErr:            true,
				checkErrorFunction: checker.IsConnectionRefused,
			},
			{
				name: "connection exception",
				args: args{
					ctx:    context.Background(),
					metric: ConnectionExceptionMetric,
					client: resty.New(),
				},
				wantErr:            true,
				checkErrorFunction: checker.IsDBTransportError,
			},
			{
				name: "EACCES",
				args: args{
					ctx:    context.Background(),
					metric: EACCESMetric,
					client: resty.New(),
				},
				wantErr:            true,
				checkErrorFunction: checker.IsFileLockedError,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				r := chi.NewRouter()
				r.Post("/update", compress.GzipMiddleware(handlers.UpdateMetricsJSONHandler(m)))

				// Создаём тестовый сервер
				ts := httptest.NewServer(r)
				defer ts.Close()

				err := PushJSON(ts.URL, "update", tt.args.metric.MType, tt.args.metric.ID, "123", tt.args.client)

				if tt.wantErr == true {
					require.Error(t, err)
					assert.Equal(t, true, tt.checkErrorFunction(err))
				}
			})
		}
	}
}

func TestPushBatch(t *testing.T) {
	deltaPointer := func(delta int64) *int64 {
		return &delta
	}
	// создаём контроллер
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := mocks.NewMockServerRepo(ctrl)

	connectionRefusedSlice := []repositories.Metric{
		{
			ID:    "connectionRefused",
			MType: "counter",
			Delta: deltaPointer(123),
		},
	}
	connectionRefusedError := &net.OpError{
		Op:  "dial",
		Net: "tcp",
		Err: &os.SyscallError{
			Syscall: "connect",
			Err:     syscall.ECONNREFUSED,
		},
	}
	m.EXPECT().AddMetricsFromSlice(gomock.Any(), connectionRefusedSlice).Return(connectionRefusedError)

	ConnectionExceptionSlice := []repositories.Metric{
		{
			ID:    "ConnectionException",
			MType: "counter",
			Delta: deltaPointer(123),
		},
	}
	ConnectionExceptionError := &pgconn.PgError{
		Code:    pgerrcode.ConnectionException,
		Message: "connection exception",
	}
	m.EXPECT().AddMetricsFromSlice(gomock.Any(), ConnectionExceptionSlice).Return(ConnectionExceptionError)

	EACCESSlice := []repositories.Metric{
		{
			ID:    "EACCES",
			MType: "counter",
			Delta: deltaPointer(123),
		},
	}
	EACCESError := syscall.EACCES
	m.EXPECT().AddMetricsFromSlice(gomock.Any(), EACCESSlice).Return(EACCESError)

	type args struct {
		ctx          context.Context
		metricsSlice []repositories.Metric
		client       *resty.Client
	}
	tests := []struct {
		name               string
		args               args
		wantErr            bool
		checkErrorFunction func(error) bool
	}{
		{
			name: "connection refused",
			args: args{
				ctx:          context.Background(),
				metricsSlice: connectionRefusedSlice,
				client:       resty.New(),
			},
			wantErr:            true,
			checkErrorFunction: checker.IsConnectionRefused,
		},
		{
			name: "connection exception",
			args: args{
				ctx:          context.Background(),
				metricsSlice: ConnectionExceptionSlice,
				client:       resty.New(),
			},
			wantErr:            true,
			checkErrorFunction: checker.IsDBTransportError,
		},
		{
			name: "EACCES",
			args: args{
				ctx:          context.Background(),
				metricsSlice: EACCESSlice,
				client:       resty.New(),
			},
			wantErr:            true,
			checkErrorFunction: checker.IsFileLockedError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Post("/updates/", compress.GzipMiddleware(handlers.UpdateMetricsBatchHandler(m)))

			// Создаём тестовый сервер
			ts := httptest.NewServer(r)
			defer ts.Close()

			err := PushBatch(ts.URL, "updates/", tt.args.metricsSlice, tt.args.client)

			if tt.wantErr == true {
				require.Error(t, err)
				assert.Equal(t, true, tt.checkErrorFunction(err))
			}
		})
	}
}

func TestPrepareAndPushBatch(t *testing.T) {
	// error: storage is nil
	{
		err := PrepareAndPushBatch("", "", nil, nil)
		require.Error(t, err)
	}
	// error: resty client is nil
	{
		metrics := &agentStorage.MetricsStats{}
		err := PrepareAndPushBatch("", "", metrics, nil)
		require.Error(t, err)
	}
}
