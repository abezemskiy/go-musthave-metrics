package ipgetter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGet(t *testing.T) {
	res, err := Get()
	require.NoError(t, err)

	assert.NotEqual(t, "", res)
}
