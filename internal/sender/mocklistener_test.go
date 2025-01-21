// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package sender

import (
	"fmt"
	"net"
	"time"

	"go.nanomsg.org/mangos/v3"
	"go.nanomsg.org/mangos/v3/protocol"
	"go.nanomsg.org/mangos/v3/protocol/pull"
)

type mockListener struct {
	url      string
	deadline time.Duration
	sock     protocol.Socket
}

func (m *mockListener) Listen() error {
	if m.deadline == 0 {
		m.deadline = 100 * time.Millisecond
	}

	sock, err := pull.NewSocket()
	if err != nil {
		return err
	}

	err = sock.SetOption(mangos.OptionRecvDeadline, m.deadline)
	if err != nil {
		return err
	}

	var url string
	if m.url == "" {
		url, err = findOpenPort()
		if err != nil {
			return err
		}
	} else {
		url = m.url
	}

	if err = sock.Listen(url); err != nil {
		return err
	}

	m.url = url
	m.sock = sock
	return nil
}

func (m *mockListener) Close() error {
	if m.sock != nil {
		return m.sock.Close()
	}
	return nil
}

// findOpenPort finds an open port for listening on.
func findOpenPort() (string, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	defer listener.Close()

	addr := listener.Addr().(*net.TCPAddr)
	return fmt.Sprintf("tcp://127.0.0.1:%d", addr.Port), nil
}
