package main

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// the embed module works as a compiler directive that instructs the compiler to create the variable defined in the next line, assigning its value to the content of the embedded file. It's necessary to have the commented embed directive (//embed rootPage.html) directly preceding the variariable declaration.

//go:embed rootPage.html
var rootPageHTML []byte

// create variable encoding a custom error value that indicates the pico w server returned invalid content
var errInvalidResponse = errors.New("unexpected response from the server")

// create a struct to hold the Celcius and Fahrenheit values parsed from the JSON
type tempValues struct {
	TempC float64 `json:"tempC"`
	TempF float64 `json:"tempF"`
}

// getTempValues()method will connect to the pico w server, make a request, receive the data, and then parse the JSON payload into an instance of the tempValues struct that will be used later to return metrics to Prometheus
func (tv *tempValues) getTempValues(client *http.Client, url string) error {
	response, err := client.Get(url)
	if err != nil {
		log.Println(err)
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		err := fmt.Errorf(
			"%w: invalid status code: %s",
			errInvalidResponse,
			response.Status,
		)
		log.Println(err)
		return err
	}
	
	if err := json.NewDecoder(response.Body).Decode(tv); err != nil {
		log.Println(err)
		return err
	}
	return nil
}

// define type metrics to cache the metrics values before sending the data to Prometheus. Goal of this type is:
//		1. define a concurrency-safe model to access data in case more than one instance of Prometheus requests them simultaneously
//		2. cache the values received from the pico w server for two seconds, preventing overloading the Pico W in case the exporter receives many concurrent requests
//  the metrics struct includes: an instance of the struct tempValues to store the metrics received, a field up that indicates when the pico w is up and running, an expire field that that represents a time when the cache expires and it must obtain new values from the pico server, and it embeds type sync.RWMutex from the standard library sync which allows locking and unlocking the struct for safe concurrent access
type metrics struct {
	results *tempValues
	up			float64
	expire	time.Time
	sync.RWMutex
}

// getMetrics() method is associated with the metrics type. It returns the existing values if the cache hasn't expired, or if the cach has expired, it uses the previously defined method getTempValues to obtain new temperature values from the pico.
func (m *metrics) getMetrics(client *http.Client, url string) *metrics {
	m.Lock()
	defer m.Unlock()

	if time.Now().Before(m.expire) {
		return m
	}

	m.up = 1
	if err := m.results.getTempValues(client, url); err != nil {
		m.up = 0
		m.results.TempC = 0
		m.results.TempF = 0
	}

	m.expire = time.Now().Add(2 * time.Second)
	return m
}

// define a set of method to access the metrics values safely (concurrently), by locking the struct instance for reading before returning their values, and unlocking it when it is done
func (m *metrics) tempC() float64 {
	m.RLock()
	defer m.RUnlock()
	return m.results.TempC
}
func (m *metrics) tempF() float64{
	m.RLock()
	defer m.RUnlock()
	return m.results.TempF
}

func (m *metrics) status() float64 {
	m.RLock()
	defer m.RUnlock()
	return m.up
}

// Export metrics over HTTP interface with HTTP server written using Go's net/http and Prometheus's promhttp packages. 
// create a new HTTP request multiplexer to handle incoming connections and dispatch them to the appropriate handler (using Prometheus promauto package to define the target metrics)
func newMux(url string) http.Handler {
	mux := http.NewServeMux()

	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	m := &metrics{
		results: &tempValues{},
	}
	promauto.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name: 				"pico_temperature",
			Help:					"Pico Sensor Temperature.",
			ConstLabels:	prometheus.Labels{"unit": "celsius"},
		},
		func() float64 {
			return m.getMetrics(client, url).tempC()
		},
	)
	promauto.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name:					"pico_temperature",
			Help:					"Pico Sensor Temperature.",
			ConstLabels:	prometheus.Labels{"unit": "fahrenheit"},
		},
		func() float64 {
			return m.getMetrics(client, url).tempF()
		},
	)
	promauto.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name: "pico_up",
			Help: "Pico Sensor Server Status.",
		},
		func() float64 {
			return m.getMetrics(client, url).status()
		},
	)
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.Write(rootPageHTML)
	})
	mux.Handle("/metrics", promhttp.Handler())

	return mux
}

// main() function to start the http server on port :3030 using the multiplexer. it also captures the pico w server URL and defines timeouts
func main() {
	time.Sleep(time.Second * 15)

	picoURL := os.Getenv("PICO_SERVER_URL")
	s := &http.Server{
		Addr: 				":3030",
		Handler:			newMux(picoURL),
		ReadTimeout:	10 * time.Second,
		WriteTimeout:	10 * time.Second,
	}
	if err := s.ListenAndServe(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}