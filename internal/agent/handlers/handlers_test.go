package handlers

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"syscall"
	"testing"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/mocks"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/repositories"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/compress"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/handlers"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-resty/resty/v2"
	"github.com/golang/mock/gomock"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			name    string
			args    args
			wantErr bool
			checkErrorFunction func(error) bool
		}{
			{
				name: "connection refused",
				args: args{
					ctx:    context.Background(),
					metric: connectionRefusedMetric,
					client: resty.New(),
				},
				wantErr: true,
				checkErrorFunction: isConnectionRefused,
			},
			{
				name: "connection exception",
				args: args{
					ctx:    context.Background(),
					metric: ConnectionExceptionMetric,
					client: resty.New(),
				},
				wantErr: true,
				checkErrorFunction: isDBTransportError,
			},
			{
				name: "EACCES",
				args: args{
					ctx:    context.Background(),
					metric: EACCESMetric,
					client: resty.New(),
				},
				wantErr: true,
				checkErrorFunction: isFileLockedError,
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

func TestBuildMetric(t *testing.T) {
	deltaPointer := func(delta int64) *int64 {
		return &delta
	}
	valuePointer := func(value float64) *float64 {
		return &value
	}
	type args struct {
		typeMetric  string
		nameMetric  string
		valueMetric string
	}
	tests := []struct {
		name       string
		args       args
		wantMetric repositories.Metric
		wantErr    bool
	}{
		{
			name: "success counter1",
			args: args{
				typeMetric:  "counter",
				nameMetric:  "counter1",
				valueMetric: "95738",
			},
			wantMetric: repositories.Metric{
				ID:    "counter1",
				MType: "counter",
				Delta: deltaPointer(95738),
				Value: nil,
			},
			wantErr: false,
		},
		{
			name: "error counter2",
			args: args{
				typeMetric:  "counter",
				nameMetric:  "counter2",
				valueMetric: "errorString",
			},
			wantMetric: repositories.Metric{},
			wantErr:    true,
		},
		{
			name: "success gauge1",
			args: args{
				typeMetric:  "gauge",
				nameMetric:  "gauge1",
				valueMetric: "95738.23598",
			},
			wantMetric: repositories.Metric{
				ID:    "gauge1",
				MType: "gauge",
				Value: valuePointer(95738.23598),
				Delta: nil,
			},
			wantErr: false,
		},
		{
			name: "error gauge2",
			args: args{
				typeMetric:  "gauge",
				nameMetric:  "gauge2",
				valueMetric: "errorString",
			},
			wantMetric: repositories.Metric{},
			wantErr:    true,
		},
		{
			name: "error wrongType",
			args: args{
				typeMetric:  "wrongType",
				nameMetric:  "counter1",
				valueMetric: "95738",
			},
			wantMetric: repositories.Metric{},
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMetric, err := BuildMetric(tt.args.typeMetric, tt.args.nameMetric, tt.args.valueMetric)
			if tt.wantErr == true {
				require.Error(t, err)
				return
			}
			assert.Equal(t, gotMetric, tt.wantMetric)
		})
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
		name    string
		args    args
		wantErr bool
		checkErrorFunction func(error) bool
	}{
		{
			name: "connection refused",
			args: args{
				ctx:          context.Background(),
				metricsSlice: connectionRefusedSlice,
				client:       resty.New(),
			},
			wantErr: true,
			checkErrorFunction: isConnectionRefused,
		},
		{
			name: "connection exception",
			args: args{
				ctx:          context.Background(),
				metricsSlice: ConnectionExceptionSlice,
				client:       resty.New(),
			},
			wantErr: true,
			checkErrorFunction: isDBTransportError,
		},
		{
			name: "EACCES",
			args: args{
				ctx:          context.Background(),
				metricsSlice: EACCESSlice,
				client:       resty.New(),
			},
			wantErr: true,
			checkErrorFunction: isFileLockedError,
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

func Test_isConnectionRefused(t *testing.T) {
	connectionRefusedError := &net.OpError{
		Op:  "dial",
		Net: "tcp",
		Err: &os.SyscallError{
			Syscall: "connect",
			Err:     syscall.ECONNREFUSED,
		},
	}

	erroWrapped := fmt.Errorf("Is wrapped error %d %w", 1, connectionRefusedError)

	tests := []struct {
		name string
		arg  error
		want bool
	}{
		{
			name: "success 1",
			arg:  connectionRefusedError,
			want: true,
		},
		{
			name: "error 1",
			arg:  nil,
			want: false,
		},
		{
			name: "success 2",
			arg:  erroWrapped,
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isConnectionRefused(tt.arg)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_isDBTransportError(t *testing.T) {
	errConnectionDoesNotExist := &pgconn.PgError{
		Code:    pgerrcode.ConnectionDoesNotExist,
		Message: "connection does not exist",
	}
	errConnectionDoesNotExistWrapped := fmt.Errorf("Is wrapped error %d %w", 1, errConnectionDoesNotExist)

	errConnectionFailure := &pgconn.PgError{
		Code:    pgerrcode.ConnectionFailure,
		Message: "connection failure",
	}

	errSQLClientUnableToEstablishSQLConnection := &pgconn.PgError{
		Code:    pgerrcode.SQLClientUnableToEstablishSQLConnection,
		Message: "SQL client unable to establish SQL connection",
	}

	errConnectionException := &pgconn.PgError{
		Code:    pgerrcode.ConnectionException,
		Message: "connection exception",
	}

	tests := []struct {
		name string
		arg  error
		want bool
	}{
		{
			name: "ConnectionDoesNotExist",
			arg:  errConnectionDoesNotExist,
			want: true,
		},
		{
			name: "error 1",
			arg:  nil,
			want: false,
		},
		{
			name: "ConnectionDoesNotExist wrapped",
			arg:  errConnectionDoesNotExistWrapped,
			want: true,
		},
		{
			name: "ConnectionDoesNotExist string",
			arg:  fmt.Errorf(error.Error(errConnectionDoesNotExist)),
			want: true,
		},
		{
			name: "ConnectionFailure",
			arg:  errConnectionFailure,
			want: true,
		},
		{
			name: "ConnectionFailure string",
			arg:  fmt.Errorf(error.Error(errConnectionFailure)),
			want: true,
		},
		{
			name: "SQLClientUnableToEstablishSQLConnection",
			arg:  errSQLClientUnableToEstablishSQLConnection,
			want: true,
		},
		{
			name: "SQLClientUnableToEstablishSQLConnection",
			arg:  fmt.Errorf(error.Error(errSQLClientUnableToEstablishSQLConnection)),
			want: true,
		},
		{
			name: "ConnectionException",
			arg:  errConnectionException,
			want: true,
		},
		{
			name: "ConnectionException",
			arg:  fmt.Errorf(error.Error(errConnectionException)),
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isDBTransportError(tt.arg)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_isFileLockedError(t *testing.T) {
	errEACCES := syscall.EACCES
	erroEACCESWrapped := fmt.Errorf("Is wrapped error %d %w", 1, errEACCES)
	errEROFS := syscall.EROFS
	errPermission := os.ErrPermission

	tests := []struct {
		name string
		arg  error
		want bool
	}{
		{
			name: "EACCES",
			arg:  errEACCES,
			want: true,
		},
		{
			name: "error 1",
			arg:  nil,
			want: false,
		},
		{
			name: "EACCESWrapped",
			arg:  erroEACCESWrapped,
			want: true,
		},
		{
			name: "EACCES string",
			arg:  fmt.Errorf(error.Error(errEACCES)),
			want: true,
		},
		{
			name: "EROFS",
			arg:  errEROFS,
			want: true,
		},
		{
			name: "EROFS string",
			arg:  fmt.Errorf(error.Error(errEROFS)),
			want: true,
		},
		{
			name: "Permission",
			arg:  errPermission,
			want: true,
		},
		{
			name: "Permission string",
			arg:  fmt.Errorf(error.Error(errPermission)),
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isFileLockedError(tt.arg)
			assert.Equal(t, tt.want, got)
		})
	}
}
