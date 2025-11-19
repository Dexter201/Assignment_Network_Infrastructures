package main

import (
	"log"
	"net"
	"sync"
	"time"
)

// represents a backend service
// Backend holds the state of a single backend server
type Backend struct {
	URL   string
	Alive bool
	// RWMutex allows many readers (GetHealthyBackends) or one writer (SetAlive)
	//ensures no read / write at the same time
	mutex sync.RWMutex
}

// represents an instance of a Healthchecker
// it stores all the Backend services
// a ticker to limit the healthcheck rate
// a wait group to wait for goroutines to end
// a timeout to represent a dead backend
type HealthChecker struct {
	backends     []*Backend
	ticker       *time.Ticker
	wg           sync.WaitGroup
	checkTimeout time.Duration
}

// set a backend to alive or dead depending on the boolean in a threadsafe manner
func (backend *Backend) SetAlive(alive bool) {
	backend.mutex.Lock()
	defer backend.mutex.Unlock()
	backend.Alive = alive
}

// check if a backend is alive or dead in a threadsafe manner
func (backend *Backend) IsAlive() bool {
	backend.mutex.RLock()
	defer backend.mutex.RUnlock()
	return backend.Alive
}

// create a Healthchecker for a given number (URLs) of backends
func createHealthChecker(backendURLs []string) *HealthChecker {
	backends := make([]*Backend, len(backendURLs))
	for i, url := range backendURLs {
		backends[i] = &Backend{
			URL:   url,
			Alive: true, // Start optimistically
		}
	}
	return &HealthChecker{
		backends:     backends,
		ticker:       time.NewTicker(10 * time.Second), // Check every 10 seconds
		checkTimeout: 2 * time.Second,
	}
}

// Start begins the periodic health checks in a new goroutine
func (healthChecker *HealthChecker) Start() {
	log.Println("Starting health check service...")
	healthChecker.wg.Add(1)
	go func() {
		defer healthChecker.wg.Done()
		for range healthChecker.ticker.C {
			healthChecker.runHealthChecks()
		}
	}()
}

// Stop terminates the health check goroutine
func (healthChecker *HealthChecker) Stop() {
	log.Println("Stopping health check service...")
	healthChecker.ticker.Stop()
	healthChecker.wg.Wait() // Wait for the goroutine to finish
}

// runHealthChecks pings all backends concurrently
func (healthChecker *HealthChecker) runHealthChecks() {
	var wg sync.WaitGroup
	for _, backend := range healthChecker.backends {
		wg.Add(1)
		go func(backend *Backend) {
			defer wg.Done()
			// Ping the backend with a 2-second timeout
			//first implementation of a raw tcp healthcheck: not smart enough
			//conn, err := net.DialTimeout("tcp", backend.URL, 2*time.Second)

			//more intelligent healthcheck
			conn, err := net.DialTimeout("tcp", backend.URL, healthChecker.checkTimeout)
			wasAlive := backend.IsAlive()

			if err != nil {
				backend.SetAlive(false)
				if wasAlive { // Only log if the state changes
					log.Printf("Health check: Backend %s is DOWN", backend.URL)
				}
			} else {
				backend.SetAlive(true)
				if !wasAlive { // Only log if the state changes
					log.Printf("Health check: Backend %s is UP", backend.URL)
				}
				conn.Close()
			}
		}(backend)
	}
	wg.Wait()
}

// GetHealthyBackends returns a slice of URLs for all backends that are currently alive
func (healthChecker *HealthChecker) GetHealthyBackends() []string {
	var healthy []string
	for _, backend := range healthChecker.backends {
		if backend.IsAlive() {
			healthy = append(healthy, backend.URL)
		}
	}
	return healthy
}
