package checker

import (
	"errors"
	"os"
	"strings"
	"syscall"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/logger"
)

// IsConnectionRefused - проверка того, что ошибка это "connect: connection refused"
func IsConnectionRefused(err error) bool {
	if err == nil {
		return false
	}
	res := errors.Is(err, syscall.ECONNREFUSED) || strings.Contains(err.Error(), "dial tcp: connect: connection refused")
	if res {
		logger.AgentLog.Debug("error isConnectionRefused")
	}
	return res
}

// IsDBTransportError - проверяет, что ошибка относится к DBTransportError
func IsDBTransportError(err error) bool {
	if err == nil {
		return false
	}
	asPgError := false
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		asPgError = (pgerrcode.IsConnectionException(pgErr.Code) ||
			pgErr.Code == pgerrcode.ConnectionDoesNotExist ||
			pgErr.Code == pgerrcode.ConnectionFailure ||
			pgErr.Code == pgerrcode.SQLClientUnableToEstablishSQLConnection)
	}
	asString := false
	asString = strings.Contains(err.Error(), "connection exception") ||
		strings.Contains(err.Error(), "connection does not exist") ||
		strings.Contains(err.Error(), "connection failure") ||
		strings.Contains(err.Error(), "SQL client unable to establish SQL connection")
	res := asPgError || asString
	if res {
		logger.AgentLog.Debug("error isDBTransportError")
	}
	return res
}

// IsFileLockedError - проверяет, что ошибка относится к FileLockedError
func IsFileLockedError(err error) bool {
	if err == nil {
		return false
	}
	asError := false
	asError = errors.Is(err, syscall.EACCES) ||
		errors.Is(err, syscall.EROFS) ||
		errors.Is(err, os.ErrPermission)

	asString := false
	asString = strings.Contains(err.Error(), "permission denied") ||
		strings.Contains(err.Error(), "read-only file system")
	res := asError || asString
	if res {
		logger.AgentLog.Debug("error isFileLockedError")
	}
	return res
}
