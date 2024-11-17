package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseConfigFile(t *testing.T) {
	reportInterval = new(int)
	pollInterval = new(int)

	testFlagNetAddr := "localhost:8081"
	testReportInterval := 21
	testPollInterval := 3
	testFlagCryptoKey := "test crypto key"

	createFile := func(name string) {
		data := fmt.Sprintf("{\"address\": \"%s\",\"report_interval\": \"%ds\",\"poll_interval\": \"%ds\",\"crypto_key\": \"%s\"}",
			testFlagNetAddr, testReportInterval, testPollInterval, testFlagCryptoKey)
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

	err := os.Remove(nameFile)
	require.NoError(t, err)
}
