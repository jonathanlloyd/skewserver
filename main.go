package main

import (
	"fmt"
)

func main() {
	// Listen for incoming TCP connections on port 61613
	// Create a new (pooling?) goroutine for each connection
	// In each goroutine:
	//   Read from connection until complete frame is read
	//   Parse frame
	fmt.Println("Hello World!")
}
