package saver

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/repositories/mocks"
)

func TestSetStoreInterval(t *testing.T) {
	storeInterval = 12
	newStoreInterval := 23
	SetStoreInterval(time.Duration(newStoreInterval))
	assert.Equal(t, time.Duration(newStoreInterval), storeInterval)
}

func TestGetStoreInterval(t *testing.T) {
	storeInterval = 46
	assert.Equal(t, storeInterval, GetStoreInterval())
}

func TestSetFilestoragePath(t *testing.T) {
	fileStoragePath = "/init/path"
	newStoragePath := "/new/storage/path"
	SetFilestoragePath(newStoragePath)
	assert.Equal(t, newStoragePath, fileStoragePath)
}

func TestGetFilestoragePath(t *testing.T) {
	fileStoragePath = "/secong/storage/path"
	assert.Equal(t, fileStoragePath, GetFilestoragePath())
}

func TestSetRestore(t *testing.T) {
	restore = true
	newRestoreValue := false
	SetRestore(newRestoreValue)
	assert.Equal(t, newRestoreValue, restore)
}

func TestGetRestore(t *testing.T) {
	restore = true
	assert.Equal(t, true, restore)
}

func TestNewWriter(t *testing.T) {
	_, err := NewWriter("")
	require.Error(t, err)
}

func TestWriteMetrics(t *testing.T) {
	createFile := func(name string) {
		_, err := os.Create(name)
		require.NoError(t, err)
	}
	deleteFile := func(name string) {
		err := os.Remove(name)
		require.NoError(t, err)
	}

	// Мокирую хранилище метрик
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	m := mocks.NewMockMetricsReader(ctrl)

	{
		testFile := "test_file"
		createFile(testFile)
		defer deleteFile(testFile)

		stor, err := NewWriter(testFile)
		require.NoError(t, err)

		// ошибка получения метрики
		m.EXPECT().GetAllMetricsSlice(gomock.Any()).Return(nil, fmt.Errorf("error"))
		err = stor.WriteMetrics(m)
		require.Error(t, err)
	}
}
