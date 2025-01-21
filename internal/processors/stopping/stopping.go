// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package stopping

import (
	"context"
	"errors"

	"github.com/xmidt-org/wrp-go/v3"
)

// Processors is a collection of Processors that can be used to process a message.
type Processors []wrp.Processor

// ProcessWRP iterates over the Processors, sequentially calling each Processor
// of the message.  The first Processor to return any value that is not
// wrp.ErrNotHandled will stop the iteration and return the error (or nil) value.
// If all Processors return ErrNotHandled, then ErrNotHandled is returned. If
// the context is canceled, the iteration stops and the context error value is
// returned.
func (p Processors) ProcessWRP(ctx context.Context, msg wrp.Message) error {
	for _, proc := range p {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if proc == nil {
			continue
		}

		err := proc.ProcessWRP(ctx, msg)
		if errors.Is(err, wrp.ErrNotHandled) {
			continue
		}
		return err
	}

	return wrp.ErrNotHandled
}
