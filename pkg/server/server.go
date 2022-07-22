package server

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	error2 "github.com/raspi/jumiks/pkg/server/error"
	"github.com/raspi/jumiks/pkg/server/header"
	"github.com/raspi/jumiks/pkg/server/internal/serverclient"
	"io"
	"net"
	"sync"
	"syscall"
)

// ConnType determines unix domain socket communication type
const ConnType = `unixpacket`

// StartPacketId determines the starting packet ID for Server
const StartPacketId = uint64(100000)

type Server struct {
	listener             *net.UnixListener                     // Listening unix domain socket
	clients              map[uint64]*serverclient.ServerClient // Connected clients
	packetId             uint64                                // Packet tracking ID
	errch                chan error2.Error                     // Errors
	tooSlowPacketsBehind uint64                                // How many packets can connected client lag behind
	messagesCh           chan []byte                           // messages sent to connected clients
	connectionNew        chan *net.UnixConn
	lock                 sync.Mutex
}

// New creates a new unix domain socket server where client.Client can connect
func New(name string, tooSlowPacketsBehind uint64, errch chan error2.Error) (s *Server, err error) {

	if tooSlowPacketsBehind == 0 {
		return nil, fmt.Errorf(`packets behind must be > 0`)
	}

	addr, err := net.ResolveUnixAddr(ConnType, name)
	if err != nil {
		return s, err
	}

	conn, err := net.ListenUnix(addr.Network(), addr)
	if err != nil {
		return s, fmt.Errorf(`couldn't listen %s:%s: %v `, addr.Network(), addr.Name, err)
	}
	conn.SetUnlinkOnClose(true)

	s = &Server{
		listener:             conn,
		packetId:             StartPacketId,
		errch:                errch,
		tooSlowPacketsBehind: tooSlowPacketsBehind,
		clients:              make(map[uint64]*serverclient.ServerClient), // connected clients
		connectionNew:        make(chan *net.UnixConn),
		messagesCh:           make(chan []byte), // Messages sent to all connected clients
		lock:                 sync.Mutex{},      // Lock when global state changes
	}

	return s, nil
}

// listenConnections listens on new connections to this server
func (s *Server) listenConnections() {
	defer s.listener.Close()

	for {
		// New connection
		conn, err := s.listener.AcceptUnix()
		if err != nil {
			s.errch <- error2.New(err)
			continue
		}

		// handshake for determining that client speaks the same protocol

		// Send our handshake
		shake := header.Handshake{
			Version: header.DefaultVersion,
		}
		err = binary.Write(conn, binary.LittleEndian, shake)
		if err != nil {
			conn.Close()
			continue
		}

		// Read client's handshake
		var clientShake header.Handshake
		err = binary.Read(conn, binary.LittleEndian, &clientShake)
		if err != nil {
			conn.Close()
			continue
		}

		if shake.Version.Major != clientShake.Version.Major {
			conn.Close()
			continue
		}

		if shake.Version.Minor != clientShake.Version.Minor {
			conn.Close()
			continue
		}

		s.connectionNew <- conn
	}
}

// generateMsg generates a new message which has a header
func generateMsg(pId uint64, msg []byte) []byte {
	var buf bytes.Buffer

	// Add header to message buffer
	msghdr := header.MessageHeaderFromServer{
		MessageHeader: header.MessageHeader{
			PacketId: pId,
		},
	}

	err := binary.Write(&buf, binary.LittleEndian, msghdr)
	if err != nil {
		return nil
	}

	_, err = buf.Write(msg)
	if err != nil {
		return nil
	}

	return buf.Bytes()
}

func (s *Server) Listen() {
	// Listen on new connections (non-blocking)
	go s.listenConnections()

	for {
		select {
		case msg := <-s.messagesCh: // new message
			s.lock.Lock()
			packetId := s.packetId
			s.lock.Unlock()

			msg = generateMsg(packetId, msg)

			for clientId, client := range s.clients {
				if client == nil {
					delete(s.clients, clientId)
					continue
				}

				client.SetPacketId(packetId)

				// Send the buffer to client
				wb, err := client.Write(msg)
				if err != nil {
					if errors.Is(err, io.EOF) {
						_ = client.Close()
						delete(s.clients, clientId)
						continue
					} else if errors.Is(err, syscall.EPIPE) {
						_ = client.Close()
						delete(s.clients, clientId)
						continue
					} else if errors.Is(err, net.ErrClosed) {
						_ = client.Close()
						delete(s.clients, clientId)
						continue
					}

					// Shouldn't happen
					panic(err)
				}

				if wb == 0 {
					panic(`no bytes written`)
				}

			}

		case conn := <-s.connectionNew: // Add new connection
			c := serverclient.NewClient(conn, s.errch, s.tooSlowPacketsBehind)
			go c.Listen()
			s.clients[c.GetId()] = c
		}
	}
}

// SendToAll sends a message to every connected client
func (s *Server) SendToAll(msg []byte) {
	s.messagesCh <- msg

	s.lock.Lock()
	s.packetId++
	s.lock.Unlock()
}

// Close closes all connected clients and then closes the listener
func (s *Server) Close() error {
	var del []uint64

	for i, c := range s.clients {
		if c != nil {
			err := c.Close()
			if err != nil {
				s.errch <- error2.New(err)
			}
		}

		del = append(del, i)
	}

	// Remove all
	for _, i := range del {
		delete(s.clients, i)
	}

	// Close the main socket listener
	return s.listener.Close()
}
