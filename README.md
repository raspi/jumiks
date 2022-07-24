# jumiks

![GitHub All Releases](https://img.shields.io/github/downloads/raspi/jumiks/total?style=for-the-badge)
![GitHub release (latest by date)](https://img.shields.io/github/v/release/raspi/jumiks?style=for-the-badge)
![GitHub tag (latest by date)](https://img.shields.io/github/v/tag/raspi/jumiks?style=for-the-badge)
[![Go Report Card](https://goreportcard.com/badge/github.com/raspi/jumiks)](https://goreportcard.com/report/github.com/raspi/jumiks)


Local `AF_UNIX` [unix domain socket](https://en.wikipedia.org/wiki/Unix_domain_socket) server and client which uses TCP-style `SOCK_SEQPACKET` (`unixpacket` in Go) packets for sending messages from one server to multiple clients.

See [example](example) directory for example.

## Protocol

The maximum unix domain socket packet size is determined with kernel parameters. 
Server counts packets from the client and user can set a limit to the server. If client gets too far behind the server disconnects that client. This is so that memory can be freed.

### Handshake

When a client connects the server sends version information. Client replies with it's version information. If the version is correct the connection is added to the server's connection list, otherwise the connection is closed.

### Packets

After the handshake the server simply sends what ever bytes it wants to the client. 

The packet first contains the packet ID (uint64) and then the actual data from end-user. Client replies with packet ID after it has handled the packet. 

