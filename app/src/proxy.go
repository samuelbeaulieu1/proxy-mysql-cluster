package main

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"regexp"
	"strings"
	"sync"
)

// Client request instance on the main proxy
type proxyInstance struct {
	proxy *proxy

	mysql  net.Conn
	client net.Conn

	mysqlWriterChan      chan byte
	mysqlHasPendingQuery bool
	mysqlLock            sync.Mutex
	database             string

	nullDbPacket []byte
}

// Main proxy server to listen to incoming client requests
type proxy struct {
	port       string
	readWriter ReaderWriter
}

// Singleton main proxy server
var mysqlProxy *proxy

func GetProxy() *proxy {
	return mysqlProxy
}

// Start the proxy on the specified port and indefinitely
// listen to incoming client requests
func StartProxy(port string, readWriter ReaderWriter) error {
	mysqlProxy = &proxy{
		port:       port,
		readWriter: readWriter,
	}

	// Listen to incoming TCP connections
	ln, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		return err
	}

	// Indefinitely read incoming client requests
	for {
		conn, err := ln.Accept()
		log.Printf("New connection accepted: %s\n", conn.RemoteAddr())
		if err != nil {
			log.Printf("Failed to accept new connection: %s\n", err.Error())
			continue
		}

		go mysqlProxy.Handle(conn)
	}
}

// Execute the client requests on the remote cluster
func (p *proxy) Handle(conn net.Conn) {
	// Start connection with the master node first
	// to authenticate the client and gather
	// information on the cluster
	master := GetCluster().Master
	mysql := ConnectRemoteMysql(master.host, master.port)

	// Greetings from mysql cluster
	p.readWriter.ReadWrite(mysql, conn)
	// Auth response from client
	p.readWriter.ReadWrite(conn, mysql)
	// Auth response from mysql cluster
	_, err := p.readWriter.ReadWrite(mysql, conn)
	// if auth failed, we exit
	if err != nil {
		return
	}

	// Initiating the client proxy instance to handle the requests
	instance := &proxyInstance{
		proxy: p,

		client: conn,
		mysql:  mysql,

		mysqlWriterChan:      make(chan byte),
		mysqlHasPendingQuery: false,
		mysqlLock:            sync.Mutex{},
	}
	// get the database the user initiated the request with
	instance.database = instance.getDatabase()
	// Then handle the requests
	instance.handleProxyLoop()
}

func (p *proxyInstance) handleProxyLoop() {
	go p.handleClientLoop()
	go p.handleMysqlServerLoop()
}

func (p *proxyInstance) handleClientLoop() {
	// Indefinitely handle incoming client requests until
	// the client exists
	for {
		// Read and wait for incoming client request
		buf, err := p.proxy.readWriter.Read(p.client)
		// EOF received, no more requests will be sent, close the connection
		if err != nil {
			break
		}

		// Handle the query on the cluster
		queriedNewDb, isSelect := p.handleClientQuery(buf)

		// Query was a DB change
		if queriedNewDb {
			// // Write client response for DB change
			p.proxy.readWriter.ReadWrite(p.client, p.mysql)
			// // Write to client new DB change
			p.proxy.readWriter.ReadWrite(p.mysql, p.client)
			p.database = p.getDatabase()
		} else if !isSelect {
			// Send write queries response from MySQL cluster
			// Read queries are handled earlier since they are
			// executed on slave nodes
			go p.queueMysqlResponse()
		}
	}
}

func (p *proxyInstance) handleMysqlServerLoop() {
	// Indefinitely write to the client MySQL cluster
	// responses to queries
	for {
		p.setMysqlServerBusyStatus(false)

		<-p.mysqlWriterChan

		p.setMysqlServerBusyStatus(true)
		p.proxy.readWriter.ReadWrite(p.mysql, p.client)
	}
}

func (p *proxyInstance) handleClientQuery(req *bytes.Buffer) (bool, bool) {
	// Regex expressions to check for select and database change
	selectDbRegex, _ := regexp.Compile(`^(select database\(\))`)
	regex, _ := regexp.Compile("^(select )")
	query := strings.TrimSpace(string(req.Bytes()[5:len(req.Bytes())]))
	queriedNewDb := selectDbRegex.MatchString(strings.ToLower(query))
	isSelect := regex.MatchString(strings.ToLower(query))

	// In case of select requests, the query is executed on slave nodes,
	// so we select a node based on the mode and execute on a separate
	// connection with the slave.
	// Then the reponse is returned to the inital TCP connection with the client
	if isSelect {
		slave := GetCluster().SelectSlave()
		log.Printf("[%s][%s] Executing '%s'\n", slave.hostType, fmt.Sprintf("%s:%d", slave.host, slave.port), query)

		// Send client request to slave
		result := slave.HandleQuery(query, p.database)
		// Send slave response to client
		p.proxy.readWriter.Write(bytes.NewBuffer(result), p.client)
	} else {
		// For write requests, we execute on the master
		master := GetCluster().Master
		log.Printf("[%s][%s] Executing '%s'\n", master.hostType, fmt.Sprintf("%s:%d", master.host, master.port), query)

		// Send client request to server
		p.proxy.readWriter.Write(req, p.mysql)
	}

	return queriedNewDb, isSelect
}

// Basic lock to prevent multiple write requests to start for the same query
func (p *proxyInstance) setMysqlServerBusyStatus(busy bool) {
	p.mysqlLock.Lock()
	p.mysqlHasPendingQuery = busy
	p.mysqlLock.Unlock()
}

// Schedule a new response to a client query
func (p *proxyInstance) queueMysqlResponse() {
	p.mysqlLock.Lock()
	if !p.mysqlHasPendingQuery {
		p.mysqlWriterChan <- 1
	}
	p.mysqlLock.Unlock()
}

// Get the database by executing a SELECT DATABASE() query on
// the MySQL cluster connection.
// Useful for select queries which are executed on separate connections.
func (p *proxyInstance) getDatabase() string {
	if p.nullDbPacket == nil {
		p.nullDbPacket = GetCluster().Master.HandleQuery("SELECT DATABASE()", "")
	}
	p.proxy.readWriter.ExecQuery(p.mysql, "SELECT DATABASE()")
	res, _ := p.proxy.readWriter.Read(p.mysql)
	if res == nil || res.String() == string(p.nullDbPacket) {
		return ""
	}

	dbRegex := regexp.MustCompile(`([a-zA-Z0-9_]+)+`)
	matches := dbRegex.FindAllStringSubmatch(res.String(), -1)
	if len(matches) == 0 {
		return ""
	}
	return matches[len(matches)-1][0]
}
