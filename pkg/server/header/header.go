package header

// MessageHeader has common fields for both server and client
type MessageHeader struct {
	PacketId uint64
}

// MessageHeaderFromServer contains header
type MessageHeaderFromServer struct {
	MessageHeader
}

// MessageHeaderFromClient contains header
type MessageHeaderFromClient struct {
	MessageHeader
}

// Version for connection handshake
type Version struct {
	Major uint16
	Minor uint16
	Patch uint16
}

// Handshake is first sent by server and then client to see that both speak the same protocol
type Handshake struct {
	Version Version
}

// DefaultVersion sets the protocol's default version
var DefaultVersion = Version{
	Major: 1,
	Minor: 0,
	Patch: 0,
}
