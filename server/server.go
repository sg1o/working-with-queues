package main

import (
	"bufio"
	"crypto/rand"
	"flag"
	"fmt"
	"log"
	"math/big"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
)

// WorkRequest defines the structure of a request from a client
type WorkRequest struct {
	Data   string
	Client net.Conn
}

var (
	queue     chan WorkRequest
	queueSize int32 // Atomic counter to keep track of the queue size
)

func main() {
	// Define a flag for the port number
	port := flag.String("port", "12345", "Port number to listen on")
	flag.Parse()

	// Initialize the queue
	queue = make(chan WorkRequest, 100)

	// Start a fixed number of workers
	numWorkers := 4
	for i := 0; i < numWorkers; i++ {
		go worker(i, queue)
	}

	// Listen on the specified TCP port
	addr := fmt.Sprintf(":%s", *port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", *port, err)
	}
	defer listener.Close()
	log.Printf("Server is listening on port %s", *port)

	// Accept connections in a loop
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}

		log.Printf("New client connected: %s", conn.RemoteAddr())
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer func() {
		log.Printf("Client disconnected: %s", conn.RemoteAddr())
		conn.Close()
	}()
	log.Printf("Accepted new connection from %s", conn.RemoteAddr())

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		text := scanner.Text()
		work := WorkRequest{Data: text, Client: conn}
		queue <- work
		atomic.AddInt32(&queueSize, 1)
		log.Printf("Enqueued request from %s. Queue size: %d", conn.RemoteAddr(), atomic.LoadInt32(&queueSize))
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading from connection: %v", err)
	}
}

func worker(id int, queue chan WorkRequest) {
	for work := range queue {
		msg := fmt.Sprintf("Your task is being processed by worker %d", id)
		fmt.Fprintf(work.Client, "Status: %s\n", msg)
		log.Printf("Worker %d: Processing request. Queue size before processing: %d", id, atomic.LoadInt32(&queueSize))

		result := processWork(work)
		fmt.Fprintf(work.Client, "Result: %s\n", result)
		log.Printf("Worker %d: Result sent to client. Queue size after processing: %d", id, atomic.AddInt32(&queueSize, -1))
	}
}

func processWork(work WorkRequest) string {
	size, _ := strconv.Atoi(work.Data)
	prime, _ := generateSafePrimeConcurrent(size, size*2, 2)
	if prime == nil {
		return fmt.Sprintf("Size to small!")
	}
	return fmt.Sprintf("Prime with size %d: %d", size, prime)
}

// generatePrimeNumber generates a prime number of a specified bit size.
func generatePrimeNumber(minBits, maxBits int) (*big.Int, error) {
	bitSize, err := rand.Int(rand.Reader, big.NewInt(int64(maxBits-minBits+1)))
	if err != nil {
		return nil, err
	}
	bitSize.Add(bitSize, big.NewInt(int64(minBits)))

	prime, err := rand.Prime(rand.Reader, int(bitSize.Int64()))
	if err != nil {
		return nil, err
	}
	return prime, nil
}

// smallPrimes is a precalculated list of small prime numbers for quick
// divisibility tests.
var smallPrimes = []int{2, 3, 5, 7, 11, 13, 17, 19, 23, 29, 31, 37, 41, 43, 47,
	53, 59, 61, 67, 71, 73, 79, 83, 89, 97}

// isDivisibleByAnySmallPrime checks if the given number is divisible by any of
// the small primes.
func isDivisibleByAnySmallPrime(num *big.Int) bool {
	for _, prime := range smallPrimes {
		if new(big.Int).Mod(num, big.NewInt(int64(prime))).Cmp(big.NewInt(0)) == 0 {
			return true
		}
	}
	return false
}

// generateSafePrime generates a safe prime number of a specified bit size.
func generateSafePrime(minBits, maxBits int) (*big.Int, error) {
	halfMinBits := (minBits + 1) / 2
	halfMaxBits := (maxBits + 1) / 2

	for {
		// Generate a candidate for (p-1)/2
		halfPrime, err := generatePrimeNumber(halfMinBits, halfMaxBits)
		if err != nil {
			return nil, err
		}

		// Skip if divisible by any small prime
		if isDivisibleByAnySmallPrime(halfPrime) {
			continue
		}

		// Construct the safe prime candidate: p = 2*halfPrime + 1
		prime := new(big.Int).Lsh(halfPrime, 1) // Left shift to multiply by 2
		prime.Add(prime, big.NewInt(1))

		// Check if the candidate is prime with fewer iterations
		if prime.ProbablyPrime(5) { // Reduced iterations for faster check
			return prime, nil
		}
	}
}

// generateSafePrimeConcurrent generates a safe prime number of a specified bit
// size using goroutines.
func generateSafePrimeConcurrent(minBits, maxBits int, numGoroutines int) (*big.Int, error) {
	fmt.Println("[!]Generating safe prime")
	primeChan := make(chan *big.Int)
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			if prime, err := generateSafePrime(minBits, maxBits); err == nil {
				select {
				case primeChan <- prime:
				default:
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(primeChan)
	}()

	for prime := range primeChan {
		return prime, nil
	}

	return nil, nil
}
