// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package websocket

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"net/url"
)

// DialError is an error that occurs while dialling a websocket server.
type DialError struct {
	*Config
	Err error
}

func (e *DialError) Error() string {
	return "websocket.Dial " + e.Config.Location.String() + ": " + e.Err.Error()
}

// NewConfig creates a new WebSocket config for client connection.
func NewConfig(server, origin string) (config *Config, err error) {
	config = new(Config)
	config.Version = ProtocolVersionHybi13
	config.Location, err = url.ParseRequestURI(server)
	if err != nil {
		return
	}
	config.Origin, err = url.ParseRequestURI(origin)
	if err != nil {
		return
	}
	config.Header = http.Header(make(map[string][]string))
	return
}

func ClientHandshake(config *Config, rw *bufio.ReadWriter) error {
	return hybiClientHandshake(config, rw.Reader, rw.Writer)
}

func NewClientConn(rwc io.ReadWriteCloser, buf *bufio.ReadWriter, c *Config) *Conn {
	return newHybiClientConn(c, buf, rwc)
}

// Dial opens a new client connection to a WebSocket.
func Dial(url_, protocol, origin string) (ws *Conn, err error) {
	config, err := NewConfig(url_, origin)
	if err != nil {
		return nil, err
	}
	if protocol != "" {
		config.Protocol = []string{protocol}
	}
	return DialConfig(config)
}

var portMap = map[string]string{
	"ws":  "80",
	"wss": "443",
}

func parseAuthority(location *url.URL) string {
	if _, ok := portMap[location.Scheme]; ok {
		if _, _, err := net.SplitHostPort(location.Host); err != nil {
			return net.JoinHostPort(location.Host, portMap[location.Scheme])
		}
	}
	return location.Host
}

func DialConfigRaw(config *Config) (net.Conn, *bufio.ReadWriter, error) {
	if config.Location == nil {
		return nil, nil, &DialError{config, ErrBadWebSocketLocation}
	}
	if config.Origin == nil {
		return nil, nil, &DialError{config, ErrBadWebSocketOrigin}
	}

	dialer := config.Dialer
	if dialer == nil {
		dialer = &net.Dialer{}
	}
	conn, err := dialWithDialer(dialer, config)
	if err != nil {
		return nil, nil, err
	}

	br := bufio.NewReader(conn)
	bw := bufio.NewWriter(conn)
	err = hybiClientHandshake(config, br, bw)
	if err != nil {
		return nil, nil, err
	}

	return conn, bufio.NewReadWriter(br, bw), nil
}

// DialConfig opens a new client connection to a WebSocket with a config.
func DialConfig(config *Config) (ws *Conn, err error) {
	client, rw, err := DialConfigRaw(config)
	if err != nil {
		return nil, &DialError{config, err}
	}
	return NewClientConn(client, rw, config), nil
}
