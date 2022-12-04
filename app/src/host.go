package main

import (
	"time"

	"github.com/go-ping/ping"
)

type Host struct {
	host     string
	port     int
	user     string
	password string

	hostType string
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
