package compress

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecompress(t *testing.T) {
	// успешная распаковка данных
	{
		initialData := []byte("data for testing")
		// сжатие данных
		compressData, err := Compress(initialData)
		require.NoError(t, err)
		// распаковка данных
		decompressData, err := Decompress(compressData)
		require.NoError(t, err)
		assert.Equal(t, initialData, decompressData)
	}
	// тест с попыткой расжать несжатые данные
	{
		initialData := []byte("data for testing")
		// распаковка данных
		_, err := Decompress(initialData)
		require.Error(t, err)
	}
}
