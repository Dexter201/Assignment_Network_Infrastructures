package main

import (
	"log"
	"net"
)

func main() {
	// 1. Load configuration
	config, err := LoadConfig("config.yaml") // From your utils.go
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 2. Start the blocking TCP listener
	log.Printf("TCP Load Balancer starting on :%s", config.Port)
	listener, err := net.Listen("tcp", ":"+config.Port)
	if err != nil {
		log.Fatalf("Failed to start TCP listener: %v", err)
	}
	defer listener.Close()

	// 3. Run the infinite loop to accept connections
	// This is the blocking call that keeps the container alive
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}

		// Just log and close the connection for now
		go func(c net.Conn) {
			log.Printf("Accepted connection from %s", c.RemoteAddr())
			c.Close()
		}(conn)
	}
}
