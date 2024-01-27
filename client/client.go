package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
)

func main() {
	// Define flags for the IP address and port
	ip := flag.String("ip", "localhost", "IP address of the server")
	port := flag.String("port", "12345", "Port number of the server")
	flag.Parse()

	// Build the server address and connect
	serverAddr := fmt.Sprintf("%s:%s", *ip, *port)
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		log.Fatalf("Failed to connect to the server: %v", err)
	}
	defer conn.Close()

	// CLI for user to send requests
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Connected to prime number generation server")
	fmt.Println("Type a number and press ENTER to request a prime number close to that size.")

	for {
		fmt.Print("-> ")
		text, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("Failed to read input: %v", err)
			continue
		}

		text = strings.TrimSpace(text) // Remove all leading and trailing whitespace

		// Send the text to the server
		_, err = conn.Write([]byte(text + "\n"))
		if err != nil {
			log.Printf("Failed to send data to server: %v", err)
			continue
		}

		// Read and process the server response
		processServerResponse(conn)
	}
}

// processServerResponse handles incoming messages from the server for a single request
func processServerResponse(conn net.Conn) {
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		text := scanner.Text()
		fmt.Printf("Server says: %s\n", text)
		if strings.Contains(text, "Result") {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		log.Printf("Error reading from server: %v", err)
	}
}
