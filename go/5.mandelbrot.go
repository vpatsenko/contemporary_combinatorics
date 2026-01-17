package main

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

const (
	SIZE     = 4000
	MAX_ITER = 50
)

func getRSSMiB() float64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return float64(m.HeapAlloc) / (1024 * 1024)
}

func computeRow(y int) []byte {
	row := make([]byte, (SIZE+7)/8)
	c1 := 2.0 / float64(SIZE)
	ci := float64(y)*c1 - 1.0

	for x := 0; x < SIZE; x++ {
		cr := float64(x)*c1 - 1.5
		zr, zi := cr, ci

		inside := true
		for i := 0; i < MAX_ITER; i++ {
			zr2, zi2 := zr*zr, zi*zi
			if zr2+zi2 > 4.0 {
				inside = false
				break
			}
			zi = 2.0*zr*zi + ci
			zr = zr2 - zi2 + cr
		}

		if inside {
			row[x/8] |= (128 >> (x % 8))
		}
	}

	return row
}

func mandelbrotSequential() [][]byte {
	result := make([][]byte, SIZE)
	for y := 0; y < SIZE; y++ {
		result[y] = computeRow(y)
	}
	return result
}

func mandelbrotThreaded() [][]byte {
	result := make([][]byte, SIZE)
	var wg sync.WaitGroup

	workers := runtime.GOMAXPROCS(0)
	jobs := make(chan int, SIZE)

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for y := range jobs {
				result[y] = computeRow(y)
			}
		}()
	}

	for y := 0; y < SIZE; y++ {
		jobs <- y
	}
	close(jobs)

	wg.Wait()
	return result
}

func benchmark(name string, fn func() [][]byte) {
	runtime.GC()
	rssBefore := getRSSMiB()
	start := time.Now()

	_ = fn()

	elapsed := time.Since(start)
	rssAfter := getRSSMiB()

	fmt.Printf("%s:\n", name)
	fmt.Printf("  time: %dms\n", elapsed.Milliseconds())
	fmt.Printf("  rss_delta: %.1fMiB\n", rssAfter-rssBefore)
}

func main() {
	fmt.Printf("Mandelbrot %dx%d, max_iter=%d\n", SIZE, SIZE, MAX_ITER)
	fmt.Printf("GOMAXPROCS: %d\n\n", runtime.GOMAXPROCS(0))

	benchmark("sequential", mandelbrotSequential)
	fmt.Println()
	benchmark("threaded", mandelbrotThreaded)
}
