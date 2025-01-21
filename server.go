// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package wrpnng

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/xmidt-org/eventor"
	"github.com/xmidt-org/wrp-go/v3"
	"github.com/xmidt-org/wrpnng/internal/processors/stopping"
	"github.com/xmidt-org/wrpnng/internal/receiver"
	"github.com/xmidt-org/wrpnng/internal/sender"
)

var (
	errInvalidMsg = errors.New("invalid message")
)

// Server is a simple controller for managing a receiver and a set of senders.
//
// ingress and egress refer to the API side of the controller.
//   - ingress describes the messages coming into the controller.
//   - egress describes the messages leaving the controller.
//
// tx and rx refer to the network side of the controller.
//   - tx describes the messages being sent out.
//   - rx describes the messages being received.
type Server struct {
	rOpts []receiver.Option
	r     *receiver.Receiver

	sOpts []sender.Option

	egress eventor.Eventor[wrp.Modifier]

	senders senderMap

	rxObservers  wrp.Observers
	txObservers  wrp.Observers
	ingressChain stopping.Processors

	heartbeatInterval time.Duration
	heartbeatCancel   context.CancelFunc
	wg                sync.WaitGroup
	lock              sync.Mutex
}

var _ wrp.Processor = (*Server)(nil)

// NewServer creates a new Controller.  The controller is not started until Start is
// called.  The controller handles the registration message and sends heartbeats
// at regular intervals.  The default heartbeat interval is 30 seconds.
func NewServer(opts ...ServerOption) (*Server, error) {
	var srv Server

	defaults := []ServerOption{
		WithHeartbeatInterval(30 * time.Second),
	}

	vadors := []ServerOption{
		createReceiver(),
		createIngressChain(),
	}

	opts = append(defaults, opts...)
	opts = append(opts, vadors...)

	for _, opt := range opts {
		if opt != nil {
			if err := opt.apply(&srv); err != nil {
				return nil, err
			}
		}
	}

	return &srv, nil
}

// Start begins listening for messages.  It is idempotent.
func (srv *Server) Start() error {
	srv.lock.Lock()
	defer srv.lock.Unlock()

	if srv.heartbeatCancel != nil {
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())

	srv.heartbeatCancel = cancel
	srv.wg.Add(1)
	go srv.sendHeartbeat(ctx)

	return srv.r.Listen()
}

// Stop halts the controller.  It is idempotent.
func (srv *Server) Stop() error {
	srv.lock.Lock()
	defer srv.lock.Unlock()

	if srv.heartbeatCancel != nil {
		srv.heartbeatCancel()
		srv.heartbeatCancel = nil
	}

	err := errors.Join(
		srv.r.Close(),
		srv.senders.Close(),
	)

	srv.wg.Wait()
	return err
}

// ProcessWRP is called when a message should be sent to the network.
func (srv *Server) ProcessWRP(ctx context.Context, msg wrp.Message) error {
	return srv.ingressChain.ProcessWRP(ctx, msg)
}

func (srv *Server) handleRegisterMsg(_ context.Context, msg wrp.Message) error {
	if msg.Type != wrp.ServiceRegistrationMessageType {
		return wrp.ErrNotHandled
	}

	if msg.ServiceName == "" || msg.URL == "" {
		return errInvalidMsg
	}

	opts := append(srv.sOpts, sender.WithURL(msg.URL))
	return srv.senders.Upsert(msg.ServiceName, opts)
}

func (srv *Server) egressWRP(ctx context.Context, msg wrp.Message) error {
	srv.egress.Visit(func(m wrp.Modifier) {
		_, _ = m.ModifyWRP(ctx, msg)
	})

	return nil
}

// sendHeartbeat sends a ServiceAlive message at regular intervals until the
// context is canceled.
func (srv *Server) sendHeartbeat(ctx context.Context) {
	defer srv.wg.Done()

	msg := wrp.Message{
		Type: wrp.ServiceAliveMessageType,
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(srv.heartbeatInterval):
			srv.txObservers.ObserveWRP(ctx, msg)
			_ = srv.senders.ProcessWRP(ctx, msg)
		}
	}
}
