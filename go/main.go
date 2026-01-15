package main

import (
	"fmt"
	"math/big"
	"runtime"
	"sync"
	"time"
)

// measureExecutionTime wraps a function and prints its execution time
func measureExecutionTime(name string, fn func()) {
	start := time.Now()
	fn()
	elapsed := time.Since(start)
	fmt.Printf("%s took %.4f seconds.\n", name, elapsed.Seconds())
}

// computeFibonacci computes the Fibonacci number for a given n iteratively
// Uses big.Int to handle large numbers (like Python's arbitrary precision integers)
func computeFibonacci(n int) *big.Int {
	a := big.NewInt(0)
	b := big.NewInt(1)
	temp := new(big.Int)

	for i := 0; i < n; i++ {
		temp.Set(a)
		a.Set(b)
		b.Add(temp, b)
	}
	return a
}

// runSingleThreaded executes tasks sequentially
func runSingleThreaded(nums []int) {
	for _, num := range nums {
		computeFibonacci(num)
	}
}

// runMultiThreaded executes tasks concurrently using goroutines
func runMultiThreaded(nums []int) {
	var wg sync.WaitGroup
	wg.Add(len(nums))

	for _, num := range nums {
		go func(n int) {
			defer wg.Done()
			computeFibonacci(n)
		}(num)
	}

	wg.Wait()
}

func main() {
	// Print Go version and runtime information
	fmt.Printf("Go Version: %s\n", runtime.Version())
	fmt.Printf("GOMAXPROCS: %d (number of CPUs available)\n", runtime.GOMAXPROCS(0))
	fmt.Printf("NumCPU: %d\n", runtime.NumCPU())

	// Note: Go doesn't have a GIL - goroutines can run truly in parallel
	fmt.Println("\nNote: Go has no GIL - goroutines execute in true parallelism")

	// Run tasks on the same input size for comparison
	nums := make([]int, 10)
	for i := range nums {
		nums[i] = 300000
	}

	fmt.Println("\nRunning Single-Threaded Task:")
	measureExecutionTime("runSingleThreaded", func() {
		runSingleThreaded(nums)
	})

	fmt.Println("\nRunning Multi-Threaded Task (Goroutines):")
	measureExecutionTime("runMultiThreaded", func() {
		runMultiThreaded(nums)
	})

	// Note: Go doesn't have a direct equivalent to Python's multiprocessing
	// Goroutines already achieve true parallelism without a GIL
	// For process-based parallelism, you would typically use os/exec or similar
	fmt.Println("\nNote: Go goroutines already provide true parallelism (no separate multiprocessing needed)")
}
