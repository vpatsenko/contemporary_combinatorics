package main

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

func getRSSMB() float64 {
	var rusage syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &rusage); err != nil {
		return 0
	}

	rss := float64(rusage.Maxrss)

	// ru_maxrss units:
	// - macOS (darwin): bytes
	// - Linux: kilobytes
	// - BSDs: often kilobytes (varies), but Linux rule works for most non-darwin here.
	if runtime.GOOS == "darwin" {
		return rss / (1024 * 1024) // bytes -> MB
	}
	return rss / 1024 // KB -> MB
}


type PeakMemoryTracker struct {
	peakRSS  atomic.Value
	stopChan chan struct{}
	wg       sync.WaitGroup
	interval time.Duration
}

func NewPeakMemoryTracker(interval time.Duration) *PeakMemoryTracker {
	t := &PeakMemoryTracker{
		stopChan: make(chan struct{}),
		interval: interval,
	}
	t.peakRSS.Store(float64(0))
	return t
}

func (t *PeakMemoryTracker) Start() {
	t.peakRSS.Store(getRSSMB())
	t.wg.Add(1)

	go func() {
		defer t.wg.Done()
		ticker := time.NewTicker(t.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				current := getRSSMB()
				for {
					old := t.peakRSS.Load().(float64)
					if current <= old {
						break
					}
					if t.peakRSS.CompareAndSwap(old, current) {
						break
					}
				}
			case <-t.stopChan:
				return
			}
		}
	}()
}

func (t *PeakMemoryTracker) Stop() float64 {
	close(t.stopChan)
	t.wg.Wait()
	return t.peakRSS.Load().(float64)
}

// memoryIntensiveTask allocates ~sizeMB and TOUCHES EACH PAGE so RSS reflects committed memory.
// This matches the "touch per page" fix used in the Python benchmark.
func memoryIntensiveTask(sizeMB int) int64 {
	numBytes := sizeMB * 1024 * 1024
	data := make([]byte, numBytes)

	// Touch each OS page to force physical commitment and make RSS meaningful.
	page := os.Getpagesize()
	for i := 0; i < len(data); i += page {
		data[i] = byte((int(data[i]) + 1) & 0xFF)
	}

	// Keep it around briefly so the peak sampler sees it.
	time.Sleep(200 * time.Millisecond)

	// Use a few bytes so the compiler can't "prove" the buffer unused.
	var total int64
	total += int64(data[0])
	total += int64(data[len(data)/2])
	total += int64(data[len(data)-1])

	runtime.KeepAlive(data)
	return total
}

func runSingleThreaded(numTasks, sizeMB int) {
	for i := 0; i < numTasks; i++ {
		memoryIntensiveTask(sizeMB)
		runtime.GC()
	}
}

func runMultiThreaded(numTasks, sizeMB int) {
	var wg sync.WaitGroup
	wg.Add(numTasks)

	for i := 0; i < numTasks; i++ {
		go func() {
			defer wg.Done()
			memoryIntensiveTask(sizeMB)
		}()
	}
	wg.Wait()
}

func measureMemory(name string, fn func(int, int), numTasks, sizeMB int) float64 {
	runtime.GC()
	time.Sleep(50 * time.Millisecond)

	rssBefore := getRSSMB()

	tracker := NewPeakMemoryTracker(5 * time.Millisecond)
	tracker.Start()

	start := time.Now()
	fn(numTasks, sizeMB)
	elapsed := time.Since(start)

	peakRSS := tracker.Stop()
	rssAfter := getRSSMB()

	fmt.Printf("  Time: %.4f seconds\n", elapsed.Seconds())
	fmt.Printf("  RSS before: %.2f MB\n", rssBefore)
	fmt.Printf("  RSS peak: %.2f MB\n", peakRSS)
	fmt.Printf("  RSS after: %.2f MB\n", rssAfter)
	fmt.Printf("  RSS delta (peak - before): %.2f MB\n", peakRSS-rssBefore)

	return peakRSS
}

func main() {
	fmt.Printf("Go Version: %s\n", runtime.Version())
	fmt.Printf("GOMAXPROCS: %d\n", runtime.GOMAXPROCS(0))
	fmt.Printf("NumCPU: %d\n", runtime.NumCPU())
	fmt.Printf("PID: %d\n", os.Getpid())

	fmt.Println("\nNote: Go has no GIL - goroutines share memory and can run in parallel")

	fmt.Println("\n============================================================")
	fmt.Println("MEMORY BENCHMARK (RSS-based)")
	fmt.Println("============================================================")

	numTasks := 4
	sizeMB := 50

	fmt.Printf("\nConfiguration:\n")
	fmt.Printf("  Number of tasks: %d\n", numTasks)
	fmt.Printf("  Memory per task: ~%d MB\n", sizeMB)
	fmt.Printf("  Expected peak (sequential): ~%d MB\n", sizeMB)
	fmt.Printf("  Expected peak (parallel): ~%d MB\n", numTasks*sizeMB)

	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	baselineRSS := getRSSMB()
	fmt.Printf("\nBaseline RSS: %.2f MB\n", baselineRSS)

	fmt.Println("\n------------------------------------------------------------")
	fmt.Println("SINGLE-THREADED (Sequential)")
	fmt.Println("------------------------------------------------------------")
	fmt.Println("Note: Memory reused between tasks, GC runs between iterations")
	measureMemory("single_threaded", runSingleThreaded, numTasks, sizeMB)

	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	fmt.Println("\n------------------------------------------------------------")
	fmt.Println("MULTI-THREADED (Goroutines - Shared Memory)")
	fmt.Println("------------------------------------------------------------")
	fmt.Println("Note: All goroutines share memory space, run concurrently")
	measureMemory("multi_threaded", runMultiThreaded, numTasks, sizeMB)

	fmt.Println("\n============================================================")
	fmt.Println("SUMMARY")
	fmt.Println("============================================================")
	fmt.Printf(`
Expected results:
- Single-threaded: ~%d MB peak (one task at a time, GC between)
- Multi-threaded: ~%d MB peak (all goroutines run in parallel)

Note: Go doesn't need multiprocessing for CPU parallelism;
goroutines already provide it without per-process interpreter overhead.
`, sizeMB, numTasks*sizeMB)
}
