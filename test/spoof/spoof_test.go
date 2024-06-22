/*
Copyright 2021 The Knative Authors

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

// spoof contains logic to make polling HTTP requests against an endpoint with optional host spoofing.

package spoof

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"sync/atomic"
	"testing"
	"time"
)

var (
	successResponse = &http.Response{
		Status:     "200 ok",
		StatusCode: http.StatusOK,
		Header:     http.Header{},
		Body:       http.NoBody,
	}
	errRetriable    = errors.New("connection reset by peer")
	errNonRetriable = errors.New("foo")
)

type fakeTransport struct {
	response *http.Response
	err      error
	calls    atomic.Int32
}

func (ft *fakeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	call := ft.calls.Add(1)
	if ft.response != nil && call == 2 {
		// If both a response and an error is defined, we return just the response on
		// the second call to simulate a retry that passes eventually.
		return ft.response, nil
	}
	return ft.response, ft.err
}

func TestSpoofingClient_CheckEndpointState(t *testing.T) {
	tests := []struct {
		name      string
		transport *fakeTransport
		inState   ResponseChecker
		wantErr   bool
		wantCalls int32
	}{{
		name:      "Non matching response doesn't trigger a second check",
		transport: &fakeTransport{response: successResponse},
		inState: func(resp *Response) (done bool, err error) {
			return false, nil
		},
		wantErr:   false,
		wantCalls: 1,
	}, {
		name:      "Error response doesn't trigger a second check",
		transport: &fakeTransport{response: successResponse},
		inState: func(resp *Response) (done bool, err error) {
			return false, errors.New("response error")
		},
		wantErr:   true,
		wantCalls: 1,
	}, {
		name:      "OK response doesn't trigger a second check",
		transport: &fakeTransport{response: successResponse},
		inState: func(resp *Response) (done bool, err error) {
			return true, nil
		},
		wantErr:   false,
		wantCalls: 1,
	}, {
		name:      "Retriable error is retried",
		transport: &fakeTransport{err: errRetriable, response: successResponse},
		inState: func(resp *Response) (done bool, err error) {
			return true, nil
		},
		wantErr:   false,
		wantCalls: 2,
	}, {
		name:      "Nonretriable error is not retried",
		transport: &fakeTransport{err: errNonRetriable, response: successResponse},
		inState: func(resp *Response) (done bool, err error) {
			return true, nil
		},
		wantErr:   true,
		wantCalls: 1,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := &SpoofingClient{
				Client:          &http.Client{Transport: tt.transport},
				Logf:            t.Logf,
				RequestInterval: 1,
				RequestTimeout:  time.Second,
			}
			url := &url.URL{
				Host:   "fake.knative.net",
				Scheme: "http",
			}
			_, err := sc.CheckEndpointState(context.TODO(), url, tt.inState, "")
			if (err != nil) != tt.wantErr {
				t.Errorf("SpoofingClient.CheckEndpointState() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got, want := tt.transport.calls.Load(), tt.wantCalls; got != want {
				t.Errorf("Expected Transport to be invoked %d time but got invoked %d", want, got)
			}
		})
	}
}

func TestSpoofingClient_WaitForEndpointState(t *testing.T) {
	tests := []struct {
		name      string
		transport *fakeTransport
		inState   ResponseChecker
		wantErr   bool
		wantCalls int32
	}{{
		name:      "OK response doesn't trigger a second request",
		transport: &fakeTransport{response: successResponse},
		inState: func(resp *Response) (done bool, err error) {
			return true, nil
		},
		wantErr:   false,
		wantCalls: 1,
	}, {
		name:      "Error response doesn't trigger more requests",
		transport: &fakeTransport{response: successResponse},
		inState: func(resp *Response) (done bool, err error) {
			return false, errors.New("response error")
		},
		wantErr:   true,
		wantCalls: 1,
	}, {
		name:      "Non matching response triggers more requests",
		transport: &fakeTransport{response: successResponse},
		inState: func() ResponseChecker {
			var calls atomic.Int32
			return func(resp *Response) (done bool, err error) {
				val := calls.Add(1)
				// Stop the looping on the third invocation
				return val == 3, nil
			}
		}(),
		wantErr:   false,
		wantCalls: 3,
	}, {
		name:      "Retriable error is retried",
		transport: &fakeTransport{err: errRetriable, response: successResponse},
		inState: func(resp *Response) (done bool, err error) {
			return true, nil
		},
		wantErr:   false,
		wantCalls: 2,
	}, {
		name:      "Nonretriable error is not retried",
		transport: &fakeTransport{err: errNonRetriable, response: successResponse},
		inState: func(resp *Response) (done bool, err error) {
			return true, nil
		},
		wantErr:   true,
		wantCalls: 1,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := &SpoofingClient{
				Client:          &http.Client{Transport: tt.transport},
				Logf:            t.Logf,
				RequestInterval: 1,
				RequestTimeout:  time.Second,
			}
			url := &url.URL{
				Host:   "fake.knative.net",
				Scheme: "http",
			}
			_, err := sc.WaitForEndpointState(context.TODO(), url, tt.inState, "")
			if (err != nil) != tt.wantErr {
				t.Errorf("SpoofingClient.CheckEndpointState() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got, want := tt.transport.calls.Load(), tt.wantCalls; got != want {
				t.Errorf("Expected Transport to be invoked %d time but got invoked %d", want, got)
			}
		})
	}
}
