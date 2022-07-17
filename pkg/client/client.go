package client

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/raspi/jumiks/pkg/server"
	"github.com/raspi/jumiks/pkg/server/header"
	"io"
	"net"
)

type Client struct {
	conn    *net.UnixConn
	hfn     HandlerFunc
	errorch chan error
}

type HandlerFunc func(b []byte)

// New creates a client which is connected to server.Server
func New(name string, handler HandlerFunc, errors chan error) (c *Client, e error) {
	addr, err := net.ResolveUnixAddr(server.ConnType, name)
	if err != nil {
		return nil, err
	}

	conn, err := net.DialUnix(addr.Network(), nil, addr)
	if err != nil {
		return nil, err
	}

	if err = handshake(conn); err != nil {
		return nil, err
	}

	c = &Client{
		conn:    conn,
		hfn:     handler,
		errorch: errors,
	}

	return c, nil
}

func (c *Client) Listen() {
	defer c.conn.Close()

	buffer := make([]byte, 1048576)

	for {
		// Blocking on read
		rb, err := c.conn.Read(buffer)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			c.errorch <- err
			continue
		}

		if rb == 0 {
			continue
		}

		// Read header
		buf := bytes.NewBuffer(buffer[:rb])
		var sheader header.MessageHeaderFromServer

		err = binary.Read(buf, binary.LittleEndian, &sheader)
		if err != nil {
			c.errorch <- err
			break
		}

		// handle message
		c.handleMsg(buf.Bytes())
		buf.Reset()

		// Send acknowledged header
		var sndbuf bytes.Buffer

		ackheader := header.MessageHeaderFromClient{
			MessageHeader: header.MessageHeader{
				PacketId: sheader.PacketId, // processed packet ID
			},
		}

		err = binary.Write(&sndbuf, binary.LittleEndian, ackheader)
		if err != nil {
			c.errorch <- err
			break
		}

		wb, err := c.conn.Write(sndbuf.Bytes())
		if err != nil {
			c.errorch <- err
			break
		}

		if wb == 0 {
			err = c.conn.Close()
			if err != nil {
				c.errorch <- err
				return
			}

			return
		}

		if wb != sndbuf.Len() {
			err = c.conn.Close()
			if err != nil {
				c.errorch <- err
			}

			return
		}
	}

}

// handleMsg sends received message to end-user
func (c *Client) handleMsg(b []byte) {
	c.hfn(b)
}

// handshake determines if both server.Server and Client are speaking the same protocol
func handshake(conn *net.UnixConn) (err error) {
	var serverHs header.Handshake
	err = binary.Read(conn, binary.LittleEndian, &serverHs)
	if err != nil {
		return err
	}

	shake := header.Handshake{Version: header.DefaultVersion}
	err = binary.Write(conn, binary.LittleEndian, shake)
	if err != nil {
		return err
	}

	if serverHs.Version.Major != shake.Version.Major {
		return fmt.Errorf(`ver mismatch, major`)
	}

	if serverHs.Version.Minor != shake.Version.Minor {
		return fmt.Errorf(`ver mismatch, minor`)
	}

	return nil
}
