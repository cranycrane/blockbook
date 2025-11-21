//go:build integration

package api

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	addr      = flag.String("addr", "", "Blockbook base URL")
	coinName  = flag.String("coin", "", "Coin fixtures (tron, eth, …)")
	blocksArg = flag.String("blocks", "", "Block range: e.g. 65000000-65000049")
)

func TestMain(m *testing.M) {
	flag.Parse()

	if *addr == "" {
		fmt.Fprintln(os.Stderr, "-addr is required")
		os.Exit(2)
	}
	if *coinName == "" {
		fmt.Fprintln(os.Stderr, "-coin is required")
		os.Exit(2)
	}

	if err := ping(*addr); err != nil {
		fmt.Fprintf(os.Stderr, "Blockbook not available at %s: %v\n", *addr, err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func ping(base string) error {
	resp, err := http.Get(base + "/api/status")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return nil
}

func fixture(t *testing.T, rel string) []byte {
	t.Helper()
	path := filepath.Join("testdata", *coinName, rel)
	b, err := os.ReadFile(path)
	require.NoError(t, err, "missing fixture %s", path)
	return b
}

func fetch(t *testing.T, url string) []byte {
	t.Helper()
	r, err := http.Get(url)
	require.NoError(t, err)
	require.Equal(t, 200, r.StatusCode)
	defer r.Body.Close()
	b, _ := io.ReadAll(r.Body)
	return b
}

func assertJSONEq(t *testing.T, wantJSON, gotJSON []byte) {
	t.Helper()
	var want, got interface{}
	require.NoError(t, json.Unmarshal(wantJSON, &want))
	require.NoError(t, json.Unmarshal(gotJSON, &got))
	filterByIgnore(&want, &got)
	require.Equal(t, want, got)
}

func filterByIgnore(want, got *interface{}) {
	switch w := (*want).(type) {
	case map[string]interface{}:
		g := (*got).(map[string]interface{})
		for k, v := range w {
			if str, ok := v.(string); ok && str == "IGNORE" {
				delete(w, k)
				delete(g, k)
				continue
			}
			wc, gc := v, g[k]
			filterByIgnore(&wc, &gc)
			w[k] = wc
			g[k] = gc
		}
	case []interface{}:
		g := (*got).([]interface{})
		for i := range w {
			if i < len(g) {
				filterByIgnore(&w[i], &g[i])
			}
		}
	}
}
