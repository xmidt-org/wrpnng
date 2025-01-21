// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/xmidt-org/wrp-go/v3"
	"github.com/xmidt-org/wrpnng"
)

func mainE() error {
	server, err := wrpnng.NewServer(
		wrpnng.RXURL("tcp://127.0.0.1:6666"),
		wrpnng.RXTimeout(10*time.Second),
		wrpnng.WithEgressModifier(
			wrp.Modifiers{
				wrp.ObserverAsModifier(
					wrp.ObserverFunc(func(_ context.Context, msg wrp.Message) {
						fmt.Println("received message", msg)
					}),
				),
			}),
	)
	if err != nil {
		return err
	}

	fmt.Println("Starting server...")
	err = server.Start()
	if err != nil {
		return err
	}
	defer server.Stop()

	// wait forever
	select {}
}

func main() {
	if err := mainE(); err != nil {
		fmt.Println(err)
	}
}
