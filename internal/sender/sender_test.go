// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package sender

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/wrp-go/v3"
)

func TestNewDial(t *testing.T) {
	tests := []struct {
		name        string
		options     []Option
		addValidURL bool
		want        *Sender
		newErr      bool
		dialErr     bool
	}{
		{
			name: "With valid URL and timeout",
			options: []Option{
				WithSendTimeout(100 * time.Millisecond),
			},
			addValidURL: true,
			want: &Sender{
				sendDeadline: 100 * time.Millisecond,
			},
		}, {
			name:   "With missing URL",
			newErr: true,
		}, {
			name: "With negative timeout",
			options: []Option{
				WithSendTimeout(-100 * time.Millisecond),
			},
			addValidURL: true,
			want: &Sender{
				sendDeadline: 0,
			},
		}, {
			name: "With invalid URL",
			options: []Option{
				WithURL("invalid://url"),
			},
			dialErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ml := mockListener{}

			if tt.addValidURL {
				url, err := findOpenPort()
				require.NoError(t, err)
				ml.url = url
				tt.options = append(tt.options, WithURL(ml.url))
				if tt.want != nil {
					tt.want.url = ml.url
				}
			}

			sdr, err := New(tt.options...)
			if tt.newErr {
				assert.Error(t, err)
				assert.Nil(t, sdr)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, sdr)
			if tt.want != nil {
				assert.Equal(t, tt.want.url, sdr.url)
				assert.Equal(t, tt.want.sendDeadline, sdr.sendDeadline)
			}

			if tt.addValidURL {
				err = ml.Listen()
				assert.NoError(t, err)
				defer ml.Close()
			}

			// Multiple calls to Dial should be fine.
			for i := 0; i < 2; i++ {
				err = sdr.Dial()
				if tt.dialErr {
					assert.Error(t, err)
					return
				}

				require.NoError(t, err, "url: '%s', iteration %d", sdr.url, i)
			}

			err = sdr.Close()
			assert.NoError(t, err)
		})
	}
}

func TestProcessWRP(t *testing.T) {
	errList := make([]error, 0)

	s, err := New(
		WithURL("invalid://url"),
		WithCloseListener(func(err error) {
			errList = append(errList, err)
		}),
	)
	require.NoError(t, err)

	s.sock = &mockSocket{
		sendRv: errors.New("send error"),
	}

	err = s.ProcessWRP(context.Background(), wrp.Message{})
	require.Error(t, err)
	require.Len(t, errList, 1)
	for _, e := range errList {
		assert.ErrorIs(t, e, errList[0])
	}
}

func TestEnd2End(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	mc := mockListener{}
	err := mc.Listen()
	require.NoError(err)

	var marker int
	var closeFn func()
	assert.Nil(closeFn)

	sdr, err := New(
		WithURL(mc.url),
		WithCloseListener(func(error) {
			marker++
		}, nil, &closeFn),
	)
	require.NoError(err)
	require.NotNil(sdr)
	assert.NotNil(closeFn)

	err = sdr.Dial()
	require.NoError(err)

	// Multiple calls to Dial should be fine.
	err = sdr.Dial()
	require.NoError(err)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	rv := make(chan []byte, 1)
	go func() {
		for {
			select {
			case <-ctx.Done():
				require.Fail("timeout")
				return
			default:
				buf, err := mc.sock.Recv()
				if err != nil {
					continue
				}
				rv <- buf
				return
			}
		}
	}()

	err = sdr.ProcessWRP(ctx, wrp.Message{
		Payload: []byte("test"),
	})
	require.NoError(err)

	select {
	case <-ctx.Done():
		require.Fail("timeout")
	case buf := <-rv:
		var got wrp.Message
		err = wrp.NewDecoderBytes(buf, wrp.Msgpack).Decode(&got)
		require.NoError(err)
		assert.Equal([]byte("test"), got.Payload)
	}

	// Send in a context that is already canceled
	ctx, cancel = context.WithCancel(context.Background())
	cancel()

	err = sdr.ProcessWRP(ctx, wrp.Message{
		Payload: []byte("test"),
	})
	require.Error(err)
	assert.ErrorIs(err, context.Canceled)

	// Close the connection.
	err = sdr.Close()
	require.NoError(err)

	// Send when the connection is closed.
	err = sdr.ProcessWRP(context.Background(), wrp.Message{
		Payload: []byte("test"),
	})
	require.Error(err)
	assert.ErrorIs(err, ErrConnClosed)

	assert.Equal(1, marker)
}
