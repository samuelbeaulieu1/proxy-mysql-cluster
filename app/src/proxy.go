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

type proxy struct {
	port       string
	readWriter ReaderWriter
}

var mysqlProxy *proxy

func GetProxy() *proxy {
	return mysqlProxy
}

func StartProxy(port string, readWriter ReaderWriter) error {
	mysqlProxy = &proxy{
		port:       port,
		readWriter: readWriter,
	}

	ln, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		return err
	}

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

func (p *proxy) Handle(conn net.Conn) {
	master := GetCluster().Master
	mysql := ConnectRemoteMysql(master.host, master.port)

	// Greetings from mysql
	p.readWriter.ReadWrite(mysql, conn)
	// Auth from client
	p.readWriter.ReadWrite(conn, mysql)
	// Auth response from mysql
	_, err := p.readWriter.ReadWrite(mysql, conn)
	if err != nil {
		return
	}

	instance := &proxyInstance{
		proxy: p,

		client: conn,
		mysql:  mysql,

		mysqlWriterChan:      make(chan byte),
		mysqlHasPendingQuery: false,
		mysqlLock:            sync.Mutex{},
	}
	instance.database = instance.getDatabase()
	instance.handleProxyLoop()
}

func (p *proxyInstance) handleProxyLoop() {
	go p.handleClientLoop()
	go p.handleMysqlServerLoop()
}

func (p *proxyInstance) handleClientLoop() {
	for {
		// Client request
		buf, err := p.proxy.readWriter.Read(p.client)
		if err != nil {
			break
		}

		queriedNewDb, isSelect := p.handleClientQuery(buf)
		if queriedNewDb {
			// // Write client response for DB change
			p.proxy.readWriter.ReadWrite(p.client, p.mysql)
			// // Write to client new DB change
			p.proxy.readWriter.ReadWrite(p.mysql, p.client)
			p.database = p.getDatabase()
		} else if !isSelect {
			go p.queueMysqlResponse()
		}
	}
}

func (p *proxyInstance) handleMysqlServerLoop() {
	for {
		p.setMysqlServerBusyStatus(false)

		<-p.mysqlWriterChan

		p.setMysqlServerBusyStatus(true)
		p.proxy.readWriter.ReadWrite(p.mysql, p.client)
	}
}

func (p *proxyInstance) handleClientQuery(req *bytes.Buffer) (bool, bool) {
	selectDbRegex, _ := regexp.Compile(`^(select database\(\))`)
	regex, _ := regexp.Compile("^(select )")
	query := strings.TrimSpace(string(req.Bytes()[5:len(req.Bytes())]))
	queriedNewDb := selectDbRegex.MatchString(strings.ToLower(query))
	isSelect := regex.MatchString(strings.ToLower(query))

	if isSelect {
		slave := GetCluster().SelectSlave()
		log.Printf("[%s][%s] Executing '%s'\n", slave.hostType, fmt.Sprintf("%s:%d", slave.host, slave.port), query)

		// Send client request to slave
		result := slave.HandleQuery(query, p.database)
		// Send slave response to client
		p.proxy.readWriter.Write(bytes.NewBuffer(result), p.client)
	} else {
		master := GetCluster().Master
		log.Printf("[%s][%s] Executing '%s'\n", master.hostType, fmt.Sprintf("%s:%d", master.host, master.port), query)

		// Send client request to server
		p.proxy.readWriter.Write(req, p.mysql)
	}

	return queriedNewDb, isSelect
}

func (p *proxyInstance) setMysqlServerBusyStatus(busy bool) {
	p.mysqlLock.Lock()
	p.mysqlHasPendingQuery = busy
	p.mysqlLock.Unlock()
}

func (p *proxyInstance) queueMysqlResponse() {
	p.mysqlLock.Lock()
	if !p.mysqlHasPendingQuery {
		p.mysqlWriterChan <- 1
	}
	p.mysqlLock.Unlock()
}

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
