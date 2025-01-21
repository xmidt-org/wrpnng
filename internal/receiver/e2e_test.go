// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package receiver_test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/wrp-go/v3"
	"github.com/xmidt-org/wrpnng/internal/receiver"
	"go.nanomsg.org/mangos/v3"
	"go.nanomsg.org/mangos/v3/protocol/push"

	// register transports
	_ "go.nanomsg.org/mangos/v3/transport/tcp"
)

func TestEnd2End(t *testing.T) {
	require := require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	port, err := findOpenPort()
	require.NoError(err)
	require.NotZero(port)

	var lock sync.Mutex
	var got []wrp.Message
	wrpRecorder := wrp.ObserverAsModifier(
		wrp.ObserverFunc(
			func(_ context.Context, m wrp.Message) {
				lock.Lock()
				defer lock.Unlock()
				got = append(got, m)
			},
		),
	)

	var closed []error
	closeRecorder := func(err error) {
		lock.Lock()
		defer lock.Unlock()
		closed = append(closed, err)
	}

	var listenerCancelFn, wrpCancelFn func()
	assert.Nil(t, listenerCancelFn)
	assert.Nil(t, wrpCancelFn)

	r, err := receiver.New(
		receiver.WithURL(fmt.Sprintf("tcp://127.0.0.1:%d", port)),
		receiver.WithRecvTimeout(100*time.Millisecond),
		receiver.WithModifyWRP(wrpRecorder, nil, &wrpCancelFn),
		receiver.WithCloseListener(closeRecorder, &listenerCancelFn, nil),
	)
	require.NoError(err)
	assert.NotNil(t, listenerCancelFn)
	assert.NotNil(t, wrpCancelFn)

	err = r.Listen()
	require.NoError(err)
	defer r.Close()

	///time.Sleep(1 * time.Second)

	send := []wrp.Message{
		{
			Type:   wrp.SimpleEventMessageType,
			Source: "11111",
		}, {
			Type:   wrp.SimpleEventMessageType,
			Source: "22222",
		},
	}
	// Send a message to the receiver.
	sock, err := sendMsgs(send, port)
	require.NoError(err)

	// Wait for the message to be received.
	for {
		if ctx.Err() != nil {
			require.Fail("timed out waiting for message")
			break
		}

		lock.Lock()
		eq := len(got) == len(send)
		lock.Unlock()
		if eq {
			sock.Close()
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	assert.Equal(t, send, got)

}

// findOpenPort finds an open port for listening on.
func findOpenPort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()

	addr := listener.Addr().(*net.TCPAddr)
	return addr.Port, nil
}

// sendMsgs sends a list of messages to the specified port.
// Send back the socket so it can be closed after all messages have been
// received.  Otherwise the messages may not be sent if aa defer sock.Close()
// is used.
func sendMsgs(msgs []wrp.Message, port int) (mangos.Socket, error) {
	sock, err := push.NewSocket()
	if err != nil {
		return nil, err
	}

	// Set the write queue length to 1.  This is the only way to ensure that
	// message delivery faiures are detected
	if err := sock.SetOption(mangos.OptionWriteQLen, 1); err != nil {
		return nil, err
	}

	// Set the send timeout to the configured value.  The other methods of
	// setting the timeout are not supported by the mangos library
	if err := sock.SetOption(mangos.OptionSendDeadline, 10*time.Millisecond); err != nil {
		return nil, err
	}

	err = sock.Dial(fmt.Sprintf("tcp://127.0.0.1:%d", port))

	if err != nil {
		return sock, err
	}

	for _, msg := range msgs {
		var buf []byte
		if err := wrp.NewEncoderBytes(&buf, wrp.Msgpack).Encode(msg); err != nil {
			return sock, err
		}

		for {
			if err := sock.Send(buf); err != nil {
				if errors.Is(err, mangos.ErrSendTimeout) {
					continue
				}

				return sock, err
			}

			break
		}
	}

	return sock, nil
}
