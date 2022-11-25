package main

import (
	"log"
	"os"
)

func main() {
	mode := ReadModeFromArgs()
	port := "3306"
	args := os.Args[1:]
	if len(args) > 1 {
		port = args[1]
	}

	InitCluster(mode)
	go StartSlaveProxy("3316", &ProxyReaderWriter{})
	err := StartProxy(port, &ProxyReaderWriter{})
	if err != nil {
		log.Fatal(err)
	}
}
