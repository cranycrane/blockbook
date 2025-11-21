//go:build integration

package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStatus(t *testing.T) {
	resp, err := http.Get(*addr + "/api/status")
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)
}

func TestTxDetail(t *testing.T) {
	dir := filepath.Join("testdata", *coinName, "tx")
	files, err := os.ReadDir(dir)
	require.NoError(t, err)

	for _, f := range files {
		if f.IsDir() || filepath.Ext(f.Name()) != ".json" {
			continue
		}
		txid := strings.TrimSuffix(f.Name(), ".json")
		t.Run(txid, func(t *testing.T) {
			want := fixture(t, filepath.Join("tx", f.Name()))
			got := fetch(t, fmt.Sprintf("%s/api/v2/tx/%s", *addr, txid))
			assertJSONEq(t, want, got)
		})
	}
}

func TestBlockDetail(t *testing.T) {
	dir := filepath.Join("testdata", *coinName, "block")
	files, err := os.ReadDir(dir)
	require.NoError(t, err)

	for _, f := range files {
		if f.IsDir() || filepath.Ext(f.Name()) != ".json" {
			continue
		}
		hash := strings.TrimSuffix(f.Name(), ".json")
		t.Run(hash, func(t *testing.T) {
			want := fixture(t, filepath.Join("block", f.Name()))
			got := fetch(t, fmt.Sprintf("%s/api/v2/block/%s", *addr, hash))
			assertJSONEq(t, want, got)
		})
	}
}
