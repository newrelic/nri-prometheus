package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"
)

func main() {
	metrics := flag.String("metrics", "", "path to the metrics file to serve")
	latency := flag.Int64("latency", 0, "artificial latency to induce in the responses (milliseconds)")
	latencyVariation := flag.Int("latency-variation", 0, "randomly variate latency by +- this value (percentage)")
	maxRoutines := flag.Int("max-routines", 0, "maximum number of requests to handle in parallel")
	listenAddress := flag.String("addr", ":9940", "address:port pair to listen in")
	flag.Parse()

	if *metrics == "" {
		fmt.Println("A metrics file (-metrics) must be specified:")
		flag.PrintDefaults()
		os.Exit(1)
	}

	ms := &metricsServer{
		MetricsFile:      *metrics,
		Latency:          *latency,
		LatencyVariation: *latencyVariation,
		MaxRoutines:      *maxRoutines,
	}

	log.Println(ms.ListenAndServe(*listenAddress))
}

type metricsServer struct {
	MetricsFile      string
	Latency          int64
	LatencyVariation int
	MaxRoutines      int

	metricsBuffer []byte
	waiter        chan struct{}
}

func (ms *metricsServer) ListenAndServe(address string) error {
	metricsFile, err := os.Open(ms.MetricsFile)
	if err != nil {
		return fmt.Errorf("could not open %s: %v", ms.MetricsFile, err)
	}

	ms.metricsBuffer, err = ioutil.ReadAll(metricsFile)
	if err != nil {
		return fmt.Errorf("could not load metrics into memory: %v", err)
	}

	log.Println("metrics loaded from disk")

	if ms.MaxRoutines != 0 {
		ms.waiter = make(chan struct{}, ms.MaxRoutines)
	}

	log.Printf("starting server in " + address)
	return http.ListenAndServe(address, ms)
}

func (ms *metricsServer) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	if ms.MaxRoutines != 0 {
		ms.waiter <- struct{}{}
		defer func() {
			<-ms.waiter
		}()
	}

	time.Sleep(ms.latency())
	_, _ = rw.Write(ms.metricsBuffer)
}

func (ms *metricsServer) latency() time.Duration {
	lat := time.Duration(ms.Latency) * time.Millisecond

	if ms.LatencyVariation == 0 {
		return lat
	}

	variation := float64(ms.LatencyVariation) / 100
	variation = (rand.Float64() - 0.5) * variation * 2 // Random in (-variation, variation)

	return time.Duration(float64(lat) + float64(lat)*variation)
}
