package main

import (
	"hash/fnv"
	"io"
	"log"
	"math"
	"net"
	"sync"

	"golang.org/x/time/rate"
)

type LoadBalancer struct {
	config *Config
	// next is the index of the backend to use for the next connection for roundrobin
	next int
	// connCounts tracks active connections for leastconn
	connCounts map[string]int
	// we use a mutex to handle next and connCounts since they are shared variables to track all connections
	mutex sync.Mutex
	// The new HealthChecker instance
	healthChecker *HealthChecker
}

// createLoadBalancer initializes the LoadBalancer, including connection counts and the HealthChecker.
func createLoadBalancer(config *Config) *LoadBalancer {

	hc := createHealthChecker(config.Backends)
	hc.Start()

	connCounts := make(map[string]int)
	for _, backend := range config.Backends {
		connCounts[backend] = 0
	}

	return &LoadBalancer{
		config:        config,
		next:          0,
		connCounts:    connCounts,
		healthChecker: hc,
	}
}

// selectBackend chooses a backend based on the configured algorithm, using only healthy backends.
func (loadBalancer *LoadBalancer) selectBackend(clientIP string) string {

	healthyBackends := loadBalancer.healthChecker.GetHealthyBackends()

	if len(healthyBackends) == 0 {
		log.Println("No healthy backends available.")
		return ""
	}

	switch loadBalancer.config.Algorithm {
	case "roundrobin":
		return loadBalancer.roundRobin(healthyBackends)
	case "leastconn":
		return loadBalancer.leastConn(healthyBackends)
	case "hashing":
		return loadBalancer.hashing(clientIP, healthyBackends)
	default:
		return loadBalancer.roundRobin(healthyBackends)
	}
}

func (loadBalancer *LoadBalancer) roundRobin(backends []string) string {
	loadBalancer.mutex.Lock()
	defer loadBalancer.mutex.Unlock()

	// Use modulo with the length of the *healthy* backends
	// We must ensure 'next' doesn't exceed the original backends list length,
	// the selection is made from the healthy list.

	if loadBalancer.next >= len(backends) {
		loadBalancer.next = 0
	}

	backend := backends[loadBalancer.next]
	// Increment 'next' for the next connection, wrapping around the healthy list length
	loadBalancer.next = (loadBalancer.next + 1) % len(backends)
	return backend
}

func (loadBalancer *LoadBalancer) leastConn(backends []string) string {
	loadBalancer.mutex.Lock()
	defer loadBalancer.mutex.Unlock()

	minConns := math.MaxInt32
	var selectedBackend string

	// Iterate over only the healthy backends
	for _, backend := range backends {
		count := loadBalancer.connCounts[backend]
		if count < minConns {
			minConns = count
			selectedBackend = backend
		}
	}

	return selectedBackend
}

func (loadBalancer *LoadBalancer) hashing(clientIP string, backends []string) string {
	// fnv == Package fnv implements non-cryptographic hash functions for us
	hash := fnv.New32a()
	// hash the clientIP with 32a algo
	hash.Write([]byte(clientIP))

	// with modulo we can distribute the indexes in function of the number of *healthy* backends we have
	index := int(hash.Sum32()) % len(backends)
	return backends[index]
}

func (loadBalancer *LoadBalancer) increment(backendHost string) {
	loadBalancer.mutex.Lock()
	defer loadBalancer.mutex.Unlock()
	loadBalancer.connCounts[backendHost]++
}

func (loadBalancer *LoadBalancer) decrement(backendHost string) {
	loadBalancer.mutex.Lock()
	defer loadBalancer.mutex.Unlock()
	loadBalancer.connCounts[backendHost]--
}

func (loadBalancer *LoadBalancer) handleConnection(clientConnection net.Conn) {
	defer clientConnection.Close() // prepare the closing of connections if handle Connection ends

	// only used for the hashing algorithm
	clientIP, _, err := net.SplitHostPort(clientConnection.RemoteAddr().String())
	if err != nil {
		log.Printf("Failed to parse client IP: %v", err)
	}

	backendHost := loadBalancer.selectBackend(clientIP)

	if backendHost == "" {
		log.Printf("Could not select a healthy backend for %s. Closing connection.", clientConnection.RemoteAddr())
		return
	}

	backendConnection, err := net.Dial("tcp", backendHost)
	if err != nil {
		log.Printf("Failed to connect to backend %s: %v", backendHost, err)
		return
	}
	defer backendConnection.Close()

	// Increment connection count for leastconn algorithm
	loadBalancer.increment(backendHost)
	defer loadBalancer.decrement(backendHost)

	// Convert Rate (MB/s) to bytes/second
	// 1 MB/s = 1024 * 1024 Bytes/s
	rateInBytes := loadBalancer.config.Rate * 1024 * 1024
	limiter := rate.NewLimiter(rate.Limit(rateInBytes), int(rateInBytes))
	// Forward traffic in both directions
	var wg sync.WaitGroup
	wg.Add(2)

	// Client -> Backend (applying rate limiting to the client's data transfer)
	go func() {
		defer wg.Done()
		rateLimitedReader := createRateLimitedReader(clientConnection, limiter) // wrap the client connection Reader with our rateLimiter
		//dataplane: forwarding or raw tcp traffic
		_, err := io.Copy(backendConnection, rateLimitedReader)
		if err != nil && err != io.EOF {
			log.Printf("Error copying client->backend: %v", err)
		}
	}()

	// Backend -> Client (applying rate limiting to the backend's data transfer)
	go func() {
		defer wg.Done()
		rateLimitedReader := createRateLimitedReader(backendConnection, limiter)
		//dataplane: forwarding or raw tcp traffic
		_, err := io.Copy(clientConnection, rateLimitedReader) // Backend -> Client
		if err != nil && err != io.EOF {
			log.Printf("Error copying backend->client: %v", err)
		}
	}()

	wg.Wait()
	log.Printf("Connection from %s to %s closed", clientConnection.RemoteAddr(), backendHost)
}
