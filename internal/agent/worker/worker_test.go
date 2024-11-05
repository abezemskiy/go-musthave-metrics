package worker

import (
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/storage"
)

func TestNewTask(t *testing.T) {
	adress := "/test/adress"
	action := "Post"
	metrics := storage.NewMetricsStats()
	metrics.CollectMetrics()
	pushFunction := func(string, string, *storage.MetricsStats, *resty.Client) error {
		return nil
	}
	wantTask := &Task{
		address:      adress,
		action:       action,
		metrics:      metrics,
		pushFunction: pushFunction,
	}
	getTask := NewTask(adress, action, metrics, pushFunction)
	assert.Equal(t, wantTask.address, getTask.address)
	assert.Equal(t, wantTask.action, getTask.action)
	assert.Equal(t, wantTask.metrics, getTask.metrics)
	assert.Equal(t, wantTask.pushFunction("", "", nil, nil), getTask.pushFunction("", "", nil, nil))
}
