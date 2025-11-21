//go:build integration

// NOTE: Blockbook must have the `notxcache` set to TRUE so the benchmarks are correct

package api

import (
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"
)

const workers = 15

func TestBlockLatencyParallelWorkers(t *testing.T) {
	file := fmt.Sprintf("systemParallel-%s.csv", *coinName)
	stop := startMonitor(10*time.Millisecond, file)
	defer func() { printStats(t, stop(), "parallel") }()

	heights := parseBlockRange(t, *blocksArg)
	numBlocks := len(heights)

	const workers = 15

	type result struct {
		Cold time.Duration
		Tx   int
		Err  error
	}

	results := make([]result, numBlocks)

	client := &http.Client{Timeout: 15 * time.Second}
	wg := sync.WaitGroup{}
	wg.Add(workers)

	jobs := make(chan int, numBlocks)

	start := time.Now()

	for w := 0; w < workers; w++ {
		go func() {
			defer wg.Done()
			for i := range jobs {
				h := heights[i]
				url := fmt.Sprintf("%s/api/v2/block/%d?details=basic", *addr, h)

				noteRequest()
				cold, tx, err := fetchBlock(client, url)
				if err == nil {
					noteProgress(tx)
				}
				results[i] = result{Cold: cold, Tx: tx, Err: err}
			}
		}()
	}

	for i := 0; i < numBlocks; i++ {
		jobs <- i
	}
	close(jobs)

	wg.Wait()
	total := time.Since(start)

	var coldTimes []time.Duration
	var txs []int

	for _, r := range results {
		if r.Err == nil {
			coldTimes = append(coldTimes, r.Cold)
			txs = append(txs, r.Tx)
		} else {
			t.Logf("block failed: %v", r.Err)
		}
	}

	report(t, "cold", coldTimes, txs, total)
}
