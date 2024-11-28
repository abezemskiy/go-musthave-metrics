package ipchecker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInTrustedSubNet(t *testing.T) {
	tests := []struct {
		name    string
		subNet  string
		realIP  string
		wantRes bool
		wantErr bool
	}{
		{
			name:    "in trusted",
			subNet:  "192.168.0.0/24",
			realIP:  "192.168.0.235",
			wantRes: true,
			wantErr: false,
		},
		{
			name:    "not in trusted",
			subNet:  "192.168.1.0/24",
			realIP:  "192.168.0.235",
			wantRes: false,
			wantErr: false,
		}, {
			name:    "wrong subNet",
			subNet:  "wrong.sub.net",
			realIP:  "192.168.0.235",
			wantRes: false,
			wantErr: true,
		},
		{
			name:    "empty subNet",
			subNet:  "",
			realIP:  "192.168.0.235",
			wantRes: false,
			wantErr: true,
		},
		{
			name:    "wrong real ip",
			subNet:  "192.168.14.0/16",
			realIP:  "wrong.real.ip",
			wantRes: false,
			wantErr: true,
		},
		{
			name:    "empty real ip",
			subNet:  "192.168.14.0/16",
			realIP:  "",
			wantRes: false,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := InTrustedSubNet(tt.subNet, tt.realIP)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.wantRes, res)
		})
	}
}
