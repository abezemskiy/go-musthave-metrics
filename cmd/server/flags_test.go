package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
