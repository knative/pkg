/*
Copyright 2018 The Knative Authors

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
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strings"
	"testing"
	"time"

	kubeclient "knative.dev/pkg/client/injection/kube/client/fake"
	_ "knative.dev/pkg/client/injection/kube/informers/core/v1/secret/fake"

	"github.com/mattbaird/jsonpatch"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/metrics/metricstest"

	. "knative.dev/pkg/webhook/testing"
)

const testTimeout = time.Duration(10 * time.Second)

type fixedAdmissionController struct {
	path     string
	response *admissionv1beta1.AdmissionResponse
}

var _ AdmissionController = (*fixedAdmissionController)(nil)

func (fac *fixedAdmissionController) Path() string {
	return fac.path
}

func (fac *fixedAdmissionController) Admit(ctx context.Context, req *admissionv1beta1.AdmissionRequest) *admissionv1beta1.AdmissionResponse {
	return fac.response
}

func TestMissingContentType(t *testing.T) {
	wh, serverURL, cancel, err := testSetup(t)
	if err != nil {
		t.Fatalf("testSetup() = %v", err)
	}
	defer cancel()
	stopCh := make(chan struct{})
	defer close(stopCh)

	go func() {
		err := wh.Run(stopCh)
		if err != nil {
			t.Errorf("Unable to run controller: %s", err)
		}
	}()

	pollErr := waitForServerAvailable(t, serverURL, testTimeout)
	if pollErr != nil {
		t.Fatalf("waitForServerAvailable() = %v", err)
	}

	tlsClient, err := createSecureTLSClient(t, wh.Client, &wh.Options)
	if err != nil {
		t.Fatalf("createSecureTLSClient() = %v", err)
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("https://%s", serverURL), nil)
	if err != nil {
		t.Fatalf("http.NewRequest() = %v", err)
	}

	response, err := tlsClient.Do(req)
	if err != nil {
		t.Fatalf("Received %v error from server %s", err, serverURL)
	}

	if got, want := response.StatusCode, http.StatusUnsupportedMediaType; got != want {
		t.Errorf("Response status code = %v, wanted %v", got, want)
	}

	defer response.Body.Close()
	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("Failed to read response body %v", err)
	}

	if !strings.Contains(string(responseBody), "invalid Content-Type") {
		t.Errorf("Response body to contain 'invalid Content-Type' , got = '%s'", string(responseBody))
	}

	// Stats are not reported for internal server errors
	metricstest.CheckStatsNotReported(t, requestCountName, requestLatenciesName)
}

func TestEmptyRequestBody(t *testing.T) {
	wh, serverURL, cancel, err := testSetup(t)
	if err != nil {
		t.Fatalf("testSetup() = %v", err)
	}
	defer cancel()

	stopCh := make(chan struct{})

	go func() {
		err := wh.Run(stopCh)
		if err != nil {
			t.Errorf("Unable to run controller: %s", err)
		}
	}()
	defer close(stopCh)

	pollErr := waitForServerAvailable(t, serverURL, testTimeout)
	if pollErr != nil {
		t.Fatalf("waitForServerAvailable() = %v", err)
	}

	tlsClient, err := createSecureTLSClient(t, wh.Client, &wh.Options)
	if err != nil {
		t.Fatalf("createSecureTLSClient() = %v", err)
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("https://%s", serverURL), nil)
	if err != nil {
		t.Fatalf("http.NewRequest() = %v", err)
	}

	req.Header.Add("Content-Type", "application/json")

	response, err := tlsClient.Do(req)
	if err != nil {
		t.Fatalf("failed to get resp %v", err)
	}

	if got, want := response.StatusCode, http.StatusBadRequest; got != want {
		t.Errorf("Response status code = %v, wanted %v", got, want)
	}
	defer response.Body.Close()

	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("Failed to read response body %v", err)
	}

	if !strings.Contains(string(responseBody), "could not decode body") {
		t.Errorf("Response body to contain 'decode failure information' , got = %q", string(responseBody))
	}
}

func TestValidResponseForResource(t *testing.T) {
	ac := &fixedAdmissionController{
		path:     "/bazinga",
		response: &admissionv1beta1.AdmissionResponse{},
	}
	wh, serverURL, cancel, err := testSetup(t, ac)
	if err != nil {
		t.Fatalf("testSetup() = %v", err)
	}
	defer cancel()

	stopCh := make(chan struct{})
	defer close(stopCh)

	go func() {
		err := wh.Run(stopCh)
		if err != nil {
			t.Errorf("Unable to run controller: %s", err)
		}
	}()

	pollErr := waitForServerAvailable(t, serverURL, testTimeout)
	if pollErr != nil {
		t.Fatalf("waitForServerAvailable() = %v", err)
	}
	tlsClient, err := createSecureTLSClient(t, wh.Client, &wh.Options)
	if err != nil {
		t.Fatalf("createSecureTLSClient() = %v", err)
	}

	admissionreq := &admissionv1beta1.AdmissionRequest{
		Operation: admissionv1beta1.Create,
		Kind: metav1.GroupVersionKind{
			Group:   "pkg.knative.dev",
			Version: "v1alpha1",
			Kind:    "Resource",
		},
	}
	testRev := CreateResource("testrev")
	marshaled, err := json.Marshal(testRev)
	if err != nil {
		t.Fatalf("Failed to marshal resource: %s", err)
	}

	admissionreq.Resource.Group = "pkg.knative.dev"
	admissionreq.Object.Raw = marshaled
	rev := &admissionv1beta1.AdmissionReview{
		Request: admissionreq,
	}

	reqBuf := new(bytes.Buffer)
	err = json.NewEncoder(reqBuf).Encode(&rev)
	if err != nil {
		t.Fatalf("Failed to marshal admission review: %v", err)
	}

	u, err := url.Parse(fmt.Sprintf("https://%s", serverURL))
	if err != nil {
		t.Fatalf("bad url %v", err)
	}

	u.Path = path.Join(u.Path, ac.Path())

	req, err := http.NewRequest("GET", u.String(), reqBuf)
	if err != nil {
		t.Fatalf("http.NewRequest() = %v", err)
	}
	req.Header.Add("Content-Type", "application/json")

	response, err := tlsClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to get response %v", err)
	}

	if got, want := response.StatusCode, http.StatusOK; got != want {
		t.Errorf("Response status code = %v, wanted %v", got, want)
	}

	defer response.Body.Close()
	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("Failed to read response body %v", err)
	}

	reviewResponse := admissionv1beta1.AdmissionReview{}

	err = json.NewDecoder(bytes.NewReader(responseBody)).Decode(&reviewResponse)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	metricstest.CheckStatsReported(t, requestCountName, requestLatenciesName)
}

func TestInvalidResponseForResource(t *testing.T) {
	expectedError := "everything is fine."
	ac := &fixedAdmissionController{
		path:     "/booger",
		response: MakeErrorStatus(expectedError),
	}
	wh, serverURL, cancel, err := testSetup(t, ac)
	if err != nil {
		t.Fatalf("testSetup() = %v", err)
	}
	defer cancel()

	stopCh := make(chan struct{})
	defer close(stopCh)

	go func() {
		err := wh.Run(stopCh)
		if err != nil {
			t.Errorf("Unable to run controller: %s", err)
		}
	}()

	pollErr := waitForServerAvailable(t, serverURL, testTimeout)
	if pollErr != nil {
		t.Fatalf("waitForServerAvailable() = %v", err)
	}
	tlsClient, err := createSecureTLSClient(t, wh.Client, &wh.Options)
	if err != nil {
		t.Fatalf("createSecureTLSClient() = %v", err)
	}

	resource := CreateResource(testResourceName)

	resource.Spec.FieldWithValidation = "not the right value"
	marshaled, err := json.Marshal(resource)
	if err != nil {
		t.Fatalf("Failed to marshal resource: %s", err)
	}

	admissionreq := &admissionv1beta1.AdmissionRequest{
		Operation: admissionv1beta1.Create,
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

	rev := &admissionv1beta1.AdmissionReview{
		Request: admissionreq,
	}
	reqBuf := new(bytes.Buffer)
	err = json.NewEncoder(reqBuf).Encode(&rev)
	if err != nil {
		t.Fatalf("Failed to marshal admission review: %v", err)
	}

	u, err := url.Parse(fmt.Sprintf("https://%s", serverURL))
	if err != nil {
		t.Fatalf("bad url %v", err)
	}

	u.Path = path.Join(u.Path, ac.Path())

	req, err := http.NewRequest("GET", u.String(), reqBuf)
	if err != nil {
		t.Fatalf("http.NewRequest() = %v", err)
	}

	req.Header.Add("Content-Type", "application/json")

	response, err := tlsClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to receive response %v", err)
	}

	if got, want := response.StatusCode, http.StatusOK; got != want {
		t.Errorf("Response status code = %v, wanted %v", got, want)
	}

	defer response.Body.Close()
	respBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("Failed to read response body %v", err)
	}

	reviewResponse := admissionv1beta1.AdmissionReview{}

	err = json.NewDecoder(bytes.NewReader(respBody)).Decode(&reviewResponse)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	var respPatch []jsonpatch.JsonPatchOperation
	err = json.Unmarshal(reviewResponse.Response.Patch, &respPatch)
	if err == nil {
		t.Fatalf("Expected to fail JSON unmarshal of resposnse")
	}

	if got, want := reviewResponse.Response.Result.Status, "Failure"; got != want {
		t.Errorf("Response status = %v, wanted %v", got, want)
	}

	if !strings.Contains(reviewResponse.Response.Result.Message, expectedError) {
		t.Errorf("Received unexpected response status message %s", reviewResponse.Response.Result.Message)
	}

	// Stats should be reported for requests that have admission disallowed
	metricstest.CheckStatsReported(t, requestCountName, requestLatenciesName)
}

func TestSetupWebhookHTTPServerError(t *testing.T) {
	defaultOpts := newDefaultOptions()
	defaultOpts.Port = -1 // invalid port
	ctx, wh, cancel := newNonRunningTestWebhook(t, defaultOpts)
	defer cancel()
	kubeClient := kubeclient.Get(ctx)

	nsErr := createNamespace(t, kubeClient, metav1.NamespaceSystem)
	if nsErr != nil {
		t.Fatalf("testSetup() = %v", nsErr)
	}
	cMapsErr := createTestConfigMap(t, kubeClient)
	if cMapsErr != nil {
		t.Fatalf("testSetup() = %v", cMapsErr)
	}

	stopCh := make(chan struct{})
	errCh := make(chan error)
	go func() {
		if err := wh.Run(stopCh); err != nil {
			errCh <- err
		}
	}()

	select {
	case <-time.After(6 * time.Second):
		t.Error("Timeout in testing bootstrap webhook http server failed")
	case errItem := <-errCh:
		if !strings.Contains(errItem.Error(), "bootstrap failed") {
			t.Error("Expected bootstrap webhook http server failed")
		}
	}
}

func testSetup(t *testing.T, acs ...AdmissionController) (*Webhook, string, context.CancelFunc, error) {
	t.Helper()
	port, err := newTestPort()
	if err != nil {
		return nil, "", nil, err
	}

	defaultOpts := newDefaultOptions()
	defaultOpts.Port = port
	_, wh, cancel := newNonRunningTestWebhook(t, defaultOpts, acs...)

	resetMetrics()
	return wh, fmt.Sprintf("0.0.0.0:%d", port), cancel, nil
}
