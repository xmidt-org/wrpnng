// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package sender

import "go.nanomsg.org/mangos/v3"

type mockSocket struct {
	sendRv error
}

var _ mangos.Socket = (*mockSocket)(nil)

func (m *mockSocket) Info() mangos.ProtocolInfo {
	return mangos.ProtocolInfo{}
}

func (m *mockSocket) Close() error {
	return nil
}

func (m *mockSocket) Send([]byte) error {
	return m.sendRv
}

func (m *mockSocket) Recv() ([]byte, error) {
	return nil, nil
}

func (m *mockSocket) SendMsg(*mangos.Message) error {
	return nil
}

func (m *mockSocket) RecvMsg() (*mangos.Message, error) {
	return nil, nil
}

func (m *mockSocket) Dial(addr string) error {
	return nil
}

func (m *mockSocket) DialOptions(addr string, options map[string]interface{}) error {
	return nil
}

func (m *mockSocket) NewDialer(addr string, options map[string]interface{}) (mangos.Dialer, error) {
	return nil, nil
}

func (m *mockSocket) Listen(addr string) error {
	return nil
}

func (m *mockSocket) ListenOptions(addr string, options map[string]interface{}) error {
	return nil
}

func (m *mockSocket) NewListener(addr string, options map[string]interface{}) (mangos.Listener, error) {
	return nil, nil
}

func (m *mockSocket) GetOption(name string) (interface{}, error) {
	return nil, nil
}

func (m *mockSocket) SetOption(name string, value interface{}) error {
	return nil
}

func (m *mockSocket) OpenContext() (mangos.Context, error) {
	return nil, nil
}

func (m *mockSocket) SetPipeEventHook(mangos.PipeEventHook) mangos.PipeEventHook {
	return nil
}
