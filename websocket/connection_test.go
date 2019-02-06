/*
Copyright 2019 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package websocket

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	ktesting "github.com/knative/pkg/logging/testing"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/gorilla/websocket"
)

const (
	target = "test"
)

type inspectableConnection struct {
	nextReaderCalls   chan struct{}
	writeMessageCalls chan struct{}
	closeCalls        chan struct{}

	nextReaderFunc func() (int, io.Reader, error)
}

func (c *inspectableConnection) WriteMessage(messageType int, data []byte) error {
	c.writeMessageCalls <- struct{}{}
	return nil
}

func (c *inspectableConnection) NextReader() (int, io.Reader, error) {
	c.nextReaderCalls <- struct{}{}
	return c.nextReaderFunc()
}

func (c *inspectableConnection) Close() error {
	c.closeCalls <- struct{}{}
	return nil
}

func TestRetriesWhileConnect(t *testing.T) {
	want := 2
	got := 0

	spy := &inspectableConnection{
		closeCalls: make(chan struct{}, 1),
	}

	connFactory := func() (rawConnection, error) {
		got++
		if got == want {
			return spy, nil
		}
		return nil, errors.New("not yet")
	}
	conn := newConnection(connFactory, nil)

	conn.connect()
	conn.Shutdown()

	if got != want {
		t.Fatalf("Wanted %v retries. Got %v.", want, got)
	}
	if len(spy.closeCalls) != 1 {
		t.Fatalf("Wanted 'Close' to be called once, but got %v", len(spy.closeCalls))
	}
}

func TestSendErrorOnNoConnection(t *testing.T) {
	want := ErrConnectionNotEstablished

	conn := &ManagedConnection{}
	got := conn.Send("test")

	if got != want {
		t.Fatalf("Wanted error to be %v, but it was %v.", want, got)
	}
}

func TestSendErrorOnEncode(t *testing.T) {
	spy := &inspectableConnection{
		writeMessageCalls: make(chan struct{}, 1),
	}

	connFactory := func() (rawConnection, error) {
		return spy, nil
	}
	conn := newConnection(connFactory, nil)
	conn.connect()
	// gob cannot encode nil values
	got := conn.Send(nil)

	if got == nil {
		t.Fatal("Expected an error but got none")
	}
	if len(spy.writeMessageCalls) != 0 {
		t.Fatalf("Expected 'WriteMessage' not to be called, but was called %v times", spy.writeMessageCalls)
	}
}

func TestSendMessage(t *testing.T) {
	spy := &inspectableConnection{
		writeMessageCalls: make(chan struct{}, 1),
	}
	connFactory := func() (rawConnection, error) {
		return spy, nil
	}
	conn := newConnection(connFactory, nil)
	conn.connect()
	got := conn.Send("test")

	if got != nil {
		t.Fatalf("Expected no error but got: %+v", got)
	}
	if len(spy.writeMessageCalls) != 1 {
		t.Fatalf("Expected 'WriteMessage' to be called once, but was called %v times", spy.writeMessageCalls)
	}
}

func TestReceiveMessage(t *testing.T) {
	testMessage := "testmessage"

	spy := &inspectableConnection{
		writeMessageCalls: make(chan struct{}, 1),
		nextReaderCalls:   make(chan struct{}, 1),
		nextReaderFunc: func() (int, io.Reader, error) {
			return websocket.TextMessage, strings.NewReader(testMessage), nil
		},
	}
	connFactory := func() (rawConnection, error) {
		return spy, nil
	}

	messageChan := make(chan []byte, 1)
	conn := newConnection(connFactory, messageChan)
	conn.connect()
	go conn.keepalive()

	got := <-messageChan

	if string(got) != testMessage {
		t.Errorf("Received the wrong message, wanted %q, got %q", testMessage, string(got))
	}
}

func TestCloseClosesConnection(t *testing.T) {
	spy := &inspectableConnection{
		closeCalls: make(chan struct{}, 1),
	}
	connFactory := func() (rawConnection, error) {
		return spy, nil
	}
	conn := newConnection(connFactory, nil)
	conn.connect()
	conn.Shutdown()

	if len(spy.closeCalls) != 1 {
		t.Fatalf("Expected 'Close' to be called once, got %v", len(spy.closeCalls))
	}
}

func TestCloseIgnoresNoConnection(t *testing.T) {
	conn := &ManagedConnection{
		closeChan: make(chan struct{}, 1),
	}
	got := conn.Shutdown()

	if got != nil {
		t.Fatalf("Expected no error, got %v", got)
	}
}

func TestDurableConnectionWhenConnectionBreaksDown(t *testing.T) {
	upgrader := websocket.Upgrader{}
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}

		// Close the connection immediately, forces a reconnect.
		c.Close()
	}))
	defer s.Close()

	logger := ktesting.TestLogger(t)
	target := "ws" + strings.TrimPrefix(s.URL, "http")
	conn := NewDurableSendingConnection(target, logger)
	defer conn.Shutdown()

	for i := 0; i < 10; i++ {
		err := wait.PollImmediate(50*time.Millisecond, 5*time.Second, func() (bool, error) {
			if err := conn.Send("test"); err != nil {
				return false, nil
			}
			return true, nil
		})

		if err != nil {
			t.Errorf("Timed out trying to send a message: %v", err)
		}
	}
}

func TestConnectFailureReturnsError(t *testing.T) {
	connFactory := func() (rawConnection, error) {
		return nil, ErrConnectionNotEstablished
	}

	conn := newConnection(connFactory, nil)

	// Shorten the connection backoff duration for this test
	conn.connectionBackoff.Duration = 1 * time.Millisecond

	got := conn.connect()

	if got == nil {
		t.Fatal("Expected an error but got none")
	}
}

func TestKeepaliveWithNoConnectionReturnsError(t *testing.T) {
	conn := newConnection(nil, nil)
	got := conn.keepalive()

	if got == nil {
		t.Fatal("Expected an error but got none")
	}
}
