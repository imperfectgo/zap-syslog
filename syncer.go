// Copyright (c) 2017 Timon Wong
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

// +build !windows,!nacl,!plan9

package zapsyslog

import (
	"net"

	"go.uber.org/zap/zapcore"
)

var (
	_ zapcore.WriteSyncer = &connSyncer{}
)

type connSyncer struct {
	network string
	raddr   string

	conn net.Conn
}

// NewConnSyncer returns a new conn sink for syslog.
func NewConnSyncer(network, raddr string) (zapcore.WriteSyncer, error) {
	s := &connSyncer{
		network: network,
		raddr:   raddr,
	}

	err := s.connect()
	if err != nil {
		return nil, err
	}

	return s, nil
}

// connect makes a connection to the syslog server.
func (s *connSyncer) connect() error {
	if s.conn != nil {
		// ignore err from close, it makes sense to continue anyway
		s.conn.Close()
		s.conn = nil
	}

	var c net.Conn
	c, err := net.Dial(s.network, s.raddr)
	if err != nil {
		return err
	}

	s.conn = c
	return nil
}

// Write writes to syslog with retry.
func (s *connSyncer) Write(p []byte) (n int, err error) {
	if s.conn != nil {
		if n, err := s.conn.Write(p); err == nil {
			return n, err
		}
	}
	if err := s.connect(); err != nil {
		return 0, err
	}

	return s.conn.Write(p)
}

func (s *connSyncer) Sync() error {
	return nil
}
