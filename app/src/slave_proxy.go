package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"os/exec"
)

// Query to be executed on specified host
type slaveQuery struct {
	slave    *Host
	query    string
	database string
	output   chan []byte
}

// Proxy to cluster with queries channel to be exected on cluster
type slaveProxy struct {
	queries    chan *slaveQuery
	port       string
	readWriter ReaderWriter
}

// Singleton proxy
var slaveProxyInstance *slaveProxy

func GetSlaveProxy() *slaveProxy {
	return slaveProxyInstance
}

/*
Start cluster proxy to listen on specified port

Will listen to all incoming MySQL TCP connections
and will wait for incoming client request from
the channel

After receiving both the TCP connection and client request,
the request will be executed on the cluster and the response
returned by the output channel provided
*/
func StartSlaveProxy(port string, readWriter ReaderWriter) {
	slaveProxyInstance = &slaveProxy{
		queries:    make(chan *slaveQuery),
		port:       port,
		readWriter: readWriter,
	}

	// Listen to incoming TCP connections
	ln, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		panic(err)
	}

	// Listen indefinitely for new client queries
	for {
		query := <-slaveProxyInstance.queries
		// Start the MySQL client connection to communicate with the cluster
		go slaveProxyInstance.startMysqlClient(query)
		conn, err := ln.Accept()
		log.Printf("New connection accepted: [%s] %s\n", query.database, query.query)
		if err != nil {
			log.Printf("Failed to accept new slave connection: %s\n", err.Error())
			continue
		}

		// After the connection is created and the query received,
		// send the query and get the result
		go slaveProxyInstance.Handle(conn, query)
	}
}

// Starting the MySQL client connection to communicate with the cluster
func (s *slaveProxy) startMysqlClient(query *slaveQuery) {
	user := fmt.Sprintf("-u%s", query.slave.user)
	pwd := fmt.Sprintf("-p'%s'", query.slave.password)
	port := fmt.Sprintf("-P %s", s.port)
	queryStr := fmt.Sprintf("-e '%s'", query.query)
	mysql := fmt.Sprintf("mysql %s -h 127.0.0.1 %s %s %s %s", port, user, pwd, query.database, queryStr)
	cmd := exec.Command("bash", "-c", mysql)

	cmd.Run()
}

// Handling the client query on the cluster
func (s *slaveProxy) Handle(conn net.Conn, query *slaveQuery) error {
	master := GetCluster().Master
	// First, connect with the remote cluster node
	mysql := ConnectRemoteMysqlSlave(master.host, master.port, query.slave)

	// Greetings from mysql cluster node
	s.readWriter.ReadWrite(mysql, conn)
	// Auth response from client
	s.readWriter.ReadWrite(conn, mysql)
	// Auth response from mysql cluster node
	s.readWriter.ReadWrite(mysql, conn)
	// Send client query to mysql cluster node
	s.readWriter.ReadWrite(conn, mysql)

	// Response buffer containing all the packets
	response := &bytes.Buffer{}

	// Read all client queries after this point and send
	// all to the mysql cluster node
	go func() {
		io.Copy(mysql, conn)
	}()

	// Read all the responses from the mysql cluster node
	// until an EOF packet is sent, which indicates that
	// the request is done
	for {
		curr, err := s.readWriter.Read(mysql)
		if err != nil {
			break
		}

		// Save the response in the buffer
		response.Write(curr.Bytes())
		conn.Write(curr.Bytes())
	}

	// At this point, the request is done and the response is saved,
	// so we send it back with the output channel
	query.output <- response.Bytes()

	return nil
}
