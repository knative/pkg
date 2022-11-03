/*
Copyright 2020 The Knative Authors

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
package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/sync/errgroup"
	jsonpatch "gomodules.xyz/jsonpatch/v2"
	admissionv1 "k8s.io/api/admission/v1"
	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
	kubeclient "knative.dev/pkg/client/injection/kube/client/fake"
	"knative.dev/pkg/metrics/metricstest"
	_ "knative.dev/pkg/metrics/testing"
)

type fixedAdmissionController struct {
	path     string
	response *admissionv1.AdmissionResponse
}

var _ AdmissionController = (*fixedAdmissionController)(nil)

func (fac *fixedAdmissionController) Path() string {
	return fac.path
}

func (fac *fixedAdmissionController) Admit(ctx context.Context, req *admissionv1.AdmissionRequest) *admissionv1.AdmissionResponse {
	r := apis.GetHTTPRequest(ctx)
	if r == nil {
		panic("nil request!")
	} else if r.URL.Path != fac.path {
		panic("wrong path!")
	}
	return fac.response
}

type readBodyTwiceAdmissionController struct {
	path     string
	response *admissionv1.AdmissionResponse
}

var _ AdmissionController = (*readBodyTwiceAdmissionController)(nil)

func (rbtac *readBodyTwiceAdmissionController) Path() string {
	return rbtac.path
}

func (rbtac *readBodyTwiceAdmissionController) Admit(ctx context.Context, req *admissionv1.AdmissionRequest) *admissionv1.AdmissionResponse {
	r := apis.GetHTTPRequest(ctx)
	if r == nil {
		panic("nil request!")
	} else if r.URL.Path != rbtac.path {
		panic("wrong path!")
	}

	var review admissionv1.AdmissionReview
	if err := json.NewDecoder(r.Body).Decode(&review); err != nil {
		panic("body closed!")
	}
	return rbtac.response
}

func TestAdmissionEmptyRequestBody(t *testing.T) {
	c := &fixedAdmissionController{
		path:     "/bazinga",
		response: &admissionv1.AdmissionResponse{},
	}

	testEmptyRequestBody(t, c)
}

func TestAdmissionValidResponseForResourceTLS(t *testing.T) {
	ac := &fixedAdmissionController{
		path:     "/bazinga",
		response: &admissionv1.AdmissionResponse{},
	}
	wh, serverURL, ctx, cancel, err := testSetup(t, ac)
	if err != nil {
		t.Fatal("testSetup() =", err)
	}

	eg, _ := errgroup.WithContext(ctx)
	eg.Go(func() error { return wh.Run(ctx.Done()) })
	defer func() {
		cancel()
		if err := eg.Wait(); err != nil {
			t.Error("Unable to run controller:", err)
		}
	}()

	pollErr := waitForServerAvailable(t, serverURL, testTimeout)
	if pollErr != nil {
		t.Fatal("waitForServerAvailable() =", err)
	}
	tlsClient, err := createSecureTLSClient(t, kubeclient.Get(ctx), &wh.Options)
	if err != nil {
		t.Fatal("createSecureTLSClient() =", err)
	}

	admissionreq := &admissionv1.AdmissionRequest{
		Operation: admissionv1.Create,
		Kind: metav1.GroupVersionKind{
			Group:   "pkg.knative.dev",
			Version: "v1alpha1",
			Kind:    "Resource",
		},
	}
	testRev := createResource("testrev")
	marshaled, err := json.Marshal(testRev)
	if err != nil {
		t.Fatal("Failed to marshal resource:", err)
	}

	admissionreq.Resource.Group = "pkg.knative.dev"
	admissionreq.Object.Raw = marshaled
	rev := &admissionv1.AdmissionReview{
		Request: admissionreq,
	}

	reqBuf := new(bytes.Buffer)
	err = json.NewEncoder(reqBuf).Encode(&rev)
	if err != nil {
		t.Fatal("Failed to marshal admission review:", err)
	}

	u, err := url.Parse("https://" + serverURL)
	if err != nil {
		t.Fatal("bad url", err)
	}

	u.Path = path.Join(u.Path, ac.Path())

	req, err := http.NewRequest("GET", u.String(), reqBuf)
	if err != nil {
		t.Fatal("http.NewRequest() =", err)
	}
	req.Header.Add("Content-Type", "application/json")

	doneCh := make(chan struct{})
	launchedCh := make(chan struct{})
	go func() {
		defer close(doneCh)

		close(launchedCh)
		response, err := tlsClient.Do(req)
		if err != nil {
			t.Error("Failed to get response", err)
			return
		}

		if got, want := response.StatusCode, http.StatusOK; got != want {
			t.Errorf("Response status code = %v, wanted %v", got, want)
			return
		}

		defer response.Body.Close()
		responseBody, err := io.ReadAll(response.Body)
		if err != nil {
			t.Error("Failed to read response body", err)
			return
		}

		reviewResponse := admissionv1.AdmissionReview{}

		err = json.NewDecoder(bytes.NewReader(responseBody)).Decode(&reviewResponse)
		if err != nil {
			t.Error("Failed to decode response:", err)
			return
		}

		if diff := cmp.Diff(rev.TypeMeta, reviewResponse.TypeMeta); diff != "" {
			t.Errorf("expected the response typeMeta to be the same as the request (-want, +got)\n%s", diff)
			return
		}
	}()

	// Wait for the goroutine to launch.
	<-launchedCh

	// Check that Admit calls block when they are initiated before informers sync.
	select {
	case <-time.After(100 * time.Millisecond):
	case <-doneCh:
		t.Fatal("Admit was called before informers had synced.")
	}

	// Signal the webhook that informers have synced.
	wh.InformersHaveSynced()

	// Check that after informers have synced that things start completing immediately (including outstanding requests).
	select {
	case <-doneCh:
	case <-time.After(5 * time.Second):
		t.Error("Timed out waiting on Admit to complete after informers synced.")
	}

	metricstest.CheckStatsReported(t, requestCountName, requestLatenciesName)
}

func TestAdmissionValidResponseForResource(t *testing.T) {
	ac := &fixedAdmissionController{
		path:     "/bazinga",
		response: &admissionv1.AdmissionResponse{},
	}
	wh, serverURL, ctx, cancel, err := testSetupNoTLS(t, ac)
	if err != nil {
		t.Fatal("testSetup() =", err)
	}

	eg, _ := errgroup.WithContext(ctx)
	eg.Go(func() error { return wh.Run(ctx.Done()) })
	defer func() {
		cancel()
		if err := eg.Wait(); err != nil {
			t.Error("Unable to run controller:", err)
		}
	}()

	pollErr := waitForServerAvailable(t, serverURL, testTimeout)
	if pollErr != nil {
		t.Fatal("waitForServerAvailable() =", err)
	}
	client := createNonTLSClient()

	admissionreq := &admissionv1.AdmissionRequest{
		Operation: admissionv1.Create,
		Kind: metav1.GroupVersionKind{
			Group:   "pkg.knative.dev",
			Version: "v1alpha1",
			Kind:    "Resource",
		},
	}
	testRev := createResource("testrev")
	marshaled, err := json.Marshal(testRev)
	if err != nil {
		t.Fatal("Failed to marshal resource:", err)
	}

	admissionreq.Resource.Group = "pkg.knative.dev"
	admissionreq.Object.Raw = marshaled
	rev := &admissionv1.AdmissionReview{
		Request: admissionreq,
	}

	reqBuf := new(bytes.Buffer)
	err = json.NewEncoder(reqBuf).Encode(&rev)
	if err != nil {
		t.Fatal("Failed to marshal admission review:", err)
	}

	u, err := url.Parse("http://" + serverURL)
	if err != nil {
		t.Fatal("bad url", err)
	}

	u.Path = path.Join(u.Path, ac.Path())

	req, err := http.NewRequest("GET", u.String(), reqBuf)
	if err != nil {
		t.Fatal("http.NewRequest() =", err)
	}
	req.Header.Add("Content-Type", "application/json")

	doneCh := make(chan struct{})
	launchedCh := make(chan struct{})
	go func() {
		defer close(doneCh)

		close(launchedCh)
		response, err := client.Do(req)
		if err != nil {
			t.Error("Failed to get response", err)
			return
		}

		if got, want := response.StatusCode, http.StatusOK; got != want {
			t.Errorf("Response status code = %v, wanted %v", got, want)
			return
		}

		defer response.Body.Close()
		responseBody, err := io.ReadAll(response.Body)
		if err != nil {
			t.Error("Failed to read response body", err)
			return
		}

		reviewResponse := admissionv1.AdmissionReview{}

		err = json.NewDecoder(bytes.NewReader(responseBody)).Decode(&reviewResponse)
		if err != nil {
			t.Error("Failed to decode response:", err)
			return
		}

		if diff := cmp.Diff(rev.TypeMeta, reviewResponse.TypeMeta); diff != "" {
			t.Errorf("expected the response typeMeta to be the same as the request (-want, +got)\n%s", diff)
			return
		}
	}()

	// Wait for the goroutine to launch.
	<-launchedCh

	// Check that Admit calls block when they are initiated before informers sync.
	select {
	case <-time.After(100 * time.Millisecond):
	case <-doneCh:
		t.Fatal("Admit was called before informers had synced.")
	}

	// Signal the webhook that informers have synced.
	wh.InformersHaveSynced()

	// Check that after informers have synced that things start completing immediately (including outstanding requests).
	select {
	case <-doneCh:
	case <-time.After(5 * time.Second):
		t.Error("Timed out waiting on Admit to complete after informers synced.")
	}

	metricstest.CheckStatsReported(t, requestCountName, requestLatenciesName)
}

func TestAdmissionInvalidResponseForResource(t *testing.T) {
	expectedError := "everything is fine."
	ac := &fixedAdmissionController{
		path:     "/booger",
		response: MakeErrorStatus(expectedError),
	}
	wh, serverURL, ctx, cancel, err := testSetup(t, ac)
	if err != nil {
		t.Fatal("testSetup() =", err)
	}

	eg, _ := errgroup.WithContext(ctx)
	eg.Go(func() error { return wh.Run(ctx.Done()) })
	wh.InformersHaveSynced()
	defer func() {
		cancel()
		if err := eg.Wait(); err != nil {
			t.Error("Unable to run controller:", err)
		}
	}()

	pollErr := waitForServerAvailable(t, serverURL, testTimeout)
	if pollErr != nil {
		t.Fatal("waitForServerAvailable() =", err)
	}
	tlsClient, err := createSecureTLSClient(t, kubeclient.Get(ctx), &wh.Options)
	if err != nil {
		t.Fatal("createSecureTLSClient() =", err)
	}

	resource := createResource(testResourceName)

	resource.Spec.FieldWithValidation = "not the right value"
	marshaled, err := json.Marshal(resource)
	if err != nil {
		t.Fatal("Failed to marshal resource:", err)
	}

	admissionreq := &admissionv1.AdmissionRequest{
		Operation: admissionv1.Create,
		Kind: metav1.GroupVersionKind{
			Group:   "pkg.knative.dev",
			Version: "v1alpha1",
			Kind:    "Resource",
		},
		UserInfo: authenticationv1.UserInfo{
			Username: user1,
		},
	}

	admissionreq.Resource.Group = "pkg.knative.dev"
	admissionreq.Object.Raw = marshaled

	rev := &admissionv1.AdmissionReview{
		Request: admissionreq,
	}
	reqBuf := new(bytes.Buffer)
	err = json.NewEncoder(reqBuf).Encode(&rev)
	if err != nil {
		t.Fatal("Failed to marshal admission review:", err)
	}

	u, err := url.Parse("https://" + serverURL)
	if err != nil {
		t.Fatal("bad url", err)
	}

	u.Path = path.Join(u.Path, ac.Path())

	req, err := http.NewRequest("GET", u.String(), reqBuf)
	if err != nil {
		t.Fatal("http.NewRequest() =", err)
	}

	req.Header.Add("Content-Type", "application/json")

	response, err := tlsClient.Do(req)
	if err != nil {
		t.Fatal("Failed to receive response", err)
	}

	if got, want := response.StatusCode, http.StatusOK; got != want {
		t.Errorf("Response status code = %v, wanted %v", got, want)
	}

	defer response.Body.Close()
	respBody, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal("Failed to read response body", err)
	}

	reviewResponse := admissionv1.AdmissionReview{}

	err = json.NewDecoder(bytes.NewReader(respBody)).Decode(&reviewResponse)
	if err != nil {
		t.Fatal("Failed to decode response:", err)
	}

	var respPatch []jsonpatch.JsonPatchOperation
	err = json.Unmarshal(reviewResponse.Response.Patch, &respPatch)
	if err == nil {
		t.Fatalf("Expected to fail JSON unmarshal of response")
	}

	if got, want := reviewResponse.Response.Result.Status, "Failure"; got != want {
		t.Errorf("Response status = %v, wanted %v", got, want)
	}

	if !strings.Contains(reviewResponse.Response.Result.Message, expectedError) {
		t.Error("Received unexpected response status message", reviewResponse.Response.Result.Message)
	}

	// Stats should be reported for requests that have admission disallowed
	metricstest.CheckStatsReported(t, requestCountName, requestLatenciesName)
}

func TestAdmissionWarningResponseForResource(t *testing.T) {
	// Test that our single warning below (with newlines) should be turned into
	// these three warnings
	expectedWarnings := []string{"everything is not fine.", "like really", "for sure"}
	ac := &fixedAdmissionController{
		path:     "/warnmeplease",
		response: &admissionv1.AdmissionResponse{Warnings: []string{"everything is not fine.\nlike really\nfor sure"}},
	}
	wh, serverURL, ctx, cancel, err := testSetup(t, ac)
	if err != nil {
		t.Fatal("testSetup() =", err)
	}

	eg, _ := errgroup.WithContext(ctx)
	eg.Go(func() error { return wh.Run(ctx.Done()) })
	wh.InformersHaveSynced()
	defer func() {
		cancel()
		if err := eg.Wait(); err != nil {
			t.Error("Unable to run controller:", err)
		}
	}()

	pollErr := waitForServerAvailable(t, serverURL, testTimeout)
	if pollErr != nil {
		t.Fatal("waitForServerAvailable() =", err)
	}
	tlsClient, err := createSecureTLSClient(t, kubeclient.Get(ctx), &wh.Options)
	if err != nil {
		t.Fatal("createSecureTLSClient() =", err)
	}

	resource := createResource(testResourceName)

	marshaled, err := json.Marshal(resource)
	if err != nil {
		t.Fatal("Failed to marshal resource:", err)
	}

	admissionreq := &admissionv1.AdmissionRequest{
		Operation: admissionv1.Create,
		Kind: metav1.GroupVersionKind{
			Group:   "pkg.knative.dev",
			Version: "v1alpha1",
			Kind:    "Resource",
		},
		UserInfo: authenticationv1.UserInfo{
			Username: user1,
		},
	}

	admissionreq.Resource.Group = "pkg.knative.dev"
	admissionreq.Object.Raw = marshaled

	rev := &admissionv1.AdmissionReview{
		Request: admissionreq,
	}
	reqBuf := new(bytes.Buffer)
	err = json.NewEncoder(reqBuf).Encode(&rev)
	if err != nil {
		t.Fatal("Failed to marshal admission review:", err)
	}

	u, err := url.Parse("https://" + serverURL)
	if err != nil {
		t.Fatal("bad url", err)
	}

	u.Path = path.Join(u.Path, ac.Path())

	req, err := http.NewRequest("GET", u.String(), reqBuf)
	if err != nil {
		t.Fatal("http.NewRequest() =", err)
	}

	req.Header.Add("Content-Type", "application/json")

	response, err := tlsClient.Do(req)
	if err != nil {
		t.Fatal("Failed to receive response", err)
	}

	if got, want := response.StatusCode, http.StatusOK; got != want {
		t.Errorf("Response status code = %v, wanted %v", got, want)
	}

	defer response.Body.Close()
	respBody, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal("Failed to read response body", err)
	}

	reviewResponse := admissionv1.AdmissionReview{}

	err = json.NewDecoder(bytes.NewReader(respBody)).Decode(&reviewResponse)
	if err != nil {
		t.Fatal("Failed to decode response:", err)
	}

	warnings := reviewResponse.Response.Warnings
	if len(warnings) != 3 {
		t.Errorf("Received unexpected warnings, wanted 3 got: %s", reviewResponse.Response.Warnings)
	}
	for i, w := range warnings {
		if expectedWarnings[i] != w {
			t.Errorf("Unexpected warning want %s got %s", expectedWarnings[i], w)
		}
	}
}

func TestAdmissionValidResponseForRequestBody(t *testing.T) {
	ac := &readBodyTwiceAdmissionController{
		path:     "/bazinga",
		response: &admissionv1.AdmissionResponse{},
	}
	wh, serverURL, ctx, cancel, err := testSetupNoTLS(t, ac)
	if err != nil {
		t.Fatal("testSetup() =", err)
	}

	eg, _ := errgroup.WithContext(ctx)
	eg.Go(func() error { return wh.Run(ctx.Done()) })
	defer func() {
		cancel()
		if err := eg.Wait(); err != nil {
			t.Error("Unable to run controller:", err)
		}
	}()

	pollErr := waitForServerAvailable(t, serverURL, testTimeout)
	if pollErr != nil {
		t.Fatal("waitForServerAvailable() =", err)
	}
	client := createNonTLSClient()

	admissionreq := &admissionv1.AdmissionRequest{
		Operation: admissionv1.Create,
		Kind: metav1.GroupVersionKind{
			Group:   "pkg.knative.dev",
			Version: "v1alpha1",
			Kind:    "Resource",
		},
	}
	testRev := createResource("testrev")
	marshaled, err := json.Marshal(testRev)
	if err != nil {
		t.Fatal("Failed to marshal resource:", err)
	}

	admissionreq.Resource.Group = "pkg.knative.dev"
	admissionreq.Object.Raw = marshaled
	rev := &admissionv1.AdmissionReview{
		Request: admissionreq,
	}

	reqBuf := new(bytes.Buffer)
	err = json.NewEncoder(reqBuf).Encode(&rev)
	if err != nil {
		t.Fatal("Failed to marshal admission review:", err)
	}

	u, err := url.Parse("http://" + serverURL)
	if err != nil {
		t.Fatal("bad url", err)
	}

	u.Path = path.Join(u.Path, ac.Path())

	req, err := http.NewRequest("GET", u.String(), reqBuf)
	if err != nil {
		t.Fatal("http.NewRequest() =", err)
	}
	req.Header.Add("Content-Type", "application/json")

	doneCh := make(chan struct{})
	launchedCh := make(chan struct{})
	go func() {
		defer close(doneCh)

		close(launchedCh)
		response, err := client.Do(req)
		if err != nil {
			t.Error("Failed to get response", err)
			return
		}

		if got, want := response.StatusCode, http.StatusOK; got != want {
			t.Errorf("Response status code = %v, wanted %v", got, want)
			return
		}

		defer response.Body.Close()
		responseBody, err := io.ReadAll(response.Body)
		if err != nil {
			t.Error("Failed to read response body", err)
			return
		}

		reviewResponse := admissionv1.AdmissionReview{}

		err = json.NewDecoder(bytes.NewReader(responseBody)).Decode(&reviewResponse)
		if err != nil {
			t.Error("Failed to decode response:", err)
			return
		}

		if diff := cmp.Diff(rev.TypeMeta, reviewResponse.TypeMeta); diff != "" {
			t.Errorf("expected the response typeMeta to be the same as the request (-want, +got)\n%s", diff)
			return
		}
	}()

	// Wait for the goroutine to launch.
	<-launchedCh

	// Check that Admit calls block when they are initiated before informers sync.
	select {
	case <-time.After(100 * time.Millisecond):
	case <-doneCh:
		t.Fatal("Admit was called before informers had synced.")
	}

	// Signal the webhook that informers have synced.
	wh.InformersHaveSynced()

	// Check that after informers have synced that things start completing immediately (including outstanding requests).
	select {
	case <-doneCh:
	case <-time.After(5 * time.Second):
		t.Error("Timed out waiting on Admit to complete after informers synced.")
	}

	metricstest.CheckStatsReported(t, requestCountName, requestLatenciesName)
}
