// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package filters

import (
	"context"
	"errors"

	"github.com/xmidt-org/wrp-go/v3"
)

var (
	ErrLocalDisallowed = errors.New("local message types are not allowed")
)

// ErrorOnLocalMsgTypes returns a ProcessorFunc that returns an error if
// the message type is a local message type.  If the message type is not a local
// message type, the ProcessorFunc returns wrp.ErrNotHandled.
func ErrorOnLocalMsgTypes() wrp.ProcessorFunc {
	return func(_ context.Context, m wrp.Message) error {
		switch m.Type {
		case wrp.AuthorizationMessageType,
			wrp.ServiceRegistrationMessageType,
			wrp.ServiceAliveMessageType:
			return ErrLocalDisallowed
		}
		return wrp.ErrNotHandled
	}
}
