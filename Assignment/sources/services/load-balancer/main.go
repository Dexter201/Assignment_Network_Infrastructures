package main

import (
	"log"
	"net"
)

func main() {

	config, err := LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	lb := createLoadBalancer(config)

	// Start a TCP listener //layer 4
	log.Printf("TCP Load Balancer starting on :%s, Algorithm: %s", config.Port, config.Algorithm)
	listener, err := net.Listen("tcp", ":"+config.Port)
	if err != nil {
		log.Fatalf("Failed to start TCP listener: %v", err)
	}
	defer listener.Close()

	// Run an infinite loop to accept connections
	for {
		connection, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}

		// Handle each new connection in its own goroutine
		go lb.handleConnection(connection)
	}
}
