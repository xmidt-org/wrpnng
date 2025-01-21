// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package filters

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xmidt-org/wrp-go/v3"
)

func TestErrorOnLocalMsgTypes(t *testing.T) {
	tests := []struct {
		name        string
		messageType wrp.MessageType
		expectedErr error
	}{
		{
			name:        "Authorization Message Type",
			messageType: wrp.AuthorizationMessageType,
			expectedErr: ErrLocalDisallowed,
		}, {
			name:        "Service Registration Message Type",
			messageType: wrp.ServiceRegistrationMessageType,
			expectedErr: ErrLocalDisallowed,
		}, {
			name:        "Service Alive Message Type",
			messageType: wrp.ServiceAliveMessageType,
			expectedErr: ErrLocalDisallowed,
		}, {
			name:        "Event Message Type",
			messageType: wrp.SimpleEventMessageType,
			expectedErr: wrp.ErrNotHandled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor := ErrorOnLocalMsgTypes()
			msg := wrp.Message{Type: tt.messageType}
			err := processor(context.Background(), msg)

			if tt.expectedErr == ErrUnsupported {
				assert.True(t, errors.Is(err, ErrUnsupported))
			} else {
				assert.Equal(t, tt.expectedErr, err)
			}
		})
	}
}
