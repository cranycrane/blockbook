//go:build integration

package api

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic" //  ← přidat
	"testing"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk" //  ← přidat
	"github.com/shirou/gopsutil/v3/mem"  //  ← přidat
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

func TestBlockLatencyBenchmarkSequential(t *testing.T) {
	stopMon := startMonitor(10*time.Millisecond, "system.csv")
	defer func() {
		printStats(t, stopMon(), "seq")
	}()

	best := getBestBlockHeight(t)
	heights := randomHeights(t, 40_000_000, int(best), 10)

	type result struct {
		Height     int
		TxCount    int
		Cold, Warm time.Duration
	}
	var results []result

	client := &http.Client{Timeout: 10 * time.Second}

	for _, h := range heights {
		url := fmt.Sprintf("%s/api/v2/block/%d?details=basic", *addr, h)

		fetch := func() (time.Duration, int) {
			start := time.Now()
			resp, err := client.Get(url)
			require.NoError(t, err)
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("block %d – expected 200, got %d\nresponse: %s", h, resp.StatusCode, body)
			}

			var blk struct {
				TxCount int `json:"txCount"`
			}
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&blk))
			return time.Since(start), blk.TxCount
		}

		coldTime, txs := fetch()
		noteProgress(txs)

		warmTime, _ := fetch()

		results = append(results, result{
			Height:  h,
			TxCount: txs,
			Cold:    coldTime,
			Warm:    warmTime,
		})
	}

	var coldTimes, warmTimes []time.Duration
	for _, r := range results {
		t.Logf("block %d – %d tx – cold: %v  warm: %v", r.Height, r.TxCount, r.Cold, r.Warm)
		coldTimes = append(coldTimes, r.Cold)
		warmTimes = append(warmTimes, r.Warm)
	}

	report := func(name string, times []time.Duration, totalTxs int) {
		sort.Slice(times, func(i, j int) bool { return times[i] < times[j] })
		sum := time.Duration(0)
		for _, d := range times {
			sum += d
		}
		avg := sum / time.Duration(len(times))
		p95 := times[int(0.95*float64(len(times)))]

		txsPerSec := float64(totalTxs) / avg.Seconds()
		t.Logf("%s: n=%d  avg=%v  p95=%v  → throughput ≈ %.1f tx/s",
			name, len(times), avg, p95, txsPerSec)
	}

	totalColdTx := 0
	totalWarmTx := 0
	for _, r := range results {
		totalColdTx += r.TxCount
		totalWarmTx += r.TxCount
	}
	report("cold", coldTimes, totalColdTx)
	report("warm", warmTimes, totalWarmTx)
}

func TestBlockLatencyParallelWorkers(t *testing.T) {
	stopMon := startMonitor(50*time.Millisecond, "system.csv")
	defer func() {
		printStats(t, stopMon(), "seq")
	}()

	const (
		numWorkers = 5
		numBlocks  = 10
	)

	best := getBestBlockHeight(t)
	heights := randomHeights(t, 0, int(best), numBlocks)

	type result struct {
		Height     int
		TxCount    int
		Cold, Warm time.Duration
		Err        error
	}

	results := make([]result, numBlocks)
	client := &http.Client{Timeout: 15 * time.Second}
	jobChan := make(chan int, numBlocks)
	wg := sync.WaitGroup{}
	wg.Add(numWorkers)

	// Start worker pool
	for w := 0; w < numWorkers; w++ {
		go func(workerID int) {
			defer wg.Done()
			for i := range jobChan {
				h := heights[i]
				url := fmt.Sprintf("%s/api/v2/block/%d?details=basic", *addr, h)

				fetch := func() (time.Duration, int, error) {
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
					if err := json.Unmarshal(body, &blk); err != nil {
						return 0, 0, err
					}
					return time.Since(start), blk.TxCount, nil
				}

				cold, txs, err := fetch()
				if err != nil {
					results[i] = result{Height: h, Err: err}
					continue
				}
				noteProgress(txs)

				warm, _, err := fetch()
				if err != nil {
					results[i] = result{Height: h, Err: err}
					continue
				}
				results[i] = result{Height: h, TxCount: txs, Cold: cold, Warm: warm}
			}
		}(w)
	}

	// Fill the queue
	for i := 0; i < numBlocks; i++ {
		jobChan <- i
	}
	close(jobChan)
	wg.Wait()

	// Process results
	var coldTimes, warmTimes []time.Duration
	totalTxCold, totalTxWarm := 0, 0

	for _, r := range results {
		if r.Err != nil {
			t.Errorf("block %d failed: %v", r.Height, r.Err)
			continue
		}
		t.Logf("block %d – %d tx – cold: %v  warm: %v", r.Height, r.TxCount, r.Cold, r.Warm)
		coldTimes = append(coldTimes, r.Cold)
		warmTimes = append(warmTimes, r.Warm)
		totalTxCold += r.TxCount
		totalTxWarm += r.TxCount
	}

	report := func(name string, times []time.Duration, totalTxs int) {
		if len(times) == 0 {
			t.Logf("%s: no successful results", name)
			return
		}
		sort.Slice(times, func(i, j int) bool { return times[i] < times[j] })
		sum := time.Duration(0)
		for _, d := range times {
			sum += d
		}
		avg := sum / time.Duration(len(times))
		p95 := times[int(0.95*float64(len(times)))]
		txsPerSec := float64(totalTxs) / avg.Seconds()
		t.Logf("%s: n=%d  avg=%v  p95=%v  → throughput ≈ %.1f tx/s",
			name, len(times), avg, p95, txsPerSec)
	}

	report("cold (workers)", coldTimes, totalTxCold)
	report("warm (workers)", warmTimes, totalTxWarm)
}

