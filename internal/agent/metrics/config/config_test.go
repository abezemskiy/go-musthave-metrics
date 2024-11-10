package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

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
