package main

import (
	"fmt"
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

const (
	MasterMode ClusterMode = iota
	RandomMode
	LatencyMode
)

var cluster *Cluster

func GetCluster() *Cluster {
	return cluster
}

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
		})
	}

	cluster = &Cluster{
		mode: mode,
		Master: &Host{
			host:     masterIP,
			port:     3306,
			user:     sqlUser,
			password: sqlPwd,
		},
		slaves: slaves,
	}
}

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

func (c *Cluster) selectRandomSlave() *Host {
	index := rand.Intn(2)

	return c.slaves[index]
}

func (c *Cluster) selectLeastLatencySlave() *Host {
	wg := sync.WaitGroup{}
	wg.Add(len(c.slaves))

	lock := sync.Mutex{}
	var leastLatency time.Duration
	leastLatencyIndex := -1

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

	return c.slaves[leastLatencyIndex]
}
