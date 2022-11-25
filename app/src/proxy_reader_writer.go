package main

import (
	"bytes"
	"net"
)

const (
	queryCommandByte = 3
	packetHeaderSize = 4
)

type Reader interface {
	Read(src net.Conn) (*bytes.Buffer, error)
}

type Writer interface {
	Write(buf *bytes.Buffer, dst net.Conn) error
	ExecQuery(dst net.Conn, query string)
}

type ReaderWriter interface {
	Reader
	Writer
	ReadWrite(src net.Conn, dst net.Conn) (*bytes.Buffer, error)
}

type ProxyReader struct{}
type ProxyWriter struct{}

type ProxyReaderWriter struct {
	ProxyReader
	ProxyWriter
}

func (r *ProxyReader) Read(src net.Conn) (*bytes.Buffer, error) {
	buf := &bytes.Buffer{}
	data := make([]byte, 8192)
	n, err := src.Read(data)
	if err != nil {
		return nil, err
	}
	buf.Write(data[:n])

	return buf, nil
}

func (w *ProxyWriter) Write(buf *bytes.Buffer, dst net.Conn) error {
	_, err := dst.Write(buf.Bytes())

	return err
}

func (w *ProxyWriter) ExecQuery(dst net.Conn, query string) {
	// Query length + 1 byte for command type
	length := 1 + len(query)
	// Packet data with query length + 1 byte command type + header size
	data := make([]byte, length+packetHeaderSize)

	// Inserting the query in the data, skipping the first 5 bytes for header and command
	copy(data[5:], query)

	// Writing 3 bytes Protocol::FixedLengthInteger MySQL (int3store from MySQL source code)
	data[0] = byte(length)
	data[1] = byte(length >> 8)
	data[2] = byte(length >> 16)
	// Sequence ID, 0 for new query
	data[3] = 0
	data[4] = queryCommandByte

	dst.Write(data)
}

func (rw *ProxyReaderWriter) ReadWrite(src net.Conn, dst net.Conn) (*bytes.Buffer, error) {
	buf, err := rw.Read(src)
	if err != nil {
		return nil, err
	}

	err = rw.Write(buf, dst)

	return buf, err
}
