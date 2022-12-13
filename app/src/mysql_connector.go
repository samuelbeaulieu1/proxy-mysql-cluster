package main

import (
	"fmt"
	"log"
	"net"
)

// Initiate a TCP connection with a remote MySQL node in the cluster
func ConnectRemoteMysql(host string, port int) net.Conn {
	address := fmt.Sprintf("%s:%d", host, port)
	mysql, err := net.Dial("tcp", address)
	if err != nil {
		log.Printf("Failed to connect to remote MySQL: %s", err.Error())
		return nil
	}

	return mysql
}

// Initiate a TCP connection with the remote MySQL node in the cluster
// through the selected slave node by using a ssh tunnel
func ConnectRemoteMysqlSlave(mysqlHost string, port int, bindHost *Host) net.Conn {
	address := fmt.Sprintf("%s:%d", mysqlHost, port)
	// Connect to the MySQL through the slave node and bind to the slave node
	mysql, err := bindHost.getSshClient().Dial("tcp", address)
	if err != nil {
		log.Printf("Failed to connect to remote MySQL: %s", err.Error())
		return nil
	}
	return mysql
}
