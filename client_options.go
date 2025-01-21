// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package wrpnng

import (
	"errors"

	"github.com/xmidt-org/wrp-go/v3"
)

// ClientOption is the interface implemented by types that can be used to
// configure the service.
type ClientOption interface {
	apply(*Client) error
}

type errClientOptionFunc func(*Client) error

func (f errClientOptionFunc) apply(c *Client) error {
	return f(c)
}

func clientOptionFunc(f func(*Client)) errClientOptionFunc {
	return errClientOptionFunc(func(c *Client) error {
		f(c)
		return nil
	})
}

// WithClientURL sets the URL used for connecting to the network server.  This is
// optional.  If not set, the client will attempt automatically to determine the
// URL.
func WithClientURL(url string) ClientOption {
	return clientOptionFunc(func(c *Client) {
		c.clientURL = url
	})
}

// WithServerURL sets the URL used for connecting to the network server.  This is
// required.  The URL should be in the format of "tcp://<ip>:<port>" unless other
// transports are registered.
func WithServerURL(url string) ClientOption {
	return clientOptionFunc(func(c *Client) {
		c.serverURL = url
	})
}

// WithReceivedModifier adds a modifier to the list of modifiers that are informed
// of messages received by the client.  The modifier can change the message, but
// any error returned by the modifier is ignored.
func WithReceivedModifier(modifier wrp.Modifier, cancel ...*func()) ClientOption {
	return clientOptionFunc(func(c *Client) {
		cancelFn := c.egress.Add(modifier)
		for i := range cancel {
			if cancel[i] != nil {
				*cancel[i] = cancelFn
			}
		}
	})
}

//------------------------------------------------------------------------------

func determineClientURL() ClientOption {
	return errClientOptionFunc(func(c *Client) error {
		if c.clientURL != "" {
			return nil
		}

		url, err := findOpenURL()
		if err != nil {
			return err
		}

		c.clientURL = url
		return nil
	})
}

func validateClient() ClientOption {
	return errClientOptionFunc(func(c *Client) error {
		if c.serverURL == "" {
			return errors.New("server URL is required")
		}

		return nil
	})
}
