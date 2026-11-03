// Copyright (c) 2026 The Notify Authors. All rights reserved.
// Use of this source code is governed by the MIT license that can be
// found in the LICENSE file.

//go:build linux && !(android && amd64)

package notify

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
	"time"
)

// Measure from immediately before creation through user delivery. Use -benchtime
// with a fixed iteration count to compare equal bursts (e.g. -benchtime=15000x).
// Cleanup and draining are excluded from throughput, but all delivered events
// contribute to the latency distribution. Missing events are reported explicitly.
func benchmarkCreation(b *testing.B, directory, churn bool) {
	root := b.TempDir()
	tr := newTree().(*nonrecursiveTree)
	defer tr.Close()
	expected := b.N
	if churn {
		expected *= 2
	}
	user := make(chan EventInfo, 3*b.N+buffer)
	mask := Create
	if churn {
		mask = All
	}
	if err := tr.Watch(filepath.Join(root, "..."), user, mask); err != nil {
		b.Fatal(err)
	}
	starts := make([]time.Time, expected)
	paths := make(map[string]int, b.N)
	names := make([]string, b.N)
	for j := range names {
		names[j] = filepath.Join(root, fmt.Sprint(j))
		paths[names[j]] = j
	}
	// The consumer records delivery timestamps independently of filesystem work.
	type arrival struct {
		path string
		at   time.Time
	}
	arrivals := make(chan []arrival, 1)
	stop := make(chan struct{})
	go func() {
		got := make([]arrival, 0, expected)
		for {
			select {
			case ei := <-user:
				if ei.Event() != Create {
					continue
				}
				got = append(got, arrival{ei.Path(), time.Now()})
				if len(got) == expected {
					arrivals <- got
					return
				}
			case <-stop:
				arrivals <- got
				return
			}
		}
	}()
	burstStart := time.Now()
	peak := runtime.NumGoroutine()
	sampleDone := make(chan struct{})
	samples := make(chan int, 1)
	go func() {
		ticker := time.NewTicker(250 * time.Microsecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if n := runtime.NumGoroutine(); n > peak {
					peak = n
				}
			case <-sampleDone:
				samples <- peak
				return
			}
		}
	}()
	b.ResetTimer()
	for j, path := range names {
		idx := j
		if churn {
			idx = 2 * j
		}
		starts[idx] = time.Now()
		var err error
		if directory {
			err = os.Mkdir(path, 0755)
		} else {
			var f *os.File
			f, err = os.Create(path)
			if err == nil {
				err = f.Close()
			}
		}
		if err != nil {
			b.Fatal(err)
		}
		if churn {
			if err = os.Remove(path); err != nil {
				b.Fatal(err)
			}
			starts[idx+1] = time.Now()
			if err = os.Mkdir(path, 0755); err != nil {
				b.Fatal(err)
			}
		}
	}
	b.StopTimer()
	burstEnd := time.Now()
	var got []arrival
	select {
	case got = <-arrivals:
	case <-time.After(3 * time.Second):
		close(stop)
		got = <-arrivals
	}
	if directory {
		waitTreeMaintenance(b, tr, root)
		b.ReportMetric(float64(time.Since(burstStart).Microseconds())/1000, "drain-ms")
	}
	// Installation is a separate measurement: user notification precedes the
	// recursive backend watch. Do not count fast delivery as installed coverage.
	if directory {
		backend := tr.w.(*inotify)
		installed := 0
		deadline := time.Now().Add(3 * time.Second)
		for {
			backend.RLock()
			if churn {
				unique := make(map[string]bool, b.N)
				for _, wd := range backend.m {
					if _, ok := paths[wd.path]; ok {
						unique[wd.path] = true
					}
				}
				installed = len(unique)
			} else {
				installed = len(backend.m) - 1
			}
			backend.RUnlock()
			if installed >= b.N || time.Now().After(deadline) {
				break
			}
			time.Sleep(250 * time.Microsecond)
		}
		now := time.Now()
		b.ReportMetric(float64(now.Sub(burstStart).Microseconds())/1000, "install-ms")
		b.ReportMetric(float64(now.Sub(burstEnd).Microseconds())/1000, "post-burst-ms")
		b.ReportMetric(float64(b.N-installed), "missing-watches")
		// Map membership alone can describe a stale descriptor in older versions.
		// Probe every final directory to measure usable coverage after the burst.
	drain:
		for {
			select {
			case <-user:
			default:
				break drain
			}
		}
		for _, path := range names {
			if err := os.WriteFile(filepath.Join(path, "notify-probe"), nil, 0644); err != nil {
				b.Fatal(err)
			}
		}
		probes := make(map[string]bool, b.N)
		timeout := time.After(3 * time.Second)
	collect:
		for len(probes) < b.N {
			select {
			case ei := <-user:
				if ei.Event() == Create && filepath.Base(ei.Path()) == "notify-probe" {
					probes[ei.Path()] = true
				}
			case <-timeout:
				break collect
			}
		}
		b.ReportMetric(float64(b.N-len(probes)), "missing-probes")

	}
	close(sampleDone)
	b.ReportMetric(float64(<-samples), "peak-goroutines")
	latencies := make([]int64, 0, len(got))
	seen := make(map[string]int, len(got))
	for _, a := range got {
		if idx, ok := paths[a.path]; ok {
			limit := 1
			if churn {
				limit = 2
				idx = 2*idx + seen[a.path]
			}
			if seen[a.path] >= limit {
				continue
			}
			seen[a.path]++
			latencies = append(latencies, a.at.Sub(starts[idx]).Nanoseconds())
		}
	}
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	if len(latencies) > 0 {
		b.ReportMetric(float64(latencies[(len(latencies)-1)/2])/1000, "p50-us")
		b.ReportMetric(float64(latencies[(len(latencies)-1)*95/100])/1000, "p95-us")
	}
	b.ReportMetric(float64(expected-len(latencies)), "missing")
}

func BenchmarkDirectoryCreates(b *testing.B) { benchmarkCreation(b, true, false) }
func BenchmarkFileBurst(b *testing.B)        { benchmarkCreation(b, false, false) }

func BenchmarkDirectoryChurn(b *testing.B) { benchmarkCreation(b, true, true) }
