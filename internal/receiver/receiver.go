// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package receiver

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/xmidt-org/eventor"
	"github.com/xmidt-org/wrp-go/v3"
	"go.nanomsg.org/mangos/v3"
	"go.nanomsg.org/mangos/v3/protocol/pull"
)

// Receiver is a simple listener for incoming messages.  It is safe for concurrent
// use.
type Receiver struct {
	url       string
	timeout   time.Duration
	onMsg     eventor.Eventor[wrp.Modifier]
	onFailure eventor.Eventor[func(error)]
	wg        sync.WaitGroup
	lock      sync.Mutex
	cancel    context.CancelFunc
}

// New creates a new Receiver.  The receiver is not started until Start is called.
func New(opts ...Option) (*Receiver, error) {
	r := &Receiver{}

	opts = append(opts, validate())

	for _, opt := range opts {
		if err := opt.apply(r); err != nil {
			return nil, err
		}
	}

	return r, nil
}

// Listen begins listening for messages.  It is safe to call Listen multiple times,
// and will restart the receiver if it was previously stopped.
func (r *Receiver) Listen() error {
	r.lock.Lock()
	defer r.lock.Unlock()

	// If it already has a cancel function, it's already running.
	if r.cancel != nil {
		return nil
	}

	sock, err := newSocket(r.url, r.timeout)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())

	r.cancel = cancel

	r.wg.Add(1)
	go r.wrapper(ctx, sock)

	fmt.Println("Listening...")
	return nil
}

// Close halts the receiver.  It is safe to call Close multiple times.
func (r *Receiver) Close() error {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.cancel != nil {
		r.cancel()
		r.cancel = nil
		r.wg.Wait()
	}
	return nil
}

func newSocket(url string, timeout time.Duration) (mangos.Socket, error) {
	// These checks are extremely defensive, and unless the upstream code changes
	// the normal flow of execution, they should never happen.
	sock, err := pull.NewSocket()
	if err == nil {
		// Use SetOption to set the receive deadline.  The other ways to set the
		// receive deadline don't seem to work.
		err = sock.SetOption(mangos.OptionRecvDeadline, timeout)
		if err == nil {
			err = sock.Listen(url)
			if err == nil {
				return sock, nil
			}
		}
	}

	return nil, err
}

// wrapper is a helper function that wraps the receive function.  It is used to
// handle the context and timeouts correctly, and to call the closure/failure
// handlers.
func (r *Receiver) wrapper(ctx context.Context, sock mangos.Socket) {
	err := r.receive(ctx, sock)

	r.Close()

	r.onFailure.Visit(func(f func(error)) {
		f(err)
	})
}

// receive is the main loop for the receiver.  It listens for messages and
// forwards them to the registered handlers.
//
// The code is a bit more involved to handle the context and timeouts correctly.
// The mangos library doesn't support context, so we have to handle it ourselves.
func (r *Receiver) receive(ctx context.Context, sock mangos.Socket) error {
	defer r.wg.Done()

	for {
		// Use a separate goroutine to receive from the socket
		recvChan := make(chan []byte, 1)
		errChan := make(chan error, 1)

		r.wg.Add(1)
		go func() {
			defer r.wg.Done()

			bytes, err := sock.Recv()
			if err != nil {
				errChan <- err
			} else {
				fmt.Println("got a message")
				recvChan <- bytes
			}
		}()

		var buf []byte
		var err error

		select {
		case <-ctx.Done():
			err = ctx.Err()
		case err = <-errChan:
		case buf = <-recvChan:
		}

		if buf != nil {
			var msg wrp.Message
			fmt.Println("decoding message")
			if err := wrp.NewDecoderBytes(buf, wrp.Msgpack).Decode(&msg); err == nil {
				// We got a message.  Tell everyone, but we don't care what they
				// do with it.  Do it in a separate goroutine so we don't block
				// the receiver.
				go func() {
					fmt.Println("sending it to the observers")
					r.onMsg.Visit(func(m wrp.Modifier) {
						_, _ = m.ModifyWRP(context.Background(), msg)
					})
				}()
			} else {
				fmt.Println("failed to decode message")
			}

			// If we get any error processing the message, we ignore the error
			// and keep going.
			continue
		}

		// Timeouts are ok, keep going.
		if errors.Is(err, mangos.ErrRecvTimeout) {
			continue
		}

		_ = sock.Close()

		// If the context was canceled, return that error, too.
		return errors.Join(err, ctx.Err())
	}
}
