// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package wrpnng

import (
	"time"

	"github.com/xmidt-org/wrp-go/v3"
	"github.com/xmidt-org/wrpnng/internal/filters"
	"github.com/xmidt-org/wrpnng/internal/processors/stopping"
	"github.com/xmidt-org/wrpnng/internal/receiver"
)

// ServerOption is the interface implemented by types that can be used to
// configure the service.
type ServerOption interface {
	apply(*Server) error
}

type errServerOptionFunc func(*Server) error

func (f errServerOptionFunc) apply(srv *Server) error {
	return f(srv)
}

func serverOptionFunc(f func(*Server)) errServerOptionFunc {
	return errServerOptionFunc(func(srv *Server) error {
		f(srv)
		return nil
	})
}

// RXURL sets the URL used for listening to network clients.  This is required.
// The URL should be in the format of "tcp://<ip>:<port>" unless other transports
// are registered.  This URL represents the rx network side of the controller.
func RXURL(url string) ServerOption {
	return serverOptionFunc(func(srv *Server) {
		srv.rOpts = append(srv.rOpts, receiver.WithURL(url))
	})
}

// RXTimeout sets the timeout for receiving messages.
func RXTimeout(timeout time.Duration) ServerOption {
	return serverOptionFunc(func(srv *Server) {
		srv.rOpts = append(srv.rOpts, receiver.WithRecvTimeout(timeout))
	})
}

// WithHeartbeatInterval sets the interval for sending heartbeats.
func WithHeartbeatInterval(interval time.Duration) ServerOption {
	return serverOptionFunc(func(srv *Server) {
		srv.heartbeatInterval = interval
	})
}

// WithRXObserver adds observers to the rx chain.  The rx chain represents the
// processing of messages received from the network.
func WithRXObserver(observer wrp.Observer) ServerOption {
	return serverOptionFunc(func(srv *Server) {
		srv.rxObservers = append(srv.rxObservers, observer)
	})
}

// WithTXObserver adds observers to the tx chain.  The tx chain represents the
// processing of messages sent to the network.
func WithTXObserver(observer wrp.Observer) ServerOption {
	return serverOptionFunc(func(srv *Server) {
		srv.txObservers = append(srv.txObservers, observer)
	})
}

// WithEgressModifier adds a modifier to the list of modifiers that are informed
// of messages leaving the controller.  Return values from the modifiers are
// ignored.
func WithEgressModifier(modifier wrp.Modifier, cancel ...*func()) ServerOption {
	return serverOptionFunc(func(srv *Server) {
		cancelFn := srv.egress.Add(modifier)
		for i := range cancel {
			if cancel[i] != nil {
				*cancel[i] = cancelFn
			}
		}
	})
}

//-----------------------------------------------------------------------------

func createReceiver() ServerOption {
	return errServerOptionFunc(func(srv *Server) error {
		chain := stopping.Processors{
			wrp.ObserverAsProcessor(srv.rxObservers),
			filters.ErrorOnUnsupportedMsgTypes(),
			wrp.ProcessorFunc(srv.handleRegisterMsg),
			filters.ErrorOnLocalMsgTypes(),
			wrp.ProcessorFunc(srv.egressWRP),
		}

		opts := append(srv.rOpts,
			receiver.WithModifyWRP(wrp.ProcessorAsModifier(chain)),
		)

		r, err := receiver.New(opts...)
		if err != nil {
			return err
		}

		srv.r = r
		return nil
	})
}

func createIngressChain() ServerOption {
	return errServerOptionFunc(func(srv *Server) error {
		srv.ingressChain = stopping.Processors{
			filters.ErrorOnUnsupportedMsgTypes(),
			filters.ErrorOnLocalMsgTypes(),
			wrp.ObserverAsProcessor(srv.txObservers),
			&srv.senders,
		}
		return nil
	})
}
