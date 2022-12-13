package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/go-ping/ping"
	"github.com/sgreben/sshtunnel"
	"golang.org/x/crypto/ssh"
)

type Host struct {
	host     string
	port     int
	user     string
	password string

	hostType string

	sshConn *ssh.Client
	lock    sync.Mutex
}

// Pre establish the ssh tunnel connection once at startup
func (h *Host) establishTunnel() {
	defer func() {
		h.lock.Unlock()
	}()
	h.lock.Lock()
	if h.sshConn != nil {
		return
	}

	address := fmt.Sprintf("%s:%d", h.host, 22)
	conn, _ := ssh.Dial("tcp", address, h.getSshConfig())
	h.sshConn = conn
}

func (h *Host) getSshClient() *ssh.Client {
	return h.sshConn
}

func (h *Host) getSshConfig() *ssh.ClientConfig {
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

	return &clientConfig
}

// Handle a select query on a cluster node
// Wait for the output and return it
func (h *Host) HandleQuery(clientQuery string, database string) []byte {
	output := make(chan []byte)
	query := &slaveQuery{
		slave:    h,
		query:    clientQuery,
		database: database,
		output:   output,
	}

	GetSlaveProxy().queries <- query
	result := <-output

	return result
}

// Calculate ping time of cluster node
// Send 1 ping only
func (h *Host) Ping() time.Duration {
	pinger, _ := ping.NewPinger(h.host)

	pinger.Count = 1
	pinger.Run()

	stats := pinger.Statistics()
	return stats.AvgRtt
}
