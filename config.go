package wsc

import (
	"crypto/tls"
	"time"
)

// Config contains configuration for the webbsocket.
type Config struct {
	WriteWait         time.Duration
	PongWait          time.Duration
	PingPeriod        time.Duration
	TLSConfig         *tls.Config
	ReadBufferSize    int
	ReadChanSize      int
	WriteBufferSize   int
	WriteChanSize     int
	EnableCompression bool
}
