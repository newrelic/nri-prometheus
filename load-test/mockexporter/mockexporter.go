package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	metrics := flagEnvString("metrics", "", "path to the metrics file to serve")
	latency := flagEnvInt("latency", 0, "artificial latency to induce in the responses (milliseconds)")
	latencyVariation := flagEnvInt("latency-variation", 0, "randomly variate latency by +- this value (percentage)")
	maxRoutines := flagEnvInt("max-routines", 0, "maximum number of requests to handle in parallel")
	listenAddress := flagEnvString("addr", ":9940", "address:port pair to listen in")
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

// Wrapper to get default from environment if present
func flagEnvString(name, defaultValue, usage string) *string {
	val := os.Getenv(strings.ToUpper(strings.ReplaceAll(name, "-", "_")))
	if val == "" {
		val = defaultValue
	}

	return flag.String(
		name,
		val,
		usage,
	)
}

// Wrapper to get default from environment if present
func flagEnvInt(name string, defaultValue int, usage string) *int {
	val, err := strconv.Atoi(os.Getenv(strings.ToUpper(strings.ReplaceAll(name, "-", "_"))))
	if err != nil {
		val = defaultValue
	}

	return flag.Int(
		name,
		val,
		usage,
	)
}

type metricsServer struct {
	MetricsFile      string
	Latency          int
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

	ms.metricsBuffer, err = io.ReadAll(metricsFile)
	if err != nil {
		return fmt.Errorf("could not load metrics into memory: %v", err)
	}

	log.Println("metrics loaded from disk")

	if ms.MaxRoutines != 0 {
		ms.waiter = make(chan struct{}, ms.MaxRoutines)
	}

	log.Printf("%s", "starting server in "+address)
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