func BenchmarkGetAddress(b *testing.B) {
	url := *addr + "/api/v2/address/TXncUDXYkRCmwhFikxYMutwAy93fbhPbbv?details=basic"
	client := &http.Client{Timeout: 5 * time.Second}

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

/* ---------- system‑wide monitor ---------- */

var (
	blkDelta, txDelta uint64
	blkTotal, txTotal uint64
)

func noteProgress(txCnt int) {
	atomic.AddUint64(&blkDelta, 1)
	atomic.AddUint64(&txDelta, uint64(txCnt))
	atomic.AddUint64(&blkTotal, 1)
	atomic.AddUint64(&txTotal, uint64(txCnt))
}

type sample struct {
	ts                time.Time
	memMiB            float64
	cpuPercent        float64
	iowaitPercent     float64
	readB, writeB     uint64
	newBlocks, newTxs uint64
	allBlocks, allTxs uint64
}

func startMonitor(period time.Duration, csvPath string) (stop func() []sample) {
	f, _ := os.Create(csvPath)
	w := csv.NewWriter(f)
	_ = w.Write([]string{
		"ts",
		"memMiB", "cpuP", "iowaitP",
		"readMiB", "writeMiB",
		"deltaBlocks", "deltaTxs",
		"allBlocks", "allTxs",
	})

	var (
		mu  sync.Mutex
		out []sample
	)
	done := make(chan struct{})

	/* ----- init kumulativních čítačů ----- */
	prevCPU, _ := cpu.Times(false)

	// helper: sečte všechny disky
	totalIO := func(m map[string]disk.IOCountersStat) (rd, wr uint64) {
		for _, v := range m {
			rd += v.ReadBytes
			wr += v.WriteBytes
		}
		return
	}

	go func() {
		tick := time.NewTicker(period)
		defer func() {
			tick.Stop()
			w.Flush()
			f.Close()
		}()

		for {
			select {
			case <-done:
				return
			case <-tick.C:
				now := time.Now()

				/* ---------- CPU/iowait ---------- */
				curCPU, _ := cpu.Times(false)
				deltaTotal := curCPU[0].Total() - prevCPU[0].Total()
				if deltaTotal == 0 {
					continue
				}
				deltaIdle := curCPU[0].Idle - prevCPU[0].Idle
				deltaIow := curCPU[0].Iowait - prevCPU[0].Iowait
				cpuPercent := 100 * (deltaTotal - deltaIdle - deltaIow) / deltaTotal
				iowaitPerc := 100 * deltaIow / deltaTotal
				prevCPU = curCPU

				/* ---------- RAM ---------- */
				vm, _ := mem.VirtualMemory()
				usedMiB := float64(vm.Total-vm.Available) / 1048576.0

				/* ---------- disc I/O  ---------- */
				curIO, _ := disk.IOCounters()
				rd, wr := totalIO(curIO)

				/* ---------- bloky / tx ---------- */
				dBlk := atomic.SwapUint64(&blkDelta, 0)
				dTx := atomic.SwapUint64(&txDelta, 0)
				aBlk := atomic.LoadUint64(&blkTotal)
				aTx := atomic.LoadUint64(&txTotal)

				s := sample{
					ts:            now,
					memMiB:        usedMiB,
					cpuPercent:    cpuPercent,
					iowaitPercent: iowaitPerc,
					readB:         rd, writeB: wr,
					newBlocks: dBlk, newTxs: dTx,
					allBlocks: aBlk, allTxs: aTx,
				}
				mu.Lock()
				out = append(out, s)
				mu.Unlock()

				_ = w.Write([]string{
					s.ts.Format(time.RFC3339Nano),
					fmt.Sprintf("%.1f", s.memMiB),
					fmt.Sprintf("%.2f", s.cpuPercent),
					fmt.Sprintf("%.2f", s.iowaitPercent),
					fmt.Sprintf("%.1f", float64(rd)/1048576),
					fmt.Sprintf("%.1f", float64(wr)/1048576),
					fmt.Sprintf("%d", dBlk),
					fmt.Sprintf("%d", dTx),
					fmt.Sprintf("%d", aBlk),
					fmt.Sprintf("%d", aTx),
				})
			}
		}
	}()

	return func() []sample {
		close(done)
		mu.Lock()
		defer mu.Unlock()
		return append([]sample(nil), out...)
	}
}

func printStats(t *testing.T, s []sample, label string) {
	if len(s) == 0 {
		t.Logf("%s: no samples", label)
		return
	}
	var peakMem, peakCPU, peakWait float64
	for _, v := range s {
		if v.memMiB > peakMem {
			peakMem = v.memMiB
		}
		if v.cpuPercent > peakCPU {
			peakCPU = v.cpuPercent
		}
		if v.iowaitPercent > peakWait {
			peakWait = v.iowaitPercent
		}
	}
	readMB := float64(s[len(s)-1].readB-s[0].readB) / 1048576
	writeMB := float64(s[len(s)-1].writeB-s[0].writeB) / 1048576
	dur := s[len(s)-1].ts.Sub(s[0].ts).Seconds()

	t.Logf("%s – runtime %.1fs  peakRAM %.1f MiB  peakCPU %.1f%%  peakIOwait %.1f%%  read %.1f MiB  write %.1f MiB",
		label, dur, peakMem, peakCPU, peakWait, readMB, writeMB)
}

/* ---------- Helpers ---------- */

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
		t.Fatalf("rozsah příliš malý: %d–%d", min, max)
	}
	src := rand.NewSource(time.Now().UnixNano())
	rng := rand.New(src)

	set := make(map[int]struct{})
	for len(set) < n {
		h := rng.Intn(max-min) + min
		set[h] = struct{}{}
	}
	var heights []int
	for h := range set {
		heights = append(heights, h)
	}
	sort.Ints(heights)
	return heights
}

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
