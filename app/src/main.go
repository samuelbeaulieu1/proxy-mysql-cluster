package main

import (
	"log"
	"os"
)

func main() {
	// Read input arguments to script
	mode := ReadModeFromArgs()
	port := "3306"
	args := os.Args[1:]
	if len(args) > 1 {
		port = args[1]
	}

	// Init the cluster configuration from env vars and input arguments
	InitCluster(mode)

	// Start slave proxy for select queries
	// Internal use only
	go StartSlaveProxy("3316", &ProxyReaderWriter{})
	// Start main proxy for all client requests
	err := StartProxy(port, &ProxyReaderWriter{})
	if err != nil {
		log.Fatal(err)
	}
}
