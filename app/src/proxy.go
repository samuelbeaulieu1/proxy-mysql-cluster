package main

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"regexp"
	"strings"
)

type proxy struct {
	port       string
	readWriter ReaderWriter

	nullDbPacket []byte
}

var proxyInstance *proxy

func GetProxy() *proxy {
	return proxyInstance
}

func StartProxy(port string, readWriter ReaderWriter) error {
	proxyInstance = &proxy{
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

		go proxyInstance.Handle(conn)
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
	db := p.getDatabase(mysql)

	p.handleProxyLoop(db, conn, mysql)
}

func (p *proxy) handleProxyLoop(db string, conn net.Conn, mysql net.Conn) {
	queriedNewDb := false
	for {
		// Client request
		buf, err := p.readWriter.Read(conn)
		if err != nil {
			break
		}
		selectDbRegex, _ := regexp.Compile(`^(select database\(\))`)
		regex, _ := regexp.Compile("^(select )")
		query := strings.TrimSpace(string(buf.Bytes()[5:len(buf.Bytes())]))
		queriedNewDb = selectDbRegex.MatchString(strings.ToLower(query))
		isSelect := regex.MatchString(strings.ToLower(query))
		if isSelect {
			// Send client request to slave
			slave := GetCluster().SelectSlave()
			result := slave.HandleQuery(query, db)

			// Send slave response to client
			p.readWriter.Write(bytes.NewBuffer(result), conn)
		} else {
			// Send client request to server
			p.readWriter.Write(buf, mysql)
			// Send server response to client
			p.readWriter.ReadWrite(mysql, conn)
		}
		if queriedNewDb {
			// Write client response for DB change
			p.readWriter.ReadWrite(conn, mysql)
			// Write to client new DB change
			p.readWriter.ReadWrite(mysql, conn)
			db = p.getDatabase(mysql)
		}
	}
}

func (p *proxy) getDatabase(mysql net.Conn) string {
	if p.nullDbPacket == nil {
		p.nullDbPacket = GetCluster().Master.HandleQuery("SELECT DATABASE()", "")
	}
	p.readWriter.ExecQuery(mysql, "SELECT DATABASE()")
	res, _ := p.readWriter.Read(mysql)
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
