//go:build integration

package api

import (
	"encoding/csv"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
)

/* ---------- Shared counters ---------- */

var (
	blkDelta, txDelta uint64
	blkTotal, txTotal uint64
	blkRequested      uint64
)

func noteProgress(txCnt int) {
	atomic.AddUint64(&blkDelta, 1)
	atomic.AddUint64(&txDelta, uint64(txCnt))
	atomic.AddUint64(&blkTotal, 1)
	atomic.AddUint64(&txTotal, uint64(txCnt))
}

func noteRequest() {
	atomic.AddUint64(&blkRequested, 1)
}

/* ---------- Sample data ---------- */

type sample struct {
	ts                time.Time
	memMiB            float64
	cpuPercent        float64
	iowaitPercent     float64
	readB, writeB     uint64
	newBlocks, newTxs uint64
	allBlocks, allTxs uint64
	requestedBlocks   uint64
}

/* ---------- Monitoring goroutine ---------- */

func startMonitor(period time.Duration, csvPath string) (stop func() []sample) {

	f, err := os.Create(csvPath)
	if err != nil {
		panic(fmt.Sprintf("cannot create CSV file %s: %v", csvPath, err))
	}

	w := csv.NewWriter(f)
	_ = w.Write([]string{
		"ts",
		"memMiB", "cpuP", "iowaitP",
		"readMiB", "writeMiB",
		"deltaBlocks", "deltaTxs",
		"allBlocks", "allTxs",
		"blkRequested",
	})

	var (
		mu  sync.Mutex
		out []sample
	)
	done := make(chan struct{})

	prevCPU, _ := cpu.Times(false)

	totalIO := func(m map[string]disk.IOCountersStat) (rd, wr uint64) {
		for _, v := range m {
			rd += v.ReadBytes
			wr += v.WriteBytes
		}
		return
	}

	go func() {
		ticker := time.NewTicker(period)
		defer func() {
			ticker.Stop()
			w.Flush()
			f.Close()
		}()

		for {
			select {
			case <-done:
				return

			case <-ticker.C:
				now := time.Now()

				/* ---------- CPU / iowait ---------- */
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

				/* ---------- disk I/O ---------- */
				curIO, _ := disk.IOCounters()
				rd, wr := totalIO(curIO)

				/* ---------- block / tx metrics ---------- */
				dBlk := atomic.SwapUint64(&blkDelta, 0)
				dTx := atomic.SwapUint64(&txDelta, 0)
				aBlk := atomic.LoadUint64(&blkTotal)
				aTx := atomic.LoadUint64(&txTotal)

				rBlk := atomic.LoadUint64(&blkRequested)
				atomic.SwapUint64(&blkRequested, 0)

				s := sample{
					ts:              now,
					memMiB:          usedMiB,
					cpuPercent:      cpuPercent,
					iowaitPercent:   iowaitPerc,
					readB:           rd,
					writeB:          wr,
					newBlocks:       dBlk,
					newTxs:          dTx,
					allBlocks:       aBlk,
					allTxs:          aTx,
					requestedBlocks: rBlk,
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
					fmt.Sprintf("%d", rBlk),
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

/* ---------- Print statistics ---------- */

func printStats(t testing.TB, s []sample, label string) {
	if len(s) == 0 {
		t.Logf("%s: no samples collected", label)
		return
	}

	var (
		peakMem, peakCPU, peakWait float64
		sumCPU, sumWait            float64
	)

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
		sumCPU += v.cpuPercent
		sumWait += v.iowaitPercent
	}

	avgCPU := sumCPU / float64(len(s))
	avgWait := sumWait / float64(len(s))

	readMB := float64(s[len(s)-1].readB-s[0].readB) / 1024 / 1024
	writeMB := float64(s[len(s)-1].writeB-s[0].writeB) / 1024 / 1024
	dur := s[len(s)-1].ts.Sub(s[0].ts).Seconds()

	t.Logf(
		"%s – runtime %.1fs  peakRAM %.1f MiB  peakCPU %.1f%%  avgCPU %.1f%%  peakIOwait %.1f%%  avgIOwait %.1f%%  read %.1f MiB  write %.1f MiB",
		label,
		dur,
		peakMem,
		peakCPU,
		avgCPU,
		peakWait,
		avgWait,
		readMB,
		writeMB,
	)
}
