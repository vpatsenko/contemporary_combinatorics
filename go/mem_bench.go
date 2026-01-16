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
	if rss > 10*1024*1024*1024 {
		return rss / (1024 * 1024)
	}
	return rss / 1024
}

func getCurrentRSSMB() float64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return float64(m.Sys) / (1024 * 1024)
}

type PeakMemoryTracker struct {
	peakRSS  atomic.Value
	stopChan chan struct{}
	wg       sync.WaitGroup
}

func NewPeakMemoryTracker() *PeakMemoryTracker {
	t := &PeakMemoryTracker{
		stopChan: make(chan struct{}),
	}
	t.peakRSS.Store(float64(0))
	return t
}

func (t *PeakMemoryTracker) Start() {
	t.peakRSS.Store(getCurrentRSSMB())
	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				current := getCurrentRSSMB()
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

func memoryIntensiveTask(sizeMB int) int64 {
	numBytes := sizeMB * 1024 * 1024
	data := make([]byte, numBytes)

	for i := 0; i < len(data); i += 1024 * 1024 {
		data[i] = byte(i % 256)
	}

	time.Sleep(100 * time.Millisecond)

	var total int64
	for i := 0; i < len(data); i += 1024 * 1024 {
		total += int64(data[i])
	}

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

	rssBefore := getCurrentRSSMB()

	tracker := NewPeakMemoryTracker()
	tracker.Start()

	start := time.Now()
	fn(numTasks, sizeMB)
	elapsed := time.Since(start)

	peakRSS := tracker.Stop()
	rssAfter := getCurrentRSSMB()

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

	fmt.Println("\nNote: Go has no GIL - goroutines share memory and run in parallel")

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
	baselineRSS := getCurrentRSSMB()
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

Note: Go doesn't have built-in multiprocessing like Python.
Goroutines already provide parallelism without process overhead.
`, sizeMB, numTasks*sizeMB)
}
