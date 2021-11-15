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

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// various error messages.
var (
	ErrWriteMessageDiscarded = fmt.Errorf("write chan full: one or more messages has not been sent")
	ErrReadMessageDiscarded  = fmt.Errorf("read chan full: one or more messages has not been received")
)

// WSConnection is the interface that must be implemented
// as a websocket. github.com/gorilla/websocket implements
// this interface.
type WSConnection interface {
	SetReadDeadline(time.Time) error
	SetWriteDeadline(time.Time) error
	SetCloseHandler(func(code int, text string) error)
	SetPingHandler(func(string) error)
	SetPongHandler(func(string) error)
	ReadMessage() (int, []byte, error)
	WriteMessage(int, []byte) error
	WriteControl(int, []byte, time.Time) error
	Close() error
}

type ws struct {
	conn        WSConnection
	readChan    chan []byte
	writeChan   chan []byte
	doneChan    chan error
	errChan     chan error
	cancel      context.CancelFunc
	closeCodeCh chan int
	config      Config
}

// Connect connects to the url and returns a Websocket.
func Connect(ctx context.Context, url string, config Config) (Websocket, *http.Response, error) {

	dialer := &websocket.Dialer{
		Proxy:             http.ProxyFromEnvironment,
		TLSClientConfig:   config.TLSConfig,
		ReadBufferSize:    config.ReadBufferSize,
		WriteBufferSize:   config.WriteBufferSize,
		EnableCompression: config.EnableCompression,
	}

	conn, resp, err := dialer.Dial(url, config.Headers)
	if err != nil {
		return nil, resp, err
	}

	s, err := Accept(ctx, conn, config)

	return s, resp, err
}

// Accept handles an already connect *websocket.Conn and returns a Websocket.
func Accept(ctx context.Context, conn WSConnection, config Config) (Websocket, error) {

	if config.PongWait == 0 {
		config.PongWait = 30 * time.Second
	}
	if config.WriteWait == 0 {
		config.WriteWait = 10 * time.Second
	}
	if config.PingPeriod == 0 {
		config.PingPeriod = 15 * time.Second
	}
	if config.WriteChanSize == 0 {
		config.WriteChanSize = 64
	}
	if config.ReadChanSize == 0 {
		config.ReadChanSize = 64
	}

	if err := conn.SetReadDeadline(time.Now().Add(config.PongWait)); err != nil {
		return nil, err
	}

	subCtx, cancel := context.WithCancel(ctx)

	s := &ws{
		conn:        conn,
		readChan:    make(chan []byte, config.ReadChanSize),
		writeChan:   make(chan []byte, config.WriteChanSize),
		doneChan:    make(chan error, 2),
		errChan:     make(chan error, 10),
		closeCodeCh: make(chan int, 1),
		cancel:      cancel,
		config:      config,
	}

	s.conn.SetCloseHandler(func(code int, text string) error {
		s.cancel()
		return nil
	})

	s.conn.SetPongHandler(func(string) error {
		return s.conn.SetReadDeadline(time.Now().Add(s.config.PongWait))
	})

	s.conn.SetPingHandler(func(string) error {
		return s.conn.SetReadDeadline(time.Now().Add(s.config.PongWait))
	})

	go s.readPump()
	go s.writePump(subCtx)

	return s, nil
}

// Write is part of the the Websocket interface implementation.
func (s *ws) Write(data []byte) {

	select {
	case s.writeChan <- data:
	default:
		s.error(ErrWriteMessageDiscarded)
	}
}

// Read is part of the the Websocket interface implementation.
func (s *ws) Read() chan []byte {

	return s.readChan
}

// Error is part of the the Websocket interface implementation.
func (s *ws) Error() chan error {

	return s.errChan
}

// Done is part of the the Websocket interface implementation.
func (s *ws) Done() chan error {

	return s.doneChan
}

// Close is part of the the Websocket interface implementation.
func (s *ws) Close(code int) {

	if code != 0 {
		select {
		case s.closeCodeCh <- code:
		default:
		}
	}

	s.cancel()
}

func (s *ws) readPump() {

	var err error
	var msg []byte
	var msgType int

	for {
		if msgType, msg, err = s.conn.ReadMessage(); err != nil {
			s.done(err)
			return
		}

		switch msgType {

		case websocket.TextMessage, websocket.BinaryMessage:
			select {
			case s.readChan <- msg:
			default:
				s.error(ErrReadMessageDiscarded)
			}

		case websocket.CloseMessage:
			return
		}
	}
}

func (s *ws) writePump(ctx context.Context) {

	var err error

	ticker := time.NewTicker(s.config.PingPeriod)
	defer ticker.Stop()

	for {
		select {

		case message := <-s.writeChan:

			_ = s.conn.SetWriteDeadline(time.Now().Add(s.config.WriteWait)) // nolint: errcheck
			if err = s.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				s.done(err)
				return
			}

		case <-ticker.C:

			_ = s.conn.SetWriteDeadline(time.Now().Add(s.config.WriteWait)) // nolint: errcheck
			if err = s.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				s.done(err)
				return
			}

		case <-ctx.Done():

			code := websocket.CloseGoingAway
			select {
			case code = <-s.closeCodeCh:
			default:
			}

			enc := make([]byte, 2)
			binary.BigEndian.PutUint16(enc, uint16(code))

			s.done(
				s.conn.WriteControl(
					websocket.CloseMessage,
					enc,
					time.Now().Add(1*time.Second),
				),
			)

			_ = s.conn.Close()

			return
		}
	}
}

func (s *ws) done(err error) {

	select {
	case s.doneChan <- err:
	default:
		if err != nil {
			log.Printf("wsc: error: unable to send done message: error: %s\n", err)
		}
	}
}

func (s *ws) error(err error) {

	select {
	case s.errChan <- err:
	default:
		if err != nil {
			log.Printf("wsc: error: unable to send error: %s\n", err)
		}
	}
}
