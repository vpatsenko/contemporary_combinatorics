package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"
)

const (
	HOST = "127.0.0.1"
	PORT = "8081"
)

func getRSSMiB() float64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return float64(m.Sys) / (1024 * 1024)
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("hello"))
}

func runServer() {
	addr := HOST + ":" + PORT
	http.HandleFunc("/", helloHandler)
	fmt.Printf("Server running on http://%s\n", addr)
	fmt.Println("Press Ctrl+C to stop")
	if err := http.ListenAndServe(addr, nil); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}

func makeRequest(client *http.Client, url string) bool {
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	return resp.StatusCode == 200
}

func runLoadTest(numRequests, concurrency int) {
	url := fmt.Sprintf("http://%s:%s/", HOST, PORT)

	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        concurrency,
			MaxIdleConnsPerHost: concurrency,
			IdleConnTimeout:     30 * time.Second,
		},
		Timeout: 10 * time.Second,
	}

	// Warm up
	for i := 0; i < 10; i++ {
		makeRequest(client, url)
	}

	rssBefore := getRSSMiB()

	work := make(chan struct{}, numRequests)
	for i := 0; i < numRequests; i++ {
		work <- struct{}{}
	}
	close(work)

	start := time.Now()

	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range work {
				makeRequest(client, url)
			}
		}()
	}

	wg.Wait()
	elapsed := time.Since(start)
	rssAfter := getRSSMiB()

	avgLatency := (elapsed.Seconds() / float64(numRequests)) * 1000

	fmt.Printf("workers: %d\n", concurrency)
	fmt.Printf("reqs: %d\n", numRequests)
	fmt.Printf("latency: %.2fms\n", avgLatency)
	fmt.Printf("rss_delta: %.1fMiB\n", rssAfter-rssBefore)
}

func runBoth(numRequests, concurrency int) {
	addr := HOST + ":" + PORT
	http.HandleFunc("/", helloHandler)

	server := &http.Server{Addr: addr}
	go func() {
		server.ListenAndServe()
	}()

	time.Sleep(300 * time.Millisecond)

	runLoadTest(numRequests, concurrency)

	server.Close()
}

func main() {
	mode := flag.String("mode", "both", "Run mode: server, client, or both")
	numRequests := flag.Int("n", 1000, "Number of requests")
	concurrency := flag.Int("c", 50, "Concurrency level")
	flag.Parse()

	// Also check positional argument for mode
	if flag.NArg() > 0 {
		*mode = flag.Arg(0)
	}

	switch *mode {
	case "server":
		runServer()
	case "client":
		runLoadTest(*numRequests, *concurrency)
	case "both":
		runBoth(*numRequests, *concurrency)
	default:
		fmt.Fprintf(os.Stderr, "Unknown mode: %s\n", *mode)
		os.Exit(1)
	}
}
