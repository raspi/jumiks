package serverclient

import (
	"encoding/binary"
	error2 "github.com/raspi/jumiks/pkg/server/error"
	"github.com/raspi/jumiks/pkg/server/header"
	"log"
	"math/rand"
	"net"
	"os"
	"time"
)

// ServerClient is a client connected to a server
// it is used internally to track connected client
type ServerClient struct {
	conn          *net.UnixConn
	connectedAt   time.Time
	lastPacket    uint64
	uuid          uint64
	writeMessages chan []byte
	errors        chan error2.Error
	logger        *log.Logger
}

func NewClient(conn *net.UnixConn, errors chan error2.Error) (c *ServerClient) {
	c = &ServerClient{
		logger:        log.New(os.Stdout, `client `, log.LstdFlags),
		conn:          conn,
		connectedAt:   time.Now(),
		uuid:          rand.Uint64(),
		errors:        errors,
		writeMessages: make(chan []byte),
		lastPacket:    0,
	}

	return c
}

func (c *ServerClient) Listen() {
	defer c.conn.Close()

	for {
		var hdr header.MessageHeaderFromClient
		err := binary.Read(c.conn, binary.LittleEndian, &hdr)
		if err != nil {
			c.errors <- error2.New(err)
			break
		}

		// Track packet ID
		c.lastPacket = hdr.PacketId
	}
}

func (c *ServerClient) GetId() uint64 {
	return c.uuid
}

func (c *ServerClient) Write(b []byte) (int, error) {
	return c.conn.Write(b)
}
