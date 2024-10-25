package checker

import (
	"fmt"
	"net"
	"os"
	"syscall"
	"testing"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
)

func TestIsConnectionRefused(t *testing.T) {
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
			got := IsConnectionRefused(tt.arg)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsDBTransportError(t *testing.T) {
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
			arg:  errConnectionDoesNotExist,
			want: true,
		},
		{
			name: "ConnectionFailure",
			arg:  errConnectionFailure,
			want: true,
		},
		{
			name: "ConnectionFailure string",
			arg:  errConnectionFailure,
			want: true,
		},
		{
			name: "SQLClientUnableToEstablishSQLConnection",
			arg:  errSQLClientUnableToEstablishSQLConnection,
			want: true,
		},
		{
			name: "SQLClientUnableToEstablishSQLConnection",
			arg:  errSQLClientUnableToEstablishSQLConnection,
			want: true,
		},
		{
			name: "ConnectionException",
			arg:  errConnectionException,
			want: true,
		},
		{
			name: "ConnectionException",
			arg:  errConnectionException,
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsDBTransportError(tt.arg)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsFileLockedError(t *testing.T) {
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
			arg:  errEACCES,
			want: true,
		},
		{
			name: "EROFS",
			arg:  errEROFS,
			want: true,
		},
		{
			name: "EROFS string",
			arg:  errEROFS,
			want: true,
		},
		{
			name: "Permission",
			arg:  errPermission,
			want: true,
		},
		{
			name: "Permission string",
			arg:  errPermission,
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsFileLockedError(tt.arg)
			assert.Equal(t, tt.want, got)
		})
	}
}
