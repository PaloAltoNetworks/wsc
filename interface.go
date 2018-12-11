package wsc

// Websocket is the interface of channel based websocket.
type Websocket interface {

	// Reads returns a channel where the incoming messages are published.
	// If nothing pumps the Read() while it is full, new messages will be
	// discarded.
	//
	// You can configure the size of the read chan in Config.
	// The default is 64 messages.
	Read() chan []byte

	// Write writes the given []byte in to the websocket.
	// If the other side of the websocket cannot get all messages
	// while the internal write channel is full, new messages will
	// be discarded.
	//
	// You can configure the size of the write chan in Config.
	// The default is 64 messages.
	Write([]byte)

	// Done returns a channel that will return when the connection
	// if closed.
	//
	// The content will be nil for clean disconnection or
	// the error that caused the disconnection. If nothing pumps the
	// Done() channel, the message will be discarded.
	Done() chan error

	// Close closes the websocket.
	//
	// Closing the websocket a second time has no effect.
	// A closed Websocket cannot be reused.
	Close(code int)
}
