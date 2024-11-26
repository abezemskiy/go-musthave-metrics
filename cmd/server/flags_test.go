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
	os.Args = []string{"cmd", "-a", ":9000", "-i", "120", "-f", "./metrics.json", "-r=false", "-d", "db_dsn", "-k", "secret"}
	defer func() { os.Args = originalArgs }()

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	result := parseFlags()

	assert.Equal(t, ":9000", flagNetAddr)
	assert.Equal(t, 120, flagStoreInterval)
	assert.Equal(t, "./metrics.json", flagFileStoragePath)
	assert.Equal(t, false, flagRestore)
	assert.Equal(t, "db_dsn", flagDatabaseDsn)
	assert.Equal(t, "secret", flagKey)
	assert.Equal(t, SAVEINDATABASE, result)
}

func TestParseFlagsPriority(t *testing.T) {
	// Устанавливаем переменные окружения
	os.Setenv("ADDRESS", ":8000")
	os.Setenv("STORE_INTERVAL", "200")
	os.Setenv("RESTORE", "true")
	defer func() {
		os.Unsetenv("ADDRESS")
		os.Unsetenv("STORE_INTERVAL")
		os.Unsetenv("RESTORE")
	}()

	// Создаём временный конфигурационный файл
	configFile := "./test_config.json"
	configContent := `{
        "address": ":7000",
        "restore": false,
        "store_interval": "60s"
    }`
	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)
	defer os.Remove(configFile)

	// Сохраняем оригинальные значения флагов
	originalArgs := os.Args
	os.Args = []string{"cmd", "-a", ":9000", "-i", "120", "-f", "./metrics.json", "-r=false", "-d", "db_dsn", "-k", "secret", "-c", "./test_config.json"}
	defer func() { os.Args = originalArgs }()

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	parseFlags()

	assert.Equal(t, ":7000", flagNetAddr)  // Флаг имеет приоритет
	assert.Equal(t, 60, flagStoreInterval) // Флаг имеет приоритет
	assert.Equal(t, false, flagRestore)    // Флаг имеет приоритет
}

func TestParseEnvironment(t *testing.T) {
	// Устанавливаем переменные окружения
	os.Setenv("ADDRESS", ":8000")
	os.Setenv("STORE_INTERVAL", "200")
	os.Setenv("RESTORE", "true")
	os.Setenv("FILE_STORAGE_PATH", "/tmp/metrics.json")
	os.Setenv("DATABASE_DSN", "env_dsn")
	os.Setenv("KEY", "env_key")
	defer func() {
		os.Unsetenv("ADDRESS")
		os.Unsetenv("STORE_INTERVAL")
		os.Unsetenv("RESTORE")
		os.Unsetenv("FILE_STORAGE_PATH")
		os.Unsetenv("DATABASE_DSN")
		os.Unsetenv("KEY")
	}()

	parseEnvironment()

	assert.Equal(t, ":8000", flagNetAddr)
	assert.Equal(t, 200, flagStoreInterval)
	assert.Equal(t, true, flagRestore)
	assert.Equal(t, "/tmp/metrics.json", flagFileStoragePath)
	assert.Equal(t, "env_dsn", flagDatabaseDsn)
	assert.Equal(t, "env_key", flagKey)
}

func TestParseConfigFile(t *testing.T) {
	testFlagNetAddr := "localhost:8082"
	testFlagRestore := true
	testFlagStoreInterval := 1
	testFlagFileStoragePath := "test/file/path"
	testFlagDatabaseDsn := "test dsn"
	testFlagCryptoKey := "test crypto key"

	createFile := func(name string) {
		data := fmt.Sprintf("{\"address\": \"%s\",\"restore\": %t,\"store_interval\": \"%ds\",\"store_file\": \"%s\",\"database_dsn\": \"%s\",\"crypto_key\": \"%s\"}",
			testFlagNetAddr, testFlagRestore, testFlagStoreInterval, testFlagFileStoragePath, testFlagDatabaseDsn, testFlagCryptoKey)
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
	assert.Equal(t, testFlagRestore, flagRestore)
	assert.Equal(t, testFlagStoreInterval, flagStoreInterval)
	assert.Equal(t, testFlagFileStoragePath, flagFileStoragePath)
	assert.Equal(t, testFlagDatabaseDsn, flagDatabaseDsn)
	assert.Equal(t, testFlagCryptoKey, flagCryptoKey)

	err := os.Remove(nameFile)
	require.NoError(t, err)
}
