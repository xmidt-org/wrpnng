// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package receiver

import (
	"errors"
	"time"

	"github.com/xmidt-org/wrp-go/v3"
)

// Option is a functional option for configuring a Receiver.
type Option interface {
	apply(*Receiver) error
}

type errOptionFunc func(*Receiver) error

func (f errOptionFunc) apply(a *Receiver) error {
	return f(a)
}

func optionFunc(f func(*Receiver)) errOptionFunc {
	return errOptionFunc(func(c *Receiver) error {
		f(c)
		return nil
	})
}

// WithURL sets the URL for the Receiver.
func WithURL(url string) Option {
	return optionFunc(func(r *Receiver) {
		r.url = url
	})
}

// WithRecvTimeout sets the receiving timeout for the Receiver.
func WithRecvTimeout(timeout time.Duration) Option {
	return optionFunc(func(r *Receiver) {
		if timeout >= 0 {
			r.timeout = timeout
		}
	})
}

// WithModifyWRP adds a WRP message handler for the Receiver, with an optional
// cancel function parameter.
//
//   - There can be multiple handlers.
//   - The order of the handlers is not guaranteed.
//   - The returned value of the wrp.Modifier is ignored.
//   - The handlers are called on a separate goroutine, so they do not block the
//     Receiver, but can impact other handlers.
func WithModifyWRP(m wrp.Modifier, cancel ...*func()) Option {
	return optionFunc(func(r *Receiver) {
		cancelFn := r.onMsg.Add(m)
		for i := range cancel {
			if cancel[i] != nil {
				*cancel[i] = cancelFn
			}
		}
	})
}

// WithCloseListener adds a listener for when the Receiver closes, with an
// optional cancel function parameter.
//
//   - There can be multiple listeners.
//   - The order of the listeners is not guaranteed.
//   - The error parameter is the reason for the close.
//   - The listeners are called on a separate goroutine, so they do not block
//     the Receiver, but can impact other listeners.
func WithCloseListener(f func(error), cancel ...*func()) Option {
	return optionFunc(func(r *Receiver) {
		cancelFn := r.onFailure.Add(f)
		for i := range cancel {
			if cancel[i] != nil {
				*cancel[i] = cancelFn
			}
		}
	})
}

func validate() Option {
	return errOptionFunc(func(r *Receiver) error {
		if r.url == "" {
			return errors.New("url is required")
		}
		return nil
	})
}
