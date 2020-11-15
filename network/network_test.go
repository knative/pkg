/*
Copyright 2020 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package network

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsKubeletProbe(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "http://example.com/", nil)
	if err != nil {
		t.Fatal("Error building request:", err)
	}
	if IsKubeletProbe(req) {
		t.Error("Not a kubelet probe but counted as such")
	}
	req.Header.Set("User-Agent", KubeProbeUAPrefix+"1.14")
	if !IsKubeletProbe(req) {
		t.Error("kubelet probe but not counted as such")
	}
	req.Header.Del("User-Agent")
	if IsKubeletProbe(req) {
		t.Error("Not a kubelet probe but counted as such")
	}
	req.Header.Set(KubeletProbeHeaderName, "no matter")
	if !IsKubeletProbe(req) {
		t.Error("kubelet probe but not counted as such")
	}
}

func TestIsKnativeProbe(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "http://example.com/", nil)
	if err != nil {
		t.Fatal("Error building request:", err)
	}
	if IsKProbe(req) {
		t.Error("Not a knative probe but counted as such")
	}
	req.Header.Set("K-Network-Probe", "probe")
	if !IsKProbe(req) {
		t.Error("knative probe but not counted as such")
	}
	req.Header.Del("K-Network-Probe")
	if IsKProbe(req) {
		t.Error("Not a knative probe but counted as such")
	}
	req.Header.Set("K-Network-Probe", "no matter")
	if IsKProbe(req) {
		t.Error("Not a knative probe but not counted as such")
	}
}

func TestServeKProbe(t *testing.T) {
	var (
		kprobehash = "hash"
		kprobe     = &http.Request{
			Header: http.Header{
				"K-Network-Probe": []string{"probe"},
				"K-Network-Hash":  []string{kprobehash},
			},
		}
		kprobeerr = &http.Request{
			Header: http.Header{
				"K-Network-Probe": []string{"probe"},
			},
		}
	)

	resp := httptest.NewRecorder()
	ServeKProbe(resp, kprobe)
	if got, want := resp.Code, http.StatusOK; got != want {
		t.Errorf("Probe status = %d, wanted %d", got, want)
	}

	if got, want := resp.Header().Get("K-Network-Hash"), kprobehash; got != want {
		t.Errorf("KProbe hash = %s, wanted %s", got, want)
	}

	resp = httptest.NewRecorder()
	ServeKProbe(resp, kprobeerr)
	if got, want := resp.Code, http.StatusBadRequest; got != want {
		t.Errorf("Probe status = %d, wanted %d", got, want)
	}
}
