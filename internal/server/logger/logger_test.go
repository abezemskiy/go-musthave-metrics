package logger

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestInitialize(t *testing.T) {
	tests := []struct {
		name        string
		level       string
		expectError bool
	}{
		{"ValidDebugLevel", "debug", false},
		{"ValidInfoLevel", "info", false},
		{"ValidWarnLevel", "warn", false},
		{"InvalidLevel", "invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Инициализирую логгер
			err := Initialize(tt.level)

			// Проверяю наличие ошибки
			if tt.expectError {
				require.Error(t, err)
			}
			if !tt.expectError {
				require.NoError(t, err)
			}

			// Если ошибок нет, проверяю уровень логирования
			if !tt.expectError {
				require.NotEqual(t, nil, ServerLog)

				// получаю текущий уровень логгера
				level := ServerLog.Core().Enabled(zap.DebugLevel)
				expectedLevel := tt.level == "debug" // уровень "debug" должен быть доступен только при debug
				require.Equal(t, expectedLevel, level)
			}
		})
	}
}

func TestInitializeInvalidConfig(t *testing.T) {
	// Создаю буфер для вывода логов
	var buf bytes.Buffer
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(&buf),
		zap.DebugLevel,
	)
	ServerLog = zap.New(core)

	// Инициализация с некорректным уровнем
	err := Initialize("invalid")
	require.Error(t, err)

	// Проверяю, что глобальный логгер не был перезаписан
	require.Equal(t, false, buf.Len() > 0)
}
