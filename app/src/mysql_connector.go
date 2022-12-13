package main

import (
	"fmt"
	"log"
	"net"
	"os"

	"github.com/sgreben/sshtunnel"
	"golang.org/x/crypto/ssh"
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
// through the selected slave node by first creating a ssh tunnel
func ConnectRemoteMysqlSlave(mysqlHost string, port int, bindHost string) net.Conn {
	privateKeyPath := os.Getenv("CLUSTER_PRIVATE_KEY_PATH")
	// Creating tunnel config first
	authConfig := sshtunnel.ConfigAuth{
		Keys: []sshtunnel.KeySource{{Path: &privateKeyPath}},
	}
	sshAuthMethods, _ := authConfig.Methods()
	clientConfig := ssh.ClientConfig{
		User:            "ubuntu",
		Auth:            sshAuthMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	tunnelConfig := sshtunnel.Config{
		SSHAddr:   fmt.Sprintf("%s:22", bindHost),
		SSHClient: &clientConfig,
	}

	address := fmt.Sprintf("%s:%d", mysqlHost, port)
	// Now connect to the MySQL and bind to the slave node
	mysql, _, err := sshtunnel.Dial("tcp", address, &tunnelConfig)
	if err != nil {
		log.Printf("Failed to connect to remote MySQL: %s", err.Error())
		return nil
	}
	return mysql
}
