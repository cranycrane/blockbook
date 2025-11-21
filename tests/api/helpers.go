//go:build integration

package api

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func fetchBlock(client *http.Client, url string) (time.Duration, int, error) {
	start := time.Now()
	resp, err := client.Get(url)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
	}

	var blk struct {
		TxCount int `json:"txCount"`
	}
	err = json.Unmarshal(body, &blk)
	return time.Since(start), blk.TxCount, err
}

func report(t testing.TB, name string, times []time.Duration, txPerCall []int, totalDuration time.Duration) {
	var filtered []time.Duration
	var txSum int
	var sumDur time.Duration

	for i, d := range times {
		if d > 0 {
			filtered = append(filtered, d)
			txSum += txPerCall[i]
			sumDur += d
		}
	}

	if len(filtered) == 0 {
		t.Logf("%s: no samples", name)
		return
	}

	sort.Slice(filtered, func(i, j int) bool { return filtered[i] < filtered[j] })

	avg := sumDur / time.Duration(len(filtered))
	p95 := filtered[int(0.95*float64(len(filtered)))]

	tps := float64(txSum) / totalDuration.Seconds()

	t.Logf("%s: n=%d total=%v avg=%v p95=%v → %.1f tx/s",
		name, len(filtered), totalDuration, avg, p95, tps)
}

func getBestBlockHeight(t *testing.T) uint32 {
	resp, err := http.Get(*addr + "/api/status")
	require.NoError(t, err)
	defer resp.Body.Close()

	var s struct {
		Blockbook struct {
			BestHeight uint32 `json:"bestHeight"`
		}
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&s))
	return s.Blockbook.BestHeight
}

func randomHeights(t *testing.T, min, max, n int) []int {
	if max-min < n {
		t.Fatalf("range too small")
	}
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	set := make(map[int]struct{})
	for len(set) < n {
		set[rng.Intn(max-min)+min] = struct{}{}
	}
	var out []int
	for h := range set {
		out = append(out, h)
	}
	sort.Ints(out)
	return out
}
