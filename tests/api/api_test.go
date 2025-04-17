//go:build integration

package api

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var (
	addr     = flag.String("addr", "http://127.0.0.1:9130", "Blockbook base URL")
	coinName = flag.String("coin", "tron", "Coin fixtures to use (tron, eth, …)")
)

/* ---------- Test bootstrap ---------- */

func TestMain(m *testing.M) {
	flag.Parse()

	if err := ping(*addr); err != nil {
		fmt.Fprintf(os.Stderr, "Blockbook nedostupný na %s: %v\n", *addr, err)
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

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	var st struct{ Blockbook struct{ InSync bool } }
	return json.NewDecoder(resp.Body).Decode(&st)
}

/* ---------- Tests ---------- */

func TestStatus(t *testing.T) {
	resp, err := http.Get(*addr + "/api/status")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestTxDetail(t *testing.T) {
	txid := "0x0b6434f268778fdac9d63dac167fb65082b11750d06c13769f0f03aa021f6a91"
	want := fixture(t, filepath.Join("tx", txid+".json"))

	got := fetch(t, *addr+"/api/v2/tx/"+txid)
	assertJSONEq(t, want, got)
}

func TestBlockTxCount(t *testing.T) {
	var list []struct {
		Height uint32 `json:"height"`
		Txs    int    `json:"txs"`
	}
	require.NoError(t, json.Unmarshal(fixture(t, "block_count.json"), &list))

	for _, item := range list {
		url := fmt.Sprintf("%s/api/v2/block/%d", *addr, item.Height)
		body := fetch(t, url)

		var block struct {
			TxCount int `json:"txs"`
		}
		require.NoError(t, json.Unmarshal(body, &block))

		require.Equal(t, item.Txs, block.TxCount,
			"height %d", item.Height)
	}
}

func BenchmarkGetAddress(b *testing.B) {
	url := *addr + "/api/v2/address/TXncUDXYkRCmwhFikxYMutwAy93fbhPbbv?details=basic"
	client := &http.Client{Timeout: 5 * time.Second}

	// „zahřívací“ hit – naplní cache
	_, _ = client.Get(url)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err := client.Get(url)
		if err != nil || resp.StatusCode != 200 {
			b.Fatalf("err %v status %d", err, resp.StatusCode)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

/* ---------- Helpers ---------- */

func fixture(t *testing.T, rel string) []byte {
	t.Helper()
	p := filepath.Join("testdata", *coinName, rel)
	b, err := os.ReadFile(p)
	require.NoError(t, err, "missing fixture %s", p)
	return b
}

// returns response as byte[]
func fetch(t *testing.T, url string) []byte {
	t.Helper()
	r, err := http.Get(url)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.StatusCode)
	defer r.Body.Close()
	b, _ := io.ReadAll(r.Body)
	return b
}

func assertJSONEq(t *testing.T, want, got []byte) {
	t.Helper()
	var jw, jg interface{}
	require.NoError(t, json.Unmarshal(rewrite(want), &jw))
	require.NoError(t, json.Unmarshal(rewrite(got), &jg))
	require.Equal(t, jw, jg)
}

func rewrite(in []byte) []byte {
	in = bytes.ReplaceAll(in, []byte(`"IGNORE"`), []byte(`null`))
	return in
}
