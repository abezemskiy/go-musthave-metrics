package main

import (
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFlagsWithFlags(t *testing.T) {
	// Сохраняем оригинальные значения флагов
	originalArgs := os.Args
	os.Args = []string{"cmd", "-a", ":9000", "-r", "120", "-p", "240", "-log=info", "-l", "3", "-k", "secret",
		"-crypto-key", "/crypto/key/path", "-protocol", "grpc"}
	defer func() { os.Args = originalArgs }()

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	parseFlags()

	assert.Equal(t, ":9000", flagNetAddr)
	assert.Equal(t, 120, *reportInterval)
	assert.Equal(t, 240, *pollInterval)
	assert.Equal(t, "info", flagLogLevel)
	assert.Equal(t, 3, *rateLimit)
	assert.Equal(t, "secret", flagKey)
	assert.Equal(t, "/crypto/key/path", cryptoKey)
	assert.Equal(t, "grpc", flagProtocol)
}

func TestParseFlagsPriority(t *testing.T) {
	// Устанавливаем переменные окружения
	os.Setenv("ADDRESS", ":8000")
	os.Setenv("REPORT_INTERVAL", "200")
	os.Setenv("POLL_INTERVAL", "314")
	os.Setenv("AGENT_LOG_LEVEL", "debug")
	os.Setenv("RATE_LIMIT", "23")
	os.Setenv("CRYPTO_KEY", "/secret/crypto/key")
	os.Setenv("PROTOCOL", "grpc")

	defer func() {
		os.Unsetenv("ADDRESS")
		os.Unsetenv("REPORT_INTERVAL")
		os.Unsetenv("POLL_INTERVAL")
		os.Unsetenv("AGENT_LOG_LEVEL")
		os.Unsetenv("RATE_LIMIT")
		os.Unsetenv("CRYPTO_KEY")
		os.Unsetenv("PROTOCOL")
	}()

	// Создаём временный конфигурационный файл
	configFile := "./test_config.json"
	configContent := `{
        "address": "localhost:8082",
        "report_interval": "17s",
        "poll_interval": "9s",
		"crypto_key": "/config/file/secret/crypto/key",
		"protocol": "grpc"
    }`
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)
	defer os.Remove(configFile)

	// Сохраняем оригинальные значения флагов
	originalArgs := os.Args
	os.Args = []string{"cmd", "-a", ":9000", "-r", "120", "-p", "240", "-log=info", "-l", "3", "-k", "secret", "-crypto-key",
		"/crypto/key/path", "-c", "./test_config.json", "-protocol", "grpc"}
	defer func() { os.Args = originalArgs }()

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	parseFlags()

	assert.Equal(t, "localhost:8082", flagNetAddr)
	assert.Equal(t, 17, *reportInterval)
	assert.Equal(t, 9, *pollInterval)
	assert.Equal(t, "debug", flagLogLevel)
	assert.Equal(t, 23, *rateLimit)
	assert.Equal(t, "secret", flagKey)
	assert.Equal(t, "/config/file/secret/crypto/key", cryptoKey)
	assert.Equal(t, "grpc", flagProtocol)
}

func TestParseEnvironment(t *testing.T) {
	// Устанавливаем переменные окружения
	os.Setenv("ADDRESS", ":8000")
	os.Setenv("REPORT_INTERVAL", "200")
	os.Setenv("POLL_INTERVAL", "314")
	os.Setenv("AGENT_LOG_LEVEL", "debug")
	os.Setenv("RATE_LIMIT", "23")
	os.Setenv("KEY", "secret")
	os.Setenv("CRYPTO_KEY", "/secret/crypto/key")
	os.Setenv("PROTOCOL", "grpc")

	defer func() {
		os.Unsetenv("ADDRESS")
		os.Unsetenv("REPORT_INTERVAL")
		os.Unsetenv("POLL_INTERVAL")
		os.Unsetenv("AGENT_LOG_LEVEL")
		os.Unsetenv("RATE_LIMIT")
		os.Unsetenv("KEY")
		os.Unsetenv("CRYPTO_KEY")
		os.Unsetenv("PROTOCOL")
	}()

	parseEnvironment()

	assert.Equal(t, ":8000", flagNetAddr)
	assert.Equal(t, 200, *reportInterval)
	assert.Equal(t, 314, *pollInterval)
	assert.Equal(t, "debug", flagLogLevel)
	assert.Equal(t, 23, *rateLimit)
	assert.Equal(t, "secret", flagKey)
	assert.Equal(t, "/secret/crypto/key", cryptoKey)
	assert.Equal(t, "grpc", flagProtocol)
}

func TestParseConfigFile(t *testing.T) {
	reportInterval = new(int)
	pollInterval = new(int)

	testFlagNetAddr := "localhost:8081"
	testReportInterval := 21
	testPollInterval := 3
	testFlagCryptoKey := "test crypto key"
	testFlagProtocol := "grpc"

	createFile := func(name string) {
		data := fmt.Sprintf("{\"address\": \"%s\",\"report_interval\": \"%ds\",\"poll_interval\": \"%ds\",\"crypto_key\": \"%s\",\"protocol\": \"%s\"}",
			testFlagNetAddr, testReportInterval, testPollInterval, testFlagCryptoKey, testFlagProtocol)
		f, err := os.Create(name)
		require.NoError(t, err)
		_, err = f.Write([]byte(data))
		require.NoError(t, err)
	}
	nameFile := "./test_config.json"
	createFile(nameFile)

	flagConfigFile = nameFile
	parseConfigFile()

	assert.Equal(t, testFlagNetAddr, flagNetAddr)
	assert.Equal(t, testReportInterval, *reportInterval)
	assert.Equal(t, testPollInterval, *pollInterval)
	assert.Equal(t, testFlagCryptoKey, cryptoKey)
	assert.Equal(t, testFlagProtocol, flagProtocol)

	err := os.Remove(nameFile)
	require.NoError(t, err)
}
