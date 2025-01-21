// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package receiver

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewStart(t *testing.T) {
	tests := []struct {
		name     string
		options  []Option
		want     *Receiver
		newErr   bool
		startErr bool
	}{
		{
			name:   "With no valid url",
			newErr: true,
		},
		{
			name: "With valid URL and timeout",
			options: []Option{
				WithURL("tcp://127.0.0.1:0"),
				WithRecvTimeout(100 * time.Millisecond),
			},
			want: &Receiver{
				url:     "tcp://127.0.0.1:0",
				timeout: 100 * time.Millisecond,
			},
		},
		{
			name: "With invalid URL - not detected until Start()",
			options: []Option{
				WithURL("invalid-url"),
			},
			want: &Receiver{
				url: "invalid-url",
			},
			startErr: true,
		},
		{
			name: "With negative timeout, should be ignored",
			options: []Option{
				WithURL("tcp://127.0.0.1:0"),
				WithRecvTimeout(100 * time.Millisecond),
				WithRecvTimeout(-100 * time.Millisecond),
			},
			want: &Receiver{
				url:     "tcp://127.0.0.1:0",
				timeout: 100 * time.Millisecond,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := New(tt.options...)
			if tt.newErr {
				assert.Error(t, err)
				assert.Nil(t, r)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, r)
			if tt.want != nil {
				assert.Equal(t, tt.want.url, r.url)
				assert.Equal(t, tt.want.timeout, r.timeout)
			}

			// Start a 2nd time to ensure it doesn't error.
			for i := 0; i < 2; i++ {
				err = r.Listen()
				if tt.startErr {
					assert.Error(t, err)
					return
				}
				assert.NoError(t, err)
			}

		})
	}
}
