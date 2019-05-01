// Copyright 2019 Aporeto Inc.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//     http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
	// is closed.
	//
	// The content will be nil for clean disconnection or
	// the error that caused the disconnection. If nothing pumps the
	// Done() channel, the error will be discarded.
	Done() chan error

	// Close closes the websocket.
	//
	// Closing the websocket a second time has no effect.
	// A closed Websocket cannot be reused.
	Close(code int)

	// Error returns a channel that will return errors like
	// read or write discards and other errors that are not
	// terminating the connection.
	//
	// If nothing pumps the Error() channel, the error will be discarded.
	Error() chan error
}
