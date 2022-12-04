package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Cluster struct {
	mode ClusterMode

	Master *Host
	slaves []*Host
}

type ClusterMode int

// Modes: Direct hit, random, latency
const (
	MasterMode ClusterMode = iota
	RandomMode
	LatencyMode
)

// Singleton cluster
var cluster *Cluster

func GetCluster() *Cluster {
	return cluster
}

// Read the mode from input arguments,
// default to direct hit
func ReadModeFromArgs() ClusterMode {
	args := os.Args[1:]
	if len(args) > 0 {
		mode := strings.ToLower(args[0])
		switch mode {
		case "mastermode":
			return MasterMode
		case "randommode":
			return RandomMode
		case "latencymode":
			return LatencyMode
		}
	}

	return MasterMode
}

// Setup the cluster config from input args and env vars
func InitCluster(mode ClusterMode) {
	rand.Seed(time.Now().UnixNano())

	masterIP := os.Getenv("MASTER_NODE_IP")
	sqlPwd := os.Getenv("SQL_PASSWORD")
	sqlUser := os.Getenv("SQL_USER")
	nSlaves, _ := strconv.Atoi(os.Getenv("N_SLAVES"))
	slaves := []*Host{}
	for i := 0; i < nSlaves; i++ {
		slaveIP := os.Getenv(fmt.Sprintf("DATA_NODE%d_IP", i+1))
		slaves = append(slaves, &Host{
			host:     slaveIP,
			port:     3306,
			user:     sqlUser,
			password: sqlPwd,
			hostType: "SLAVE",
		})
	}

	cluster = &Cluster{
		mode: mode,
		Master: &Host{
			host:     masterIP,
			port:     3306,
			user:     sqlUser,
			password: sqlPwd,
			hostType: "MASTER",
		},
		slaves: slaves,
	}
}

// Selecting a slave node to execute a select query based on the mode
func (c *Cluster) SelectSlave() *Host {
	switch c.mode {
	case MasterMode:
		return c.Master
	case LatencyMode:
		return c.selectLeastLatencySlave()
	case RandomMode:
		return c.selectRandomSlave()
	default:
		return c.Master
	}
}

// Randomly select one of the slaves
func (c *Cluster) selectRandomSlave() *Host {
	index := rand.Intn(len(c.slaves))

	return c.slaves[index]
}

// Select a the slave with the least latency from
// ping command
func (c *Cluster) selectLeastLatencySlave() *Host {
	wg := sync.WaitGroup{}
	wg.Add(len(c.slaves))

	lock := sync.Mutex{}
	var leastLatency time.Duration
	leastLatencyIndex := -1

	// Start all on different threads
	for index, host := range c.slaves {
		go func(index int, host *Host) {
			time := host.Ping()

			lock.Lock()
			if leastLatencyIndex == -1 || leastLatency > time {
				leastLatency = time
				leastLatencyIndex = index
			}
			lock.Unlock()
			wg.Done()
		}(index, host)
	}

	wg.Wait()

	slave := c.slaves[leastLatencyIndex]
	log.Printf("Selected slave %s with latency %s\n", fmt.Sprintf("%s:%d", slave.host, slave.port), leastLatency)
	return slave
}
