package main

import (
	"fmt"
	"log"
	"net"
)

func ConnectRemoteMysql(host string, port int) net.Conn {
	address := fmt.Sprintf("%s:%d", host, port)
	mysql, err := net.Dial("tcp", address)
	if err != nil {
		log.Printf("Failed to connect to remote MySQL: %s", err.Error())
		return nil
	}

	return mysql
}
