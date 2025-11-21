//go:build integration

// NOTE: Blockbook must have the `notxcache` set to TRUE so the benchmarks are correct

package api

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestBlockLatencyBenchmarkSequential(t *testing.T) {
	file := fmt.Sprintf("systemSeq-%s.csv", *coinName)
	stop := startMonitor(10*time.Millisecond, file)
	defer func() { printStats(t, stop(), "seq") }()

	// parse -blocks=65000000-65000049
	heights := parseBlockRange(t, *blocksArg)

	client := &http.Client{Timeout: 10 * time.Second}

	var coldTimes []time.Duration
	var coldTxs []int
	var total time.Duration

	start := time.Now()

	for _, h := range heights {
		url := fmt.Sprintf("%s/api/v2/block/%d", *addr, h)

		noteRequest()
		cold, txs, err := fetchBlock(client, url)
		if err == nil {
			noteProgress(txs)
		}

		coldTimes = append(coldTimes, cold)
		coldTxs = append(coldTxs, txs)
		total += cold
	}

	report(t, "cold", coldTimes, coldTxs, time.Since(start))
}
