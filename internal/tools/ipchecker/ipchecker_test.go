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
			name:    "in trusted, address with port",
			subNet:  "192.168.0.0/24",
			realIP:  "192.168.0.235:47886",
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
		{
			name:    "wrong real ip",
			subNet:  "192.168.14.0/16",
			realIP:  "invalid:address:with:too:many:colons",
			wantRes: false,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := InTrustedSubNet(tt.subNet, tt.realIP)
			if tt.wantErr {
				if err == nil {
					assert.Equal(t, false, res)
				}
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.wantRes, res)
		})
	}
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name        string
		ipAddress   string
		wantAddress string
		wantErr     bool
	}{
		{
			name:        "successful test#1",
			ipAddress:   "192.168.0.0:2345",
			wantAddress: "192.168.0.0",
			wantErr:     false,
		},
		{
			name:        "successful test#2",
			ipAddress:   "192.168.1.100",
			wantAddress: "192.168.1.100",
			wantErr:     false,
		},
		{
			name:        "successful test#3",
			ipAddress:   "[::1]:8080",
			wantAddress: "::1",
			wantErr:     false,
		},
		{
			name:        "successful test#4",
			ipAddress:   "localhost:9090",
			wantAddress: "localhost",
			wantErr:     false,
		},
		{
			name:        "wrong ip address test#5",
			ipAddress:   "wrong_address",
			wantAddress: "wrong_address",
			wantErr:     false,
		},
		{
			name:        "wrong ip address test#6",
			ipAddress:   "invalid:address:with:too:many:colons",
			wantAddress: "",
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getAddr, err := getClientIP(tt.ipAddress)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantAddress, getAddr)
			}
		})
	}
}
