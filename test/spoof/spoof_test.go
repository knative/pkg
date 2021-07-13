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
	"fmt"
	"net/http"
	"net/url"

	"testing"
)

type fakeTransport struct{}

func (ft *fakeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		Status:     "200 ok",
		StatusCode: 200,
		Header:     http.Header{},
		Body:       http.NoBody,
	}, nil
}

type countCalls struct {
	calls int32
}

func (c *countCalls) count(rc ResponseChecker) ResponseChecker {
	return func(resp *Response) (done bool, err error) {
		c.calls++
		return rc(resp)
	}
}

func TestSpoofingClient_CheckEndpointState(t *testing.T) {
	type args struct {
		url     *url.URL
		inState ResponseChecker
		desc    string
		opts    []RequestOption
	}
	tests := []struct {
		name      string
		args      args
		wantErr   bool
		wantCalls int32
	}{{
		name: "Non matching response doesn't trigger a second check",
		args: args{
			url: &url.URL{
				Host:   "fake.knative.net",
				Scheme: "http",
			},
			inState: func(resp *Response) (done bool, err error) {
				return false, nil
			},
		},
		wantErr:   false,
		wantCalls: 1,
	}, {
		name: "Error response doesn't trigger a second check",
		args: args{
			url: &url.URL{
				Host:   "fake.knative.net",
				Scheme: "http",
			},
			inState: func(resp *Response) (done bool, err error) {
				return false, fmt.Errorf("response error")
			},
		},
		wantErr:   true,
		wantCalls: 1,
	}, {
		name: "OK response doesn't trigger a second check",
		args: args{
			url: &url.URL{
				Host:   "fake.knative.net",
				Scheme: "http",
			},
			inState: func(resp *Response) (done bool, err error) {
				return true, nil
			},
		},
		wantErr:   false,
		wantCalls: 1,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := &SpoofingClient{
				Client:          &http.Client{Transport: &fakeTransport{}},
				Logf:            t.Logf,
				RequestInterval: 1,
				RequestTimeout:  1,
			}
			counter := countCalls{}
			_, err := sc.CheckEndpointState(context.TODO(), tt.args.url, counter.count(tt.args.inState), tt.args.desc, tt.args.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("SpoofingClient.CheckEndpointState() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if counter.calls != tt.wantCalls {
				t.Errorf("Expected ResponseChecker to be invoked %d time but got invoked %d", tt.wantCalls, counter.calls)
			}
		})
	}
}

func TestSpoofingClient_WaitForEndpointState(t *testing.T) {
	type args struct {
		url     *url.URL
		inState ResponseChecker
		desc    string
		opts    []RequestOption
	}
	tests := []struct {
		name      string
		args      args
		wantErr   bool
		wantCalls int32
	}{{
		name: "OK response doesn't trigger a second request",
		args: args{
			url: &url.URL{
				Host:   "fake.knative.net",
				Scheme: "http",
			},
			inState: func(resp *Response) (done bool, err error) {
				return true, nil
			},
		},
		wantErr:   false,
		wantCalls: 1,
	}, {
		name: "Error response doesn't trigger more requests",
		args: args{
			url: &url.URL{
				Host:   "fake.knative.net",
				Scheme: "http",
			},
			inState: func(resp *Response) (done bool, err error) {
				return false, fmt.Errorf("response error")
			},
		},
		wantErr:   true,
		wantCalls: 1,
	}, {
		name: "Non matching response triggers more requests",
		args: args{
			url: &url.URL{
				Host:   "fake.knative.net",
				Scheme: "http",
			},
			inState: func(resp *Response) (done bool, err error) {
				return false, nil
			},
		},
		wantErr:   true,
		wantCalls: 3,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := &SpoofingClient{
				Client:          &http.Client{Transport: &fakeTransport{}},
				Logf:            t.Logf,
				RequestInterval: 1,
				RequestTimeout:  1,
			}
			counter := countCalls{}
			_, err := sc.WaitForEndpointState(context.TODO(), tt.args.url, counter.count(tt.args.inState), tt.args.desc, tt.args.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("SpoofingClient.CheckEndpointState() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if counter.calls != tt.wantCalls {
				t.Errorf("Expected ResponseChecker to be invoked %d time but got invoked %d", tt.wantCalls, counter.calls)
			}
		})
	}
}
