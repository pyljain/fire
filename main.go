package main

import (
	"bytes"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

func main() {
	// Use flags to accept the following inputs
	concurrency := flag.Int("c", 5, "concurreny for HTTP requests to initiate")
	totalNumberOfRequests := flag.Int("n", 100, "total number of HTTP requests to make")
	method := flag.String("X", "GET", "pass in the HTTP method")
	var headers HeadersArray
	flag.Var(&headers, "H", "Header to add to the fired request")
	isInsecure := flag.Bool("isInsecure", false, "determines whether the initiated request would verify server certificate, defaulted to true")
	data := flag.String("d", "", "Body of the request")

	flag.Parse()

	endpoint := flag.Arg(0)
	if endpoint == "" {
		fmt.Printf("An HTTP endpoint must be specificed to use Fire")
		os.Exit(-1)
	}

	// Create a channel to make the go routine to pick up work
	workCh := make(chan struct{})
	statistics := stats{}
	wg := sync.WaitGroup{}

	// Start go routines
	for i := 0; i < *concurrency; i++ {

		go fireRequests(endpoint, isInsecure, *method, headers, *data, workCh, &statistics, &wg)
	}

	for req := 0; req < *totalNumberOfRequests; req++ {
		wg.Add(1)
		workCh <- struct{}{}
	}

	wg.Wait()
	close(workCh)

	averageResponseTime := int64(0)
	if statistics.totalSuccesses > 0 {
		averageResponseTime = statistics.totalTimeTaken / statistics.totalSuccesses
	}

	fmt.Printf("statistics \n AVERAGE RESPONSE TIME: %d \n TOTAL SUCCESSFUL REQS: %d \n TOTAL SUCCESSFUL FAILURES: %d \n LONGEST RUNNING REQUEST: %d\n", averageResponseTime, statistics.totalSuccesses, statistics.totalFailures, statistics.longestRunningRequest)

}

func fireRequests(endpoint string, isInsecure *bool, method string, headers []string, body string, workCh chan struct{}, statistics *stats, wg *sync.WaitGroup) {
	for range workCh {

		req, err := http.NewRequest(method, endpoint, bytes.NewBufferString(body))
		if err != nil {
			log.Printf("Error occured while constructing the outbound request %s", err)
		}

		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: *isInsecure},
		}

		for _, h := range headers {
			headerFragments := strings.Split(h, ":")
			req.Header.Add(headerFragments[0], headerFragments[1])
		}

		client := http.Client{Transport: tr}

		reqStartTime := time.Now().UTC()
		var statusCode int
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("Error occured while making the outbound request %s", err)
			statusCode = http.StatusInternalServerError
		} else {
			statusCode = resp.StatusCode
		}

		reqEndTime := time.Now().UTC()

		timeTaken := reqEndTime.Sub(reqStartTime).Milliseconds()

		// fmt.Printf("statusCode: %d \n time Taken: %d\n", statusCode, timeTaken)
		statistics.lock.Lock()
		if statusCode < 400 {
			statistics.totalSuccesses += 1
			statistics.totalTimeTaken += timeTaken
			if statistics.longestRunningRequest < int32(timeTaken) {
				statistics.longestRunningRequest = int32(timeTaken)
			}

			if statistics.shortestRunningRequest > int32(timeTaken) {
				statistics.shortestRunningRequest = int32(timeTaken)
			}
		} else {
			statistics.totalFailures += 1
			statistics.totalTimeTaken += timeTaken
		}

		statistics.lock.Unlock()
		wg.Done()
	}
}

type stats struct {
	totalSuccesses         int64
	totalFailures          int64
	totalTimeTaken         int64
	longestRunningRequest  int32
	shortestRunningRequest int32
	lock                   sync.Mutex
}
