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

func TestErrorOnUnsupportedMsgTypes(t *testing.T) {
	tests := []struct {
		name        string
		messageType wrp.MessageType
		expectedErr error
	}{
		{
			name:        "Unsupported Message Type",
			messageType: wrp.MessageType(wrp.LastMessageType + 1),
			expectedErr: ErrUnsupported,
		},
		{
			name:        "Invalid0 Message Type",
			messageType: wrp.Invalid0MessageType,
			expectedErr: ErrUnsupported,
		},
		{
			name:        "Invalid1 Message Type",
			messageType: wrp.Invalid1MessageType,
			expectedErr: ErrUnsupported,
		},
		{
			name:        "Supported Message Type",
			messageType: wrp.SimpleRequestResponseMessageType,
			expectedErr: wrp.ErrNotHandled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor := ErrorOnUnsupportedMsgTypes()
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
