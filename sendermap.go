// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package wrpnng

import (
	"context"
	"sync"

	"github.com/xmidt-org/wrp-go/v3"
	"github.com/xmidt-org/wrpnng/internal/sender"
)

type limitedSender interface {
	ProcessWRP(context.Context, wrp.Message) error
	Dial() error
	Close() error
}

type limitedSenderFactory func(...sender.Option) (limitedSender, error)

// senderMap is a map of senders that can process WRP messages.  It is safe for
// concurrent access.
//
// If a sender is closed, it is removed from the map automatically.
type senderMap struct {
	senders map[string]limitedSender
	lock    sync.RWMutex
}

// ProcessWRP sends the message to the appropriate sender.  If the message is a
// ServiceAlive message, it is sent to all senders.  If the message destination
// is not found, ErrNotHandled is returned.
func (sm *senderMap) ProcessWRP(ctx context.Context, msg wrp.Message) error {
	if msg.Type == wrp.ServiceAliveMessageType {
		// Send the message to all senders.

		senders := make([]limitedSender, 0, len(sm.senders))

		// Only lock while making a copy of the sender list.
		sm.lock.RLock()
		for _, s := range sm.senders {
			senders = append(senders, s)
		}
		sm.lock.RUnlock()

		for _, s := range senders {
			_ = s.ProcessWRP(ctx, msg)
		}
		return nil
	}

	// Send the message to the appropriate sender.
	dest, err := wrp.ParseLocator(msg.To())
	if err != nil {
		return err
	}

	sm.lock.RLock()
	target := sm.senders[dest.Service]
	sm.lock.RUnlock()

	if target != nil {
		return target.ProcessWRP(ctx, msg)
	}

	return wrp.ErrNotHandled
}

// Upsert adds or updates a sender in the map.  If a sender with the same name
// already exists, it is closed and replaced with the new sender.  The new
// sender is dialed being added to the map.
//
// Upsert also sends the sender an authorization message.
func (sm *senderMap) Upsert(name string, opts []sender.Option) error {
	factory := func(opts ...sender.Option) (limitedSender, error) {
		return sender.New(opts...)
	}
	return sm.upsert(name, opts, factory)
}

// upsert is broken out for testing purposes.  Mainly so we can inject a mock
// sender factory.
func (sm *senderMap) upsert(name string,
	opts []sender.Option,
	factory limitedSenderFactory,
) error {
	opts = append(opts, sender.WithCloseListener(func(error) {
		_ = sm.Remove(name)
	}))

	s, err := factory(opts...)
	if err != nil {
		return err
	}

	err = s.Dial()
	if err != nil {
		_ = s.Close()
		return err
	}

	sm.lock.Lock()

	if sm.senders == nil {
		sm.senders = make(map[string]limitedSender)
	}

	existing := sm.senders[name]
	if existing != nil {
		_ = existing.Close()
	}
	sm.senders[name] = s

	sm.lock.Unlock()

	// Send a message to the new sender to authorize it.
	status := int64(200)
	_ = s.ProcessWRP(context.Background(), wrp.Message{
		Type:   wrp.AuthorizationMessageType,
		Status: &status,
	})

	return nil
}

// Remove removes a sender from the map.  If the sender is found, it is closed
// and removed.
func (sm *senderMap) Remove(name string) error {
	sm.lock.Lock()
	defer sm.lock.Unlock()

	s := sm.senders[name]
	if s != nil {
		_ = s.Close()
		delete(sm.senders, name)
	}

	return nil
}

// Close closes all senders in the map.
func (sm *senderMap) Close() error {
	sm.lock.Lock()
	defer sm.lock.Unlock()

	for _, s := range sm.senders {
		_ = s.Close()
	}

	sm.senders = nil
	return nil
}
