package serverclient

import (
	"bytes"
	"encoding/binary"
	error2 "github.com/raspi/jumiks/pkg/server/error"
	"github.com/raspi/jumiks/pkg/server/header"
	"math/rand"
	"net"
	"sync"
	"time"
)

// ServerClient is a client connected to a server
// it is used internally to track connected client
type ServerClient struct {
	conn                 *net.UnixConn
	connectedAt          time.Time
	lastPacket           uint64 // Last packet ID processed
	uuid                 uint64 // UUID for this client
	writeMessages        chan []byte
	errors               chan error2.Error
	tooSlowPacketsBehind uint64 // How many packets client can lag behind
	lock                 sync.Mutex
}

func NewClient(conn *net.UnixConn, errors chan error2.Error, tooSlowPacketsBehind uint64) (c *ServerClient) {
	c = &ServerClient{
		conn:                 conn,
		connectedAt:          time.Now(),
		uuid:                 rand.Uint64(),
		errors:               errors,
		writeMessages:        make(chan []byte),
		lastPacket:           0,
		tooSlowPacketsBehind: tooSlowPacketsBehind,
		lock:                 sync.Mutex{},
	}

	return c
}

func (c *ServerClient) Listen() {
	defer c.conn.Close()

	buffer := make([]byte, 1048576)

	for {
		rb, err := c.conn.Read(buffer)
		if err != nil {
			c.errors <- error2.New(err)
			break
		}

		if rb == 0 {
			continue
		}

		buf := bytes.NewBuffer(buffer[:rb])

		var hdr header.MessageHeaderFromClient
		err = binary.Read(buf, binary.LittleEndian, &hdr)
		if err != nil {
			c.errors <- error2.New(err)
			break
		}
		buf.Reset()

		packetsBehind := uint64(0)

		if c.lastPacket != 0 && c.lastPacket != hdr.PacketId {
			c.lock.Lock()
			packetsBehind = c.lastPacket - hdr.PacketId
			c.lock.Unlock()
		}

		if packetsBehind > c.tooSlowPacketsBehind {
			// We are too slow
			break
		}

	}
}

func (c *ServerClient) GetId() uint64 {
	return c.uuid
}

func (c *ServerClient) Write(b []byte) (int, error) {
	return c.conn.Write(b)
}

func (c *ServerClient) Close() error {
	return c.conn.Close()
}

// SetPacketId sets Packet ID from the server side for tracking too slow processing
func (c *ServerClient) SetPacketId(pid uint64) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.lastPacket = pid
}
