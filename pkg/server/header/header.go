package header

type MessageHeader struct {
	PacketId uint64
}

type MessageHeaderFromServer struct {
	MessageHeader
}

type MessageHeaderFromClient struct {
	MessageHeader
}

// Version for connection handshake
type Version struct {
	Major uint16
	Minor uint16
	Patch uint16
}

type Handshake struct {
	Version Version
}

var DefaultVersion = Version{
	Major: 1,
	Minor: 0,
	Patch: 0,
}
