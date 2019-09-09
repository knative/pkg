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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"knative.dev/pkg/testutils/common"
)

var (
	fakeHost = "fakehost"
	fakeRes  = "{\"name\": \"res\", \"type\": \"t\", \"state\": \"d\"}"

	client Client
)

func setup() {
	client = Client{}
}

// create a fake server as Boskos server, must close() afterwards
func fakeServer(f func(http.ResponseWriter, *http.Request)) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(f))
}

func TestAcquireGKEProject(t *testing.T) {
	mockJobName := "mockjobname"
	datas := []struct {
		serverErr bool
		host      *string
		expHost   string
		expErr    bool
	}{
		// Test boskos server error
		{true, &fakeHost, "fakehost", true},
		// Test passing host as param
		{false, &fakeHost, "fakehost", false},
		// Test using default host
		{false, nil, "mockjobname", false},
	}
	oldBoskosURI := boskosURI
	defer func() {
		boskosURI = oldBoskosURI
	}()
	oldGetOSEnv := common.GetOSEnv
	common.GetOSEnv = func(s string) string {
		if s == "JOB_NAME" {
			return mockJobName
		}
		return oldGetOSEnv(s)
	}
	defer func() {
		common.GetOSEnv = oldGetOSEnv
	}()
	for _, data := range datas {
		setup()
		ts := fakeServer(func(w http.ResponseWriter, r *http.Request) {
			if data.serverErr {
				http.Error(w, "", http.StatusBadRequest)
			} else {
				// RequestURI for acquire contains a random hash, doing
				// substring matching instead
				for _, s := range []string{"/acquire?", "owner=" + data.expHost, "state=free", "dest=busy", "type=gke-project"} {
					if !strings.Contains(r.RequestURI, s) {
						t.Fatalf("request URI doesn't match: want: contains '%s', got: '%s'", s, r.RequestURI)
					}
				}
				fmt.Fprint(w, fakeRes)
			}
		})
		defer ts.Close()
		boskosURI = ts.URL
		_, err := client.AcquireGKEProject(data.host)
		if data.expErr && (nil == err) {
			t.Fatalf("testing acquiring GKE project, want: err, got: no err")
		}
		if !data.expErr && (nil != err) {
			t.Fatalf("testing acquiring GKE project, want: no err, got: err '%v'", err)
		}
	}
}

func TestReleaseGKEProject(t *testing.T) {
	mockJobName := "mockjobname"
	datas := []struct {
		serverErr bool
		host      *string
		resName   string
		expReq    string
		expErr    bool
	}{
		// Test boskos server error
		{true, &fakeHost, "a", "/release?dest=dirty&name=a&owner=fakehost", true},
		// Test passing host as param
		{false, &fakeHost, "b", "/release?dest=dirty&name=b&owner=fakehost", false},
		// Test using default host
		{false, nil, "c", "/release?dest=dirty&name=c&owner=mockjobname", false},
	}
	oldBoskosURI := boskosURI
	defer func() {
		boskosURI = oldBoskosURI
	}()
	oldGetOSEnv := common.GetOSEnv
	common.GetOSEnv = func(s string) string {
		if s == "JOB_NAME" {
			return mockJobName
		}
		return oldGetOSEnv(s)
	}
	defer func() {
		common.GetOSEnv = oldGetOSEnv
	}()
	for _, data := range datas {
		setup()
		ts := fakeServer(func(w http.ResponseWriter, r *http.Request) {
			if data.serverErr {
				http.Error(w, "", http.StatusBadRequest)
			} else if r.RequestURI != data.expReq {
				t.Fatalf("request URI doesn't match: want: '%s', got: '%s'", data.expReq, r.RequestURI)
			} else {
				fmt.Fprint(w, "")
			}
		})
		defer ts.Close()
		boskosURI = ts.URL
		err := client.ReleaseGKEProject(data.host, data.resName)
		if data.expErr && (nil == err) {
			t.Fatalf("testing acquiring GKE project, want: err, got: no err")
		}
		if !data.expErr && (nil != err) {
			t.Fatalf("testing acquiring GKE project, want: no err, got: err '%v'", err)
		}
	}
}
