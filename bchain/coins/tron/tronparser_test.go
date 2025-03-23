//go:build unittest

package tron

import (
	"encoding/hex"
	"reflect"
	"testing"
)

func TestTronParser_GetAddrDescFromAddress(t *testing.T) {
	type args struct {
		address string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name:    "Base58 Tron Address",
			args:    args{address: "TJngGWiRMLgNFScEybQxLEKQMNdB4nR6Vx"},
			want:    "60bb513e91aa723a10a4020ae6fcce39bce7e240", // Hexadecimal format with prefix 41
			wantErr: false,
		},
		{
			name:    "Hex Tron Address as from JSON-RPC",
			args:    args{address: "0xef51c82ea6336ba1544c4a182a7368e9fbe28274"},
			want:    "ef51c82ea6336ba1544c4a182a7368e9fbe28274", // descriptor without prefix and checksum -> len = 20
			wantErr: false,
		},
		{
			name:    "Invalid Tron Address",
			args:    args{address: "invalidAddress"},
			want:    "",
			wantErr: true,
		},
	}
	parser := NewTronParser(1, false)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parser.GetAddrDescFromAddress(tt.args.address)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAddrDescFromAddress() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			h := hex.EncodeToString(got)
			if h != tt.want {
				t.Errorf("GetAddrDescFromAddress() = %v, want %v", h, tt.want)
			}
		})
	}
}

func TestTronParser_GetAddressesFromAddrDesc(t *testing.T) {
	type args struct {
		desc string
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			name:    "Desc to Base58 Tron Address",
			args:    args{desc: "f3f1c189594e2642e5d42d7669b4ec60a69802a9"},
			want:    []string{"TYD4pB7wGi1p8zK67rBTV3KdfEb9nvNDXh"},
			wantErr: false,
		},
		{
			name:    "Desc to Base58 Tron Address 2",
			args:    args{desc: "ef51c82ea6336ba1544c4a182a7368e9fbe28274"},
			want:    []string{"TXncUDXYkRCmwhFikxYMutwAy93fbhPbbv"},
			wantErr: false,
		},
		{
			name:    "Invalid Hex Address",
			args:    args{desc: "invalidHex"},
			want:    nil,
			wantErr: true,
		},
	}
	parser := NewTronParser(1, false)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := hex.DecodeString(tt.args.desc)
			if err != nil && !tt.wantErr {
				t.Errorf("GetAddressesFromAddrDesc() error = %v", err)
				return
			}

			got, _, err := parser.GetAddressesFromAddrDesc(b)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAddressesFromAddrDesc() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetAddressesFromAddrDesc() = %v, want %v", got, tt.want)
			}
		})
	}
}
