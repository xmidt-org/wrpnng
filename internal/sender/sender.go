// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package sender

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/xmidt-org/eventor"
	"github.com/xmidt-org/wrp-go/v3"
	"go.nanomsg.org/mangos/v3"
	"go.nanomsg.org/mangos/v3/protocol"
	"go.nanomsg.org/mangos/v3/protocol/push"

	// register transports
	_ "go.nanomsg.org/mangos/v3/transport/tcp"
)

var (
	ErrConnClosed   = errors.New("connection closed")
	ErrFailedToSend = errors.New("failed to send message")
)

// Sender is a simple connection to an external service.  It is safe for concurrent
// use.
type Sender struct {
	url          string
	onClose      eventor.Eventor[func(error)]
	lock         sync.Mutex
	sock         protocol.Socket
	sendDeadline time.Duration
}

// New creates a new Sender.  The Sender is not connected to the remote service
// until Dial() is called.  The Sender is safe for concurrent use.  The option
// WithURL is required.
func New(opts ...Option) (*Sender, error) {
	var s Sender

	vadors := []Option{
		validate(),
	}

	opts = append(opts, vadors...)

	for _, opt := range opts {
		if opt != nil {
			if err := opt.apply(&s); err != nil {
				return nil, err
			}
		}
	}

	return &s, nil
}

// Dial connects the Sender to the remote service.  This method is idempotent.
func (s *Sender) Dial() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.sock != nil {
		return nil
	}

	sock, err := dialNewSocket(s.url, s.sendDeadline)
	if err != nil {
		return err
	}

	s.sock = sock

	return nil
}

// dialNewSocket is a helper function that creates a new socket and connects it
// to the specified URL.  The deadline parameter is used to set the send timeout
// for the socket.
func dialNewSocket(url string, deadline time.Duration) (mangos.Socket, error) {
	// These checks are extremely defensive, and unless the upstream code changes
	// the normal flow of execution, they should never happen.
	sock, err := push.NewSocket()
	if err == nil {
		// Set the write queue length to 1.  This is the only way to ensure that
		// message delivery faiures are detected
		err = sock.SetOption(mangos.OptionWriteQLen, 1)
		if err == nil {
			// Set the send timeout to the configured value.  The other methods of
			// setting the timeout are not supported by the mangos library
			err = sock.SetOption(mangos.OptionSendDeadline, deadline)
			if err == nil {
				err = sock.Dial(url)
				if err == nil {
					return sock, nil
				}
			}
		}
	}

	return nil, err
}

// Close closes the connection to the remote service.  This method is idempotent.
func (s *Sender) Close() error {
	var trigger bool

	s.lock.Lock()
	if s.sock != nil {
		trigger = true
		_ = s.sock.Close()
		s.sock = nil
	}
	s.lock.Unlock()

	if trigger {
		s.visitOnClose(nil)
	}
	return nil
}

// ProcessWRP sends a WRP message to the remote service.  The context is used to
// set a timeout for the send operation.  If the context is canceled, the send
// operation will fail with a context.Canceled error.  If the connection is closed,
// the send operation will fail with ErrConnClosed.  If the send operation fails
// for any other reason, the error will be wrapped with ErrFailedToSend.
// ProcessWRP will never return wrp.ErrNotHandled.
func (s *Sender) ProcessWRP(ctx context.Context, msg wrp.Message) error {
	if ctx == nil {
		ctx = context.Background()
	}

	var buf []byte
	if err := wrp.NewEncoderBytes(&buf, wrp.Msgpack).Encode(msg); err != nil {
		return err
	}

	s.lock.Lock()
	if s.sock == nil {
		s.lock.Unlock()
		return ErrConnClosed
	}

	rv := make(chan error, 1)

	go func() {
		// Only when we're done sending the message or timing out can we
		// release the lock.  This may be after ProcessWRP() returns, but that's
		// correct.
		err := s.sock.Send(buf)

		if err != nil { // This error is not recoverable.  Close the connection.
			_ = s.sock.Close()
			s.sock = nil

			s.lock.Unlock()

			s.visitOnClose(errors.Join(err, ErrFailedToSend))
			rv <- err
			return
		}

		s.lock.Unlock()

		if ctx.Err() != nil {
			// The context was canceled, but the connection is fine.  Just return
			// the error, but don't close the connection.
			err = ctx.Err()
		}

		rv <- err
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-rv:
		return err
	}
}

// visitOnClose is a helper function that calls all of the functions registered
// with the onClose eventor.
func (s *Sender) visitOnClose(err error) {
	s.onClose.Visit(func(f func(error)) {
		f(err)
	})
}
