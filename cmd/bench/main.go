package main

import (
	"flag"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	url := flag.String("url", "http://127.0.0.1:9999/", "target URL")
	concurrency := flag.Int("c", 100, "concurrent connections")
	duration := flag.Int("t", 10, "test duration in seconds")
	flag.Parse()

	fmt.Printf("Benchmarking %s\n", *url)
	fmt.Printf("Concurrency: %d, Duration: %ds\n\n", *concurrency, *duration)

	var (
		totalRequests int64
		successCount  int64
		errorCount    int64
		totalLatency  int64 // microseconds
	)

	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        *concurrency,
			MaxIdleConnsPerHost: *concurrency,
			IdleConnTimeout:     30 * time.Second,
		},
		Timeout: 10 * time.Second,
	}

	ctx := make(chan struct{})
	var wg sync.WaitGroup

	// Spawn workers
	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx:
					return
				default:
				}

				start := time.Now()
				resp, err := client.Get(*url)
				elapsed := time.Since(start).Microseconds()

				atomic.AddInt64(&totalRequests, 1)
				atomic.AddInt64(&totalLatency, elapsed)

				if err != nil {
					atomic.AddInt64(&errorCount, 1)
					continue
				}
				resp.Body.Close()

				if resp.StatusCode == 200 {
					atomic.AddInt64(&successCount, 1)
				} else {
					atomic.AddInt64(&errorCount, 1)
				}
			}
		}()
	}

	// Let it run
	time.Sleep(time.Duration(*duration) * time.Second)
	close(ctx)
	wg.Wait()

	total := atomic.LoadInt64(&totalRequests)
	success := atomic.LoadInt64(&successCount)
	errors := atomic.LoadInt64(&errorCount)
	avgLatency := float64(atomic.LoadInt64(&totalLatency)) / float64(total) / 1000.0 // ms

	fmt.Printf("Results:\n")
	fmt.Printf("  Total Requests:  %d\n", total)
	fmt.Printf("  Success:         %d\n", success)
	fmt.Printf("  Errors:          %d\n", errors)
	fmt.Printf("  QPS:             %.2f req/s\n", float64(total)/float64(*duration))
	fmt.Printf("  Avg Latency:     %.2f ms\n", avgLatency)
}
