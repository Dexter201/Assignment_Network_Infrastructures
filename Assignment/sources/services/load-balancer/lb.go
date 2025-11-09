package lb

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
	//we use a mutex to handle next and connCounts since they are shared variables to track all conenctions
	mutex sync.Mutex
}

func createLoadBalancer(config *Config) *LoadBalancer {
	return &LoadBalancer{
		config: config,
		next:   0,
	}
}

func (lb *LoadBalancer) selectBackend(clientIP string) string {
	switch lb.config.Algorithm {
	case "roundrobin":
		return lb.roundRobin()
	case "leastconn":
		return lb.leastConn()
	case "hashing":
		return lb.hashing(clientIP)
	default:
		return lb.roundRobin()
	}
}

func (loadBalancer *LoadBalancer) roundRobin() string {
	loadBalancer.mutex.Lock()
	defer loadBalancer.mutex.Unlock()

	backend := loadBalancer.config.Backends[loadBalancer.next]
	loadBalancer.next = (loadBalancer.next + 1) % len(loadBalancer.config.Backends)
	return backend
}

func (loadBalancer *LoadBalancer) leastConn() string {
	loadBalancer.mutex.Lock()
	defer loadBalancer.mutex.Unlock()

	minConns := math.MaxInt32
	var selectedBackend string

	// Find the backend with the minimum connection count
	for _, backend := range loadBalancer.config.Backends {
		// TODO: This doesn't account for backends that are down.
		// Health checks will be needed here.
		count := loadBalancer.connCounts[backend]
		if count < minConns {
			minConns = count
			selectedBackend = backend
		}
	}
	return selectedBackend
}

func (loadBalancer *LoadBalancer) hashing(clientIP string) string {
	//fnv == Package fnv implements non-cryptographic hash functions for us
	hash := fnv.New32a()
	//hash the clientIP with 32a algo
	hash.Write([]byte(clientIP))
	// with modulo we can distribute the indexes in function of the number of backends we have
	index := int(hash.Sum32()) % len(loadBalancer.config.Backends)
	return loadBalancer.config.Backends[index]
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
	defer clientConnection.Close() //prepare the closing of connections if handle Connection ends

	//only used for the hashing algorithm
	clientIP, _, err := net.SplitHostPort(clientConnection.RemoteAddr().String())
	if err != nil {
		log.Printf("Failed to parse client IP: %v", err)
	}

	backendHost := loadBalancer.selectBackend(clientIP)

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
	rateInBytes := loadBalancer.config.Rate * 1024 * 1024
	limiter := rate.NewLimiter(rate.Limit(rateInBytes), int(rateInBytes))
	// Forward traffic in both directions
	var wg sync.WaitGroup
	wg.Add(2)

	//This goroutine is responsible for copying all data from the client to the backend,like POST /api/profile/me and the JSON data.
	go func() {
		defer wg.Done()
		rateLimitedReader := createRateLimitedReader(clientConnection, limiter) //wrap the Reader with our rateLimiter
		io.Copy(backendConnection, rateLimitedReader)                           // Client -> Backend
	}()
	//This goroutine is responsible for copying all data from the backend back to the client, like HTTP 200 OK and the resulting JSON.
	go func() {
		defer wg.Done()
		rateLimitedReader := createRateLimitedReader(clientConnection, limiter)
		io.Copy(clientConnection, rateLimitedReader) // Backend -> Client
	}()

	wg.Wait()
	log.Printf("Connection from %s to %s closed", clientConnection.RemoteAddr(), backendHost)
}
