package protocol

const (
	// ProtocolMajor is the current major protocol version.
	ProtocolMajor byte = 1
	// ProtocolMinor is the current minor protocol version.
	ProtocolMinor byte = 0

	// OpHello is sent by the server as the first frame after connection.
	// Format: [0x00][1B major][1B minor]
	OpHello byte = 0x00

	// OpClientHello is the client's response to the server hello.
	// Format: [0x12][1B major][1B minor]
	OpClientHello byte = 0x12
)

// HelloFrame represents a protocol version handshake message.
type HelloFrame struct {
	Major byte
	Minor byte
}
