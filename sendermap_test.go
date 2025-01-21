// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package wrpnng

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/wrp-go/v3"
	"github.com/xmidt-org/wrpnng/internal/sender"
)

type mockSender struct {
	processErr   error
	processCount int
	dialErr      error
}

func (m *mockSender) ProcessWRP(_ context.Context, _ wrp.Message) error {
	m.processCount++
	return m.processErr
}

func (m *mockSender) Close() error {
	return nil
}

func (m *mockSender) Dial() error {
	return m.dialErr
}

func TestSenderMap_ProcessWRP(t *testing.T) {
	randomErr := errors.New("random error")
	tests := []struct {
		name        string
		senders     map[string]*mockSender
		msg         wrp.Message
		expect      map[string]*mockSender
		expectedErr error
	}{
		{
			name: "ServiceAliveMessageType",
			senders: map[string]*mockSender{
				"service_1": {},
				"service_2": {},
			},
			msg: wrp.Message{Type: wrp.ServiceAliveMessageType},
			expect: map[string]*mockSender{
				"service_1": {processCount: 1},
				"service_2": {processCount: 1},
			},
		}, {
			name: "Valid Destination",
			senders: map[string]*mockSender{
				"service_1": {},
				"service_2": {},
			},
			msg: wrp.Message{
				Type:        wrp.SimpleRequestResponseMessageType,
				Destination: "mac:112233445566/service_1/ignored",
			},
			expect: map[string]*mockSender{
				"service_1": {processCount: 1},
				"service_2": {},
			},
		}, {
			name: "Invalid Destination",
			senders: map[string]*mockSender{
				"service_1": {},
			},
			msg: wrp.Message{
				Type:        wrp.SimpleRequestResponseMessageType,
				Destination: "mac:112233445566/invalid/ignored",
			},
			expectedErr: wrp.ErrNotHandled,
			expect: map[string]*mockSender{
				"service_1": {},
			},
		}, {
			name: "Sender Error",
			senders: map[string]*mockSender{
				"service_1": {processErr: randomErr},
			},
			msg: wrp.Message{
				Type:        wrp.SimpleRequestResponseMessageType,
				Destination: "mac:112233445566/service_1/ignored",
			},
			expectedErr: randomErr,
		}, {
			name: "Invalid locator",
			msg: wrp.Message{
				Type:        wrp.SimpleRequestResponseMessageType,
				Destination: "service_1/ignored",
			},
			expectedErr: wrp.ErrorInvalidLocator,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := &senderMap{
				senders: make(map[string]limitedSender),
			}

			for k, v := range tt.senders {
				sm.senders[k] = v
			}

			err := sm.ProcessWRP(context.Background(), tt.msg)
			if tt.expectedErr != nil {
				assert.ErrorIs(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
			}

			if tt.expect != nil {
				for k, v := range tt.expect {
					require.NotNil(t, sm.senders[k])
					assert.Equal(t, v.processCount, sm.senders[k].(*mockSender).processCount)
				}
			}
		})
	}
}

func TestSenderMap_upsert(t *testing.T) {
	factory := func(opts ...sender.Option) (limitedSender, error) {
		return &mockSender{}, nil
	}

	tests := []struct {
		name           string
		initialSenders map[string]limitedSender
		upsertName     string
		factory        limitedSenderFactory
		opts           []sender.Option
		expectError    bool
	}{
		{
			name:       "Upsert new sender",
			upsertName: "service_1",
		}, {
			name: "Upsert existing sender",
			initialSenders: map[string]limitedSender{
				"service_1": new(mockSender),
			},
			upsertName: "service_1",
		}, {
			name:       "Cause the factory to fail",
			upsertName: "service_1",
			factory: func(opts ...sender.Option) (limitedSender, error) {
				return nil, errors.New("factory error")
			},
			expectError: true,
		}, {
			name:       "using a faulty dialer",
			upsertName: "service_1",
			factory: func(opts ...sender.Option) (limitedSender, error) {
				return &mockSender{
					dialErr: errors.New("dial error"),
				}, nil
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := &senderMap{
				senders: tt.initialSenders,
			}

			if tt.factory == nil {
				tt.factory = factory
			}

			err := sm.upsert(tt.upsertName, tt.opts, tt.factory)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, sm.senders[tt.upsertName])
			}
		})
	}
}

func TestSenderMap_Remove(t *testing.T) {
	sm := &senderMap{
		senders: make(map[string]limitedSender),
	}

	sm.senders["service1"] = &mockSender{}
	err := sm.Remove("service1")
	assert.NoError(t, err)
	assert.Nil(t, sm.senders["service1"])
}

func TestSenderMap_Close(t *testing.T) {
	sm := &senderMap{
		senders: make(map[string]limitedSender),
	}

	sm.senders["service1"] = &mockSender{}
	sm.senders["service2"] = &mockSender{}

	err := sm.Close()
	assert.NoError(t, err)
	assert.Nil(t, sm.senders)
}
