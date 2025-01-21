// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package wrpnng

import (
	"context"
	"fmt"
	"net"

	"github.com/xmidt-org/eventor"
	"github.com/xmidt-org/wrp-go/v3"
	"github.com/xmidt-org/wrpnng/internal/receiver"
	"github.com/xmidt-org/wrpnng/internal/sender"
)

// Client is a WRP <-> nanomsg client.  The client is responsible for sending
// messages to the network and receiving messages from the network.  It also
// handles the registration message and sends heartbeats at regular intervals.
type Client struct {
	clientURL string
	serverURL string

	rOpts []receiver.Option
	r     *receiver.Receiver

	sOpts []sender.Option
	s     *sender.Sender

	egress eventor.Eventor[wrp.Modifier]
}

// NewClient creates a new client.  The client is not started until Start is
// called.
func NewClient(opts ...ClientOption) (*Client, error) {
	var client Client

	defaults := []ClientOption{}

	vadors := []ClientOption{
		determineClientURL(),
		validateClient(),
	}

	opts = append(defaults, opts...)
	opts = append(opts, vadors...)

	for _, opt := range opts {
		if opt != nil {
			if err := opt.apply(&client); err != nil {
				return nil, err
			}
		}
	}

	return &Client{}, nil
}

// Start starts the client.  This call is idempotent.
func (c *Client) Start() error {
	return nil
}

// Stop stops the client.  This call is idempotent.
func (c *Client) Stop() error {
	return nil
}

// ProcessWRP is called when a message should be sent to the network.
func (c *Client) ProcessWRP(ctx context.Context, msg wrp.Message) error {
	return nil
}

func findOpenURL() (string, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	defer listener.Close()

	addr := listener.Addr().(*net.TCPAddr)
	return fmt.Sprintf("tcp://127.0.0.1:%d", addr.Port), nil
}
