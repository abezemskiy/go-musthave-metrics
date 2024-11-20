package saver

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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
