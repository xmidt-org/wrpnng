// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package stopping

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xmidt-org/wrp-go/v3"
)

type mockProcessor struct {
	f   func()
	err error
}

func (m *mockProcessor) ProcessWRP(_ context.Context, _ wrp.Message) error {
	if m.f != nil {
		m.f()
	}
	return m.err
}

func TestProcessors_ProcessWRP(t *testing.T) {
	tests := []struct {
		name        string
		processors  Processors
		expectedErr error
	}{
		{
			name: "All Processors Return ErrNotHandled",
			processors: Processors{
				&mockProcessor{err: wrp.ErrNotHandled},
				nil,
				&mockProcessor{err: wrp.ErrNotHandled},
			},
			expectedErr: wrp.ErrNotHandled,
		},
		{
			name: "Processor Returns Error",
			processors: Processors{
				&mockProcessor{err: wrp.ErrNotHandled},
				&mockProcessor{err: errors.New("some error")},
				&mockProcessor{err: nil, f: func() {
					t.Error("should not be called")
				}},
			},
			expectedErr: errors.New("some error"),
		},
		{
			name: "Processor Returns Nil",
			processors: Processors{
				&mockProcessor{err: wrp.ErrNotHandled},
				&mockProcessor{err: nil},
				&mockProcessor{err: nil, f: func() {
					t.Error("should not be called")
				}},
			},
			expectedErr: nil,
		},
		{
			name: "Context Canceled",
			processors: Processors{
				&mockProcessor{err: wrp.ErrNotHandled,
					f: func() {
						t.Error("should not be called")
					}},
			},
			expectedErr: context.Canceled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			if tt.name == "Context Canceled" {
				cancel()
			} else {
				defer cancel()
			}

			err := tt.processors.ProcessWRP(ctx, wrp.Message{})
			if tt.name == "Processor Returns Error" {
				assert.EqualError(t, err, tt.expectedErr.Error())
			} else {
				assert.Equal(t, tt.expectedErr, err)
			}
		})
	}
}
