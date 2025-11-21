//go:build integration

package api

import (
	"strconv"
	"strings"
	"testing"
)

func parseBlockRange(t testing.TB, arg string) []int {
	if arg == "" {
		t.Fatalf("missing required -blocks parameter (example: -blocks=65000000-65000049)")
	}

	parts := strings.Split(arg, "-")
	if len(parts) != 2 {
		t.Fatalf("invalid block range %q – expected format start-end", arg)
	}

	start, err := strconv.Atoi(parts[0])
	if err != nil {
		t.Fatalf("invalid start height %q: %v", parts[0], err)
	}

	end, err := strconv.Atoi(parts[1])
	if err != nil {
		t.Fatalf("invalid end height %q: %v", parts[1], err)
	}

	if end < start {
		t.Fatalf("invalid range %d-%d: end < start", start, end)
	}

	n := end - start + 1
	heights := make([]int, n)

	for i := 0; i < n; i++ {
		heights[i] = start + i
	}

	return heights
}
