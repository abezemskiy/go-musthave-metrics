package config

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/storage"
)

func TestSetPollInterval(t *testing.T) {
	assert.Equal(t, time.Duration(2), pollInterval)
	SetPollInterval(10)
	assert.Equal(t, time.Duration(10), pollInterval)
}

func TestGetPollInterval(t *testing.T) {
	SetPollInterval(15)
	assert.Equal(t, time.Duration(15), GetPollInterval())
}

func TestSetReportInterval(t *testing.T) {
	assert.Equal(t, time.Duration(10), reportInterval)
	SetReportInterval(20)
	assert.Equal(t, time.Duration(20), reportInterval)
}

func TestGetReportInterval(t *testing.T) {
	SetReportInterval(30)
	assert.Equal(t, time.Duration(30), GetReportInterval())
}

func TestSetContextTimeout(t *testing.T) {
	assert.Equal(t, time.Duration(500*time.Millisecond), contextTimeout)
	SetContextTimeout(700 * time.Millisecond)
	assert.Equal(t, time.Duration(700*time.Millisecond), contextTimeout)
}

func TestGetContextTimeout(t *testing.T) {
	SetContextTimeout(800 * time.Millisecond)
	assert.Equal(t, time.Duration(800*time.Millisecond), GetContextTimeout())
}

func TestSyncCollectMetrics(t *testing.T) {
	metrics := &storage.MetricsStats{}
	SyncCollectMetrics(metrics)
	assert.NotEqual(t, storage.MetricsStats{}, metrics)
}

func TestSetCryptoKey(t *testing.T) {
	cryptoKey = ""
	newPath := "new/path/for/private/key.pem"
	SetCryptoKey(newPath)
	assert.Equal(t, newPath, cryptoKey)
}

func TestGetCryptoKey(t *testing.T) {
	cryptoKey = "new/path/for/private/key.pem"
	assert.Equal(t, cryptoKey, GetCryptoKey())
}

func TestParseConfigFile(t *testing.T) {
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

	configs, err := ParseConfigFile(nameFile)
	require.NoError(t, err)

	assert.Equal(t, testFlagNetAddr, configs.Address)
	assert.Equal(t, testReportInterval, int(configs.ReportInterval.Seconds()))
	assert.Equal(t, testPollInterval, int(configs.PollInterval.Seconds()))
	assert.Equal(t, testFlagCryptoKey, configs.CryptoKey)

	err = os.Remove(nameFile)
	require.NoError(t, err)
}
