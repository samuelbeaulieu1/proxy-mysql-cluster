package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"os/exec"
)

type slaveQuery struct {
	slave    *Host
	query    string
	database string
	output   chan []byte
}

type slaveProxy struct {
	queries    chan *slaveQuery
	port       string
	readWriter ReaderWriter
}

var slaveProxyInstance *slaveProxy

func GetSlaveProxy() *slaveProxy {
	return slaveProxyInstance
}

func StartSlaveProxy(port string, readWriter ReaderWriter) {
	slaveProxyInstance = &slaveProxy{
		queries:    make(chan *slaveQuery),
		port:       port,
		readWriter: readWriter,
	}

	ln, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		panic(err)
	}

	for {
		query := <-slaveProxyInstance.queries
		go slaveProxyInstance.startMysqlClient(query)
		conn, err := ln.Accept()
		log.Printf("New slave connection accepted:  [%s:%d][%s] %s\n", query.slave.host, query.slave.port, query.database, query.query)
		if err != nil {
			log.Printf("Failed to accept new slave connection: %s\n", err.Error())
			continue
		}

		go slaveProxyInstance.Handle(conn, query)
	}
}

func (s *slaveProxy) startMysqlClient(query *slaveQuery) {
	user := fmt.Sprintf("-u%s", query.slave.user)
	pwd := fmt.Sprintf("-p'%s'", query.slave.password)
	port := fmt.Sprintf("-P %s", s.port)
	queryStr := fmt.Sprintf("-e '%s'", query.query)
	mysql := fmt.Sprintf("mysql %s -h 127.0.0.1 %s %s %s %s", port, user, pwd, query.database, queryStr)
	cmd := exec.Command("bash", "-c", mysql)

	cmd.Run()
}

func (s *slaveProxy) Handle(conn net.Conn, query *slaveQuery) error {
	mysql := ConnectRemoteMysql(query.slave.host, query.slave.port)

	// Greetings from slave mysql
	s.readWriter.ReadWrite(mysql, conn)
	// Auth from client
	s.readWriter.ReadWrite(conn, mysql)
	// Auth response from slave mysql
	s.readWriter.ReadWrite(mysql, conn)
	// Query from client
	s.readWriter.ReadWrite(conn, mysql)

	// Response from mysql
	response := &bytes.Buffer{}

	go func() {
		io.Copy(mysql, conn)
	}()

	for {
		curr, err := s.readWriter.Read(mysql)
		if err != nil {
			break
		}

		response.Write(curr.Bytes())
		conn.Write(curr.Bytes())
	}

	query.output <- response.Bytes()

	return nil
}
