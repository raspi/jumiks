package server

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	error2 "github.com/raspi/jumiks/pkg/server/error"
	"github.com/raspi/jumiks/pkg/server/header"
	"github.com/raspi/jumiks/pkg/server/internal/serverclient"
	"log"
	"net"
	"os"
	"sync"
	"syscall"
)

// ConnType determines unix domain socket communication type
const ConnType = `unixpacket`

// StartPacketId determines the starting packet ID for Server
const StartPacketId = uint64(100000)

type Server struct {
	logger               *log.Logger
	listener             *net.UnixListener                     // Listening unix domain socket
	clients              map[uint64]*serverclient.ServerClient // Connected clients
	packetId             uint64                                // Packet tracking ID
	errch                chan error2.Error                     // Errors
	tooSlowPacketsBehind uint64                                // How many packets can connected client lag behind
	messagesCh           chan []byte                           // messages sent to connected clients
	connectionNew        chan *net.UnixConn
	connectionClose      chan uint64
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
		logger:               log.New(os.Stdout, ``, log.LstdFlags),
		listener:             conn,
		packetId:             StartPacketId,
		errch:                errch,
		tooSlowPacketsBehind: tooSlowPacketsBehind,
		clients:              make(map[uint64]*serverclient.ServerClient),
		connectionNew:        make(chan *net.UnixConn),
		connectionClose:      make(chan uint64),
		messagesCh:           make(chan []byte),
		lock:                 sync.Mutex{},
	}

	return s, nil
}

func (s *Server) listenConnections() {
	defer s.listener.Close()

	for {
		// New connection
		s.logger.Printf(`listening for new connection`)
		conn, err := s.listener.AcceptUnix()
		if err != nil {
			s.errch <- error2.New(err)
			continue
		}
		s.logger.Printf(`new connection, sending handshake`)

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

		s.logger.Printf(`handshake ok`)
		s.connectionNew <- conn
	}
}

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
	go s.listenConnections()

	for {
		select {
		case err := <-s.errch:
			s.logger.Printf(`error: %v`, err)
		case msg := <-s.messagesCh: // new message
			s.logger.Printf(`received message from channel`)
			s.lock.Lock()
			pId := s.packetId
			s.lock.Unlock()

			for clientId, client := range s.clients {
				if client == nil {
					s.logger.Printf(`client is nil!`)
					s.connectionClose <- clientId
					continue
				}

				// Send the buffer to client
				s.logger.Printf(`writing to client`)
				wb, err := client.Write(generateMsg(pId, msg))
				if err != nil {
					if errors.Is(err, syscall.EPIPE) {
						s.connectionClose <- clientId
						continue
					}

					panic(err)
				}

				if wb == 0 {
					panic(`no bytes written`)
				}
			}

		case conn := <-s.connectionNew:
			s.logger.Printf(`adding client`)
			c := serverclient.NewClient(conn, s.errch)
			go c.Listen()
			s.clients[c.GetId()] = c
			s.logger.Printf(`client added`)

		case cId := <-s.connectionClose:
			s.logger.Printf(`!!!! disconnecting client #%v`, cId)

			/*
				err := s.clients[cId].Close()
				if err != nil {
					s.errch <- error2.New(err)
				}

			*/

			delete(s.clients, cId)
		default:

		}
	}
}

// SendToAll sends a message to every connected client
func (s *Server) SendToAll(msg []byte) {
	if len(msg) > 0 {
		s.messagesCh <- msg
	}

	s.lock.Lock()
	s.packetId++
	s.lock.Unlock()
}
