// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package wrpnng

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/wrp-go/v3"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		options     []ServerOption
		expectError bool
	}{
		{
			name:        "No options",
			expectError: true,
		}, {
			name: "Valid options",
			options: []ServerOption{
				RXURL("url"),
				RXTimeout(10 * time.Second),
				WithHeartbeatInterval(10 * time.Second),
				WithRXObserver(wrp.ObserverFunc(func(_ context.Context, _ wrp.Message) {})),
				WithTXObserver(wrp.ObserverFunc(func(_ context.Context, _ wrp.Message) {})),
				WithEgressModifier(wrp.ModifierFunc(func(_ context.Context, _ wrp.Message) (wrp.Message, error) {
					return wrp.Message{}, nil
				})),
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			got, err := NewServer(tt.options...)
			if tt.expectError {
				require.Error(t, err)
				assert.Nil(t, got)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, got)
		})
	}
}

/*
func TestController_Start(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(c *Controller)
		expectError bool
	}{
		{
			name: "Start successfully",
			setup: func(c *Controller) {
				c.r = &receiver.Receiver{}
				c.heartbeatInterval = time.Second
			},
			expectError: false,
		},
		{
			name: "Receiver is nil",
			setup: func(c *Controller) {
				c.heartbeatInterval = time.Second
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Controller{}
			tt.setup(c)

			err := c.Start()
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
*/

func TestEnd2End(t *testing.T) {
	url, err := findOpenURL()
	require.NoError(t, err)
	require.NotEmpty(t, url)

	var heartbeat int
	var other int
	var lock sync.Mutex
	c, err := NewServer(
		RXURL(url),
		WithHeartbeatInterval(100*time.Millisecond),
		WithTXObserver(
			wrp.ObserverFunc(func(_ context.Context, msg wrp.Message) {
				if msg.Type == wrp.ServiceAliveMessageType {
					lock.Lock()
					heartbeat++
					lock.Unlock()
					return
				}
				fmt.Println("Other")
				lock.Lock()
				other++
				lock.Unlock()
			})),
	)
	require.NoError(t, err)
	require.NotNil(t, c)

	// Start the controller
	err = c.Start()
	require.NoError(t, err)

	// Starting a second time should be a no-op.
	err = c.Start()
	require.NoError(t, err)

	_ = c.ProcessWRP(context.Background(), wrp.Message{
		Type: wrp.SimpleEventMessageType,
	})

	for {
		lock.Lock()
		hb := heartbeat
		o := other
		lock.Unlock()

		if hb > 0 && o > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	err = c.Stop()
	assert.NoError(t, err)
}
