// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package sender

import (
	"errors"
	"time"
)

type Option interface {
	apply(*Sender) error
}

type errOptionFunc func(*Sender) error

func (f errOptionFunc) apply(a *Sender) error {
	return f(a)
}

func optionFunc(f func(*Sender)) errOptionFunc {
	return errOptionFunc(func(c *Sender) error {
		f(c)
		return nil
	})
}

// WithURL sets the target URL for the connection.  This option is required.
func WithURL(url string) Option {
	return optionFunc(func(c *Sender) {
		c.url = url
	})
}

// WithSendTimeout sets the timeout for sending messages.
func WithSendTimeout(timeout time.Duration) Option {
	return optionFunc(func(c *Sender) {
		if 0 < timeout {
			c.sendDeadline = timeout
		}
	})
}

// WithCloseListener sets the function to call when the connection is closed.
// If cancel is provided, it will be populated with a function that can be used
// to remove the listener.
func WithCloseListener(f func(error), cancel ...*func()) Option {
	return optionFunc(func(c *Sender) {
		cancelFn := c.onClose.Add(f)

		for i := range cancel {
			if cancel[i] != nil {
				*cancel[i] = cancelFn
			}
		}
	})
}

// -- Only Validators Below ----------------------------------------------------
func validate() Option {
	return errOptionFunc(func(c *Sender) error {
		if c.url == "" {
			return errors.New("url is required")
		}

		return nil
	})
}
