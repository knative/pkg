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

package boskos

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
)

var (
	fakeHost = "fakehost"
	fakeRes  = "{\"name\": \"res\", \"type\": \"t\", \"state\": \"d\"}"
)

// create a fake server as Boskos server, must close() afterwards
func fakeServer(f func(http.ResponseWriter, *http.Request)) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(f))
}

func TestAcquireGKEProject(t *testing.T) {
	datas := []struct {
		serverErr bool
		expErr    bool
	}{
		{true, true},
		{false, false},
	}
	oldBoskosURI := boskosURI
	defer func() {
		boskosURI = oldBoskosURI
	}()
	for _, data := range datas {
		ts := fakeServer(func(w http.ResponseWriter, r *http.Request) {
			if data.serverErr {
				http.Error(w, "", http.StatusBadRequest)
			} else {
				fmt.Fprint(w, fakeRes)
			}
		})
		defer ts.Close()
		boskosURI = ts.URL
		_, err := AcquireGKEProject(&fakeHost)
		if data.expErr && (nil == err) {
			log.Fatalf("testing acquiring GKE project, want: err, got: no err")
		}
		if !data.expErr && (nil != err) {
			log.Fatalf("testing acquiring GKE project, want: no err, got: err '%v'", err)
		}
	}
}

func TestReleaseGKEProject(t *testing.T) {
	datas := []struct {
		serverErr bool
		expErr    bool
	}{
		{true, true},
		{false, false},
	}
	oldBoskosURI := boskosURI
	defer func() {
		boskosURI = oldBoskosURI
	}()
	for _, data := range datas {
		ts := fakeServer(func(w http.ResponseWriter, r *http.Request) {
			if data.serverErr {
				http.Error(w, "", http.StatusBadRequest)
			} else {
				fmt.Fprint(w, "")
			}
		})
		defer ts.Close()
		boskosURI = ts.URL
		err := ReleaseGKEProject(&fakeHost, "foobar")
		if data.expErr && (nil == err) {
			log.Fatalf("testing acquiring GKE project, want: err, got: no err")
		}
		if !data.expErr && (nil != err) {
			log.Fatalf("testing acquiring GKE project, want: no err, got: err '%v'", err)
		}
	}
}
