// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package filters

import (
	"context"
	"errors"
	"fmt"

	"github.com/xmidt-org/wrp-go/v3"
)

var (
	ErrUnsupported = errors.New("unsupported message type")
)

// ErrorOnUnsupportedMsgTypes returns a ProcessorFunc that returns an error if
// the message type is not supported.  If the message type is supported, the
// ProcessorFunc returns wrp.ErrNotHandled.
func ErrorOnUnsupportedMsgTypes() wrp.ProcessorFunc {
	return func(_ context.Context, m wrp.Message) error {
		if m.Type >= wrp.LastMessageType ||
			m.Type < 0 ||
			m.Type == wrp.Invalid0MessageType ||
			m.Type == wrp.Invalid1MessageType {
			return errors.Join(
				fmt.Errorf("invalid message type: %d", m.Type),
				ErrUnsupported,
			)
		}

		return wrp.ErrNotHandled
	}
}
