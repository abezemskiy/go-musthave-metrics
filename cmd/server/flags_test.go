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
	os.Args = []string{"cmd", "-a", ":9000", "-grpc-address", ":9002", "-l", "debug", "-i", "120", "-f", "./metrics.json", "-r=false", "-d", "db_dsn",
		"-k", "secret", "-crypto-key", "./path/to/crypto/key", "-t", "192.168.0.2/24"}
	defer func() { os.Args = originalArgs }()

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	result := parseFlags()

	assert.Equal(t, ":9000", flagNetAddr)
	assert.Equal(t, ":9002", flagGRPCNetAddr)
	assert.Equal(t, "debug", flagLogLevel)
	assert.Equal(t, 120, flagStoreInterval)
	assert.Equal(t, "./metrics.json", flagFileStoragePath)
	assert.Equal(t, false, flagRestore)
	assert.Equal(t, "db_dsn", flagDatabaseDsn)
	assert.Equal(t, "secret", flagKey)
	assert.Equal(t, "./path/to/crypto/key", flagCryptoKey)
	assert.Equal(t, SAVEINDATABASE, result)
	assert.Equal(t, "192.168.0.2/24", flagTrustedSubnet)
}

func TestParseFlagsPriority(t *testing.T) {
	// Устанавливаем переменные окружения
	os.Setenv("ADDRESS", ":8000")
	os.Setenv("GRPC_ADDRESS", ":8009")
	os.Setenv("SERVER_LOG_LEVEL", "debug")
	os.Setenv("STORE_INTERVAL", "200")
	os.Setenv("RESTORE", "true")
	os.Setenv("TRUSTED_SUBNET", "192.168.0.4/24")

	defer func() {
		os.Unsetenv("ADDRESS")
		os.Unsetenv("GRPC_ADDRESS")
		os.Unsetenv("STORE_INTERVAL")
		os.Unsetenv("TRUSTED_SUBNET")
		os.Unsetenv("SERVER_LOG_LEVEL")
	}()

	// Создаём временный конфигурационный файл
	configFile := "./test_config.json"
	configContent := `{
        "address": ":7000",
		"grpc_address": ":8099",
		"log_level": "error",
        "restore": false,
        "store_interval": "60s",
		"trusted_subnet": "192.168.0.6/24"
    }`
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)
	defer os.Remove(configFile)

	// Сохраняем оригинальные значения флагов
	originalArgs := os.Args
	os.Args = []string{"cmd", "-a", ":9000", "-grpc-address", ":8999", "-l", "info", "-i", "120", "-f",
		"./metrics.json", "-r=false", "-d", "db_dsn", "-k", "secret", "-c",
		"./test_config.json", "-t", "192.168.0.8/24"}
	defer func() { os.Args = originalArgs }()

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	parseFlags()

	// Файл имеет приоритет
	assert.Equal(t, ":7000", flagNetAddr)
	assert.Equal(t, ":8099", flagGRPCNetAddr)
	assert.Equal(t, "error", flagLogLevel)
	assert.Equal(t, 60, flagStoreInterval)
	assert.Equal(t, false, flagRestore)
	assert.Equal(t, "192.168.0.6/24", flagTrustedSubnet)
}

func TestParseEnvironment(t *testing.T) {
	// Устанавливаем переменные окружения
	os.Setenv("ADDRESS", ":8000")
	os.Setenv("GRPC_ADDRESS", ":8009")
	os.Setenv("SERVER_LOG_LEVEL", "debug")
	os.Setenv("STORE_INTERVAL", "200")
	os.Setenv("FILE_STORAGE_PATH", "./file/storage/path")
	os.Setenv("RESTORE", "true")
	os.Setenv("DATABASE_DSN", "env_dsn")
	os.Setenv("KEY", "env_key")
	os.Setenv("CRYPTO_KEY", "env_crypto_key")
	os.Setenv("CONFIG", "test_name_of_config_file")
	os.Setenv("TRUSTED_SUBNET", "192.168.0.12/24")
	defer func() {
		os.Unsetenv("ADDRESS")
		os.Unsetenv("GRPC_ADDRESS")
		os.Unsetenv("STORE_INTERVAL")
		os.Unsetenv("FILE_STORAGE_PATH")
		os.Unsetenv("RESTORE")
		os.Unsetenv("DATABASE_DSN")
		os.Unsetenv("KEY")
		os.Unsetenv("CRYPTO_KEY")
		os.Unsetenv("CONFIG")
		os.Unsetenv("TRUSTED_SUBNET")
	}()

	parseEnvironment()

	assert.Equal(t, ":8000", flagNetAddr)
	assert.Equal(t, ":8009", flagGRPCNetAddr)
	assert.Equal(t, "debug", flagLogLevel)
	assert.Equal(t, 200, flagStoreInterval)
	assert.Equal(t, "./file/storage/path", flagFileStoragePath)
	assert.Equal(t, true, flagRestore)
	assert.Equal(t, "env_dsn", flagDatabaseDsn)
	assert.Equal(t, "env_key", flagKey)
	assert.Equal(t, "env_crypto_key", flagCryptoKey)
	assert.Equal(t, "test_name_of_config_file", flagConfigFile)
	assert.Equal(t, "192.168.0.12/24", flagTrustedSubnet)
}

func TestParseConfigFile(t *testing.T) {
	testFlagNetAddr := "localhost:8082"
	testFlagLogLevel := "info"
	testFlagRestore := true
	testFlagStoreInterval := 1
	testFlagFileStoragePath := "test/file/path"
	testFlagDatabaseDsn := "test dsn"
	testFlagCryptoKey := "test crypto key"
	testFlagTrustedSubnet := "192.169.0.14/24"
	testFlagGRPCNetAddr := ":9999"

	createFile := func(name string) {
		data := fmt.Sprintf("{\"address\": \"%s\",\"log_level\": \"%s\",\"restore\": %t,\"store_interval\": \"%ds\",\"store_file\": \"%s\",\"database_dsn\": \"%s\",\"crypto_key\": \"%s\", \"trusted_subnet\": \"%s\", \"grpc_address\": \"%s\"}",
			testFlagNetAddr, testFlagLogLevel, testFlagRestore, testFlagStoreInterval, testFlagFileStoragePath,
			testFlagDatabaseDsn, testFlagCryptoKey, testFlagTrustedSubnet, testFlagGRPCNetAddr)
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
	assert.Equal(t, testFlagLogLevel, flagLogLevel)
	assert.Equal(t, testFlagRestore, flagRestore)
	assert.Equal(t, testFlagStoreInterval, flagStoreInterval)
	assert.Equal(t, testFlagFileStoragePath, flagFileStoragePath)
	assert.Equal(t, testFlagDatabaseDsn, flagDatabaseDsn)
	assert.Equal(t, testFlagCryptoKey, flagCryptoKey)
	assert.Equal(t, testFlagTrustedSubnet, flagTrustedSubnet)
	assert.Equal(t, testFlagGRPCNetAddr, flagGRPCNetAddr)

	err := os.Remove(nameFile)
	require.NoError(t, err)
}
