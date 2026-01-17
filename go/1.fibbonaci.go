package main

import (
	"fmt"
	"math/big"
	"runtime"
	"sync"
	"time"
)

func measureExecutionTime(name string, fn func()) {
	start := time.Now()
	fn()
	elapsed := time.Since(start)
	fmt.Printf("%s took %.4f seconds.\n", name, elapsed.Seconds())
}

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

func runSingleThreaded(nums []int) {
	for _, num := range nums {
		computeFibonacci(num)
	}
}

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
	fmt.Printf("Go Version: %s\n", runtime.Version())
	fmt.Printf("GOMAXPROCS: %d (number of CPUs available)\n", runtime.GOMAXPROCS(0))
	fmt.Printf("NumCPU: %d\n", runtime.NumCPU())

	fmt.Println("\nNote: Go has no GIL - goroutines execute in true parallelism")

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

	fmt.Println("\nNote: Go goroutines already provide true parallelism (no separate multiprocessing needed)")
}
