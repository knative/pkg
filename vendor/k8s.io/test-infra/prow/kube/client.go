/*
Copyright 2016 The Kubernetes Authors.

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

package kube

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"

	prowapi "k8s.io/test-infra/prow/apis/prowjobs/v1"
)

const (
	// TestContainerName specifies the primary container name.
	TestContainerName = "test"

	inClusterBaseURL = "https://kubernetes.default"
	maxRetries       = 8
	retryDelay       = 2 * time.Second
	requestTimeout   = time.Minute

	// EmptySelector selects everything
	EmptySelector = ""

	// DefaultClusterAlias specifies the default context for resources owned by jobs (pods/builds).
	DefaultClusterAlias = "default" // TODO(fejta): rename to context
	// InClusterContext specifies the context for prowjob resources.
	InClusterContext = ""
)

// newClient is used to allow mocking out the behavior of 'NewClient' while testing.
var newClient = NewClient

// Logger can print debug messages
type Logger interface {
	Debugf(s string, v ...interface{})
}

// Client interacts with the Kubernetes api-server.
type Client struct {
	// If logger is non-nil, log all method calls with it.
	logger Logger

	baseURL   string
	deckURL   string
	client    *http.Client
	token     string
	namespace string
	fake      bool
}

// Namespace returns a copy of the client pointing at the specified namespace.
func (c *Client) Namespace(ns string) *Client {
	nc := *c
	nc.namespace = ns
	return &nc
}

func (c *Client) log(methodName string, args ...interface{}) {
	if c.logger == nil {
		return
	}
	var as []string
	for _, arg := range args {
		as = append(as, fmt.Sprintf("%v", arg))
	}
	c.logger.Debugf("%s(%s)", methodName, strings.Join(as, ", "))
}

// ConflictError is http 409.
type ConflictError struct {
	e error
}

func (e ConflictError) Error() string {
	return e.e.Error()
}

// NewConflictError returns an error with the embedded inner error
func NewConflictError(e error) ConflictError {
	return ConflictError{e: e}
}

// UnprocessableEntityError happens when the apiserver returns http 422.
type UnprocessableEntityError struct {
	e error
}

func (e UnprocessableEntityError) Error() string {
	return e.e.Error()
}

// NewUnprocessableEntityError returns an error with the embedded inner error
func NewUnprocessableEntityError(e error) UnprocessableEntityError {
	return UnprocessableEntityError{e: e}
}

// NotFoundError happens when the apiserver returns http 404
type NotFoundError struct {
	e error
}

func (e NotFoundError) Error() string {
	return e.e.Error()
}

// NewNotFoundError returns an error with the embedded inner error
func NewNotFoundError(e error) NotFoundError {
	return NotFoundError{e: e}
}

type request struct {
	method      string
	path        string
	deckPath    string
	contentType string
	query       map[string]string
	requestBody interface{}
}

func (c *Client) request(r *request, ret interface{}) error {
	out, err := c.requestRetry(r)
	if err != nil {
		return err
	}
	if ret != nil {
		if err := json.Unmarshal(out, ret); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) retry(r *request) (*http.Response, error) {
	var resp *http.Response
	var err error
	backoff := retryDelay
	for retries := 0; retries < maxRetries; retries++ {
		resp, err = c.doRequest(r.method, r.deckPath, r.path, r.contentType, r.query, r.requestBody)
		if err == nil {
			if resp.StatusCode < 500 {
				break
			}
			resp.Body.Close()
		}

		time.Sleep(backoff)
		backoff *= 2
	}
	return resp, err
}

// Retry on transport failures. Does not retry on 500s.
func (c *Client) requestRetryStream(r *request) (io.ReadCloser, error) {
	if c.fake && r.deckPath == "" {
		return nil, nil
	}
	resp, err := c.retry(r)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == 409 {
		return nil, NewConflictError(fmt.Errorf("body cannot be streamed"))
	} else if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("response has status \"%s\"", resp.Status)
	}
	return resp.Body, nil
}

// Retry on transport failures. Does not retry on 500s.
func (c *Client) requestRetry(r *request) ([]byte, error) {
	if c.fake && r.deckPath == "" {
		return []byte("{}"), nil
	}
	resp, err := c.retry(r)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	rb, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == 404 {
		return nil, NewNotFoundError(fmt.Errorf("body: %s", string(rb)))
	} else if resp.StatusCode == 409 {
		return nil, NewConflictError(fmt.Errorf("body: %s", string(rb)))
	} else if resp.StatusCode == 422 {
		return nil, NewUnprocessableEntityError(fmt.Errorf("body: %s", string(rb)))
	} else if resp.StatusCode == 404 {
		return nil, NewNotFoundError(fmt.Errorf("body: %s", string(rb)))
	} else if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("response has status \"%s\" and body \"%s\"", resp.Status, string(rb))
	}
	return rb, nil
}

func (c *Client) doRequest(method, deckPath, urlPath, contentType string, query map[string]string, body interface{}) (*http.Response, error) {
	url := c.baseURL + urlPath
	if c.deckURL != "" && deckPath != "" {
		url = c.deckURL + deckPath
	}
	var buf io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		buf = bytes.NewBuffer(b)
	}
	req, err := http.NewRequest(method, url, buf)
	if err != nil {
		return nil, err
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	} else {
		req.Header.Set("Content-Type", "application/json")
	}

	q := req.URL.Query()
	for k, v := range query {
		q.Add(k, v)
	}
	req.URL.RawQuery = q.Encode()

	return c.client.Do(req)
}

// NewFakeClient creates a client that doesn't do anything. If you provide a
// deck URL then the client will hit that for the supported calls.
func NewFakeClient(deckURL string) *Client {
	return &Client{
		namespace: "default",
		deckURL:   deckURL,
		client:    &http.Client{},
		fake:      true,
	}
}

// NewClientInCluster creates a Client that works from within a pod.
func NewClientInCluster(namespace string) (*Client, error) {
	tokenFile := "/var/run/secrets/kubernetes.io/serviceaccount/token"
	token, err := ioutil.ReadFile(tokenFile)
	if err != nil {
		return nil, err
	}

	rootCAFile := "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
	certData, err := ioutil.ReadFile(rootCAFile)
	if err != nil {
		return nil, err
	}

	cp := x509.NewCertPool()
	cp.AppendCertsFromPEM(certData)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
			RootCAs:    cp,
		},
	}
	return &Client{
		logger:    logrus.WithField("client", "kube"),
		baseURL:   inClusterBaseURL,
		client:    &http.Client{Transport: tr, Timeout: requestTimeout},
		token:     string(token),
		namespace: namespace,
	}, nil
}

// Cluster represents the information necessary to talk to a Kubernetes
// master endpoint.
// NOTE: if your cluster runs on GKE you can use the following command to get these credentials:
// gcloud --project <gcp_project> container clusters describe --zone <zone> <cluster_name>
type Cluster struct {
	// The IP address of the cluster's master endpoint.
	Endpoint string `json:"endpoint"`
	// Base64-encoded public cert used by clients to authenticate to the
	// cluster endpoint.
	ClientCertificate []byte `json:"clientCertificate"`
	// Base64-encoded private key used by clients..
	ClientKey []byte `json:"clientKey"`
	// Base64-encoded public certificate that is the root of trust for the
	// cluster.
	ClusterCACertificate []byte `json:"clusterCaCertificate"`
}

// NewClientFromFile reads a Cluster object at clusterPath and returns an
// authenticated client using the keys within.
func NewClientFromFile(clusterPath, namespace string) (*Client, error) {
	data, err := ioutil.ReadFile(clusterPath)
	if err != nil {
		return nil, err
	}
	var c Cluster
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return NewClient(&c, namespace)
}

// UnmarshalClusterMap reads a map[string]Cluster in yaml bytes.
func UnmarshalClusterMap(data []byte) (map[string]Cluster, error) {
	var raw map[string]Cluster
	if err := yaml.Unmarshal(data, &raw); err != nil {
		// If we failed to unmarshal the multicluster format try the single Cluster format.
		var singleConfig Cluster
		if err := yaml.Unmarshal(data, &singleConfig); err != nil {
			return nil, err
		}
		raw = map[string]Cluster{DefaultClusterAlias: singleConfig}
	}
	return raw, nil
}

// MarshalClusterMap writes c as yaml bytes.
func MarshalClusterMap(c map[string]Cluster) ([]byte, error) {
	return yaml.Marshal(c)
}

// ClientMapFromFile reads the file at clustersPath and attempts to load a map of cluster aliases
// to authenticated clients to the respective clusters.
// The file at clustersPath is expected to be a yaml map from strings to Cluster structs OR it may
// simply be a single Cluster struct which will be assigned the alias $DefaultClusterAlias.
// If the file is an alias map, it must include the alias $DefaultClusterAlias.
func ClientMapFromFile(clustersPath, namespace string) (map[string]*Client, error) {
	data, err := ioutil.ReadFile(clustersPath)
	if err != nil {
		return nil, fmt.Errorf("read error: %v", err)
	}
	raw, err := UnmarshalClusterMap(data)
	if err != nil {
		return nil, fmt.Errorf("unmarshal error: %v", err)
	}
	foundDefault := false
	result := map[string]*Client{}
	for alias, config := range raw {
		client, err := newClient(&config, namespace)
		if err != nil {
			return nil, fmt.Errorf("failed to load config for build cluster alias %q in file %q: %v", alias, clustersPath, err)
		}
		result[alias] = client
		if alias == DefaultClusterAlias {
			foundDefault = true
		}
	}
	if !foundDefault {
		return nil, fmt.Errorf("failed to find the required %q alias in build cluster config %q", DefaultClusterAlias, clustersPath)
	}
	return result, nil
}

// NewClient returns an authenticated Client using the keys in the Cluster.
func NewClient(c *Cluster, namespace string) (*Client, error) {
	// Relies on json encoding/decoding []byte as base64
	// https://golang.org/pkg/encoding/json/#Marshal
	cc := c.ClientCertificate
	ck := c.ClientKey
	ca := c.ClusterCACertificate

	cert, err := tls.X509KeyPair(cc, ck)
	if err != nil {
		return nil, err
	}

	cp := x509.NewCertPool()
	cp.AppendCertsFromPEM(ca)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion:   tls.VersionTLS12,
			Certificates: []tls.Certificate{cert},
			RootCAs:      cp,
		},
	}
	return &Client{
		logger:    logrus.WithField("client", "kube"),
		baseURL:   c.Endpoint,
		client:    &http.Client{Transport: tr, Timeout: requestTimeout},
		namespace: namespace,
	}, nil
}

// GetPod is analogous to kubectl get pods/NAME namespace=client.namespace
func (c *Client) GetPod(name string) (v1.Pod, error) {
	c.log("GetPod", name)
	var retPod v1.Pod
	err := c.request(&request{
		path: fmt.Sprintf("/api/v1/namespaces/%s/pods/%s", c.namespace, name),
	}, &retPod)
	return retPod, err
}

// ListPods is analogous to kubectl get pods --selector=SELECTOR --namespace=client.namespace
func (c *Client) ListPods(selector string) ([]v1.Pod, error) {
	c.log("ListPods", selector)
	var pl struct {
		Items []v1.Pod `json:"items"`
	}
	err := c.request(&request{
		path:  fmt.Sprintf("/api/v1/namespaces/%s/pods", c.namespace),
		query: map[string]string{"labelSelector": selector},
	}, &pl)
	return pl.Items, err
}

// DeletePod deletes the pod at name in the client's specified namespace.
//
// Analogous to kubectl delete pod --namespace=client.namespace
func (c *Client) DeletePod(name string) error {
	c.log("DeletePod", name)
	return c.request(&request{
		method: http.MethodDelete,
		path:   fmt.Sprintf("/api/v1/namespaces/%s/pods/%s", c.namespace, name),
	}, nil)
}

// CreateProwJob creates a prowjob in the client's specified namespace.
//
// Analogous to kubectl create prowjob --namespace=client.namespace
func (c *Client) CreateProwJob(j prowapi.ProwJob) (prowapi.ProwJob, error) {
	var representation string
	if out, err := json.Marshal(j); err == nil {
		representation = string(out[:])
	} else {
		representation = fmt.Sprintf("%v", j)
	}
	c.log("CreateProwJob", representation)
	var retJob prowapi.ProwJob
	err := c.request(&request{
		method:      http.MethodPost,
		path:        fmt.Sprintf("/apis/prow.k8s.io/v1/namespaces/%s/prowjobs", c.namespace),
		requestBody: &j,
	}, &retJob)
	return retJob, err
}

// GetProwJob returns the prowjob at name in the client's specified namespace.
//
// Analogous to kubectl get prowjob/NAME --namespace=client.namespace
func (c *Client) GetProwJob(name string) (prowapi.ProwJob, error) {
	c.log("GetProwJob", name)
	var pj prowapi.ProwJob
	err := c.request(&request{
		path: fmt.Sprintf("/apis/prow.k8s.io/v1/namespaces/%s/prowjobs/%s", c.namespace, name),
	}, &pj)
	return pj, err
}

// ListProwJobs lists prowjobs using the specified labelSelector in the client's specified namespace.
//
// Analogous to kubectl get prowjobs --selector=SELECTOR --namespace=client.namespace
func (c *Client) ListProwJobs(selector string) ([]prowapi.ProwJob, error) {
	c.log("ListProwJobs", selector)
	var jl struct {
		Items []prowapi.ProwJob `json:"items"`
	}
	err := c.request(&request{
		path:     fmt.Sprintf("/apis/prow.k8s.io/v1/namespaces/%s/prowjobs", c.namespace),
		deckPath: "/prowjobs.js",
		query:    map[string]string{"labelSelector": selector},
	}, &jl)
	if err == nil {
		var pjs []prowapi.ProwJob
		for _, pj := range jl.Items {
			pjs = append(pjs, pj)
		}
		jl.Items = pjs
	}
	return jl.Items, err
}

// DeleteProwJob deletes the prowjob at name in the client's specified namespace.
//
// Analogous to kubectl delete prowjob/NAME --namespace=client.namespace
func (c *Client) DeleteProwJob(name string) error {
	c.log("DeleteProwJob", name)
	return c.request(&request{
		method: http.MethodDelete,
		path:   fmt.Sprintf("/apis/prow.k8s.io/v1/namespaces/%s/prowjobs/%s", c.namespace, name),
	}, nil)
}

// ReplaceProwJob will replace name with job in the client's specified namespace.
//
// Analogous to kubectl replace prowjobs/NAME --namespace=client.namespace
func (c *Client) ReplaceProwJob(name string, job prowapi.ProwJob) (prowapi.ProwJob, error) {
	c.log("ReplaceProwJob", name, job)
	var retJob prowapi.ProwJob
	err := c.request(&request{
		method:      http.MethodPut,
		path:        fmt.Sprintf("/apis/prow.k8s.io/v1/namespaces/%s/prowjobs/%s", c.namespace, name),
		requestBody: &job,
	}, &retJob)
	return retJob, err
}

// CreatePod creates a pod in the client's specified namespace.
//
// Analogous to kubectl create pod --namespace=client.namespace
func (c *Client) CreatePod(p v1.Pod) (v1.Pod, error) {
	c.log("CreatePod", p)
	var retPod v1.Pod
	err := c.request(&request{
		method:      http.MethodPost,
		path:        fmt.Sprintf("/api/v1/namespaces/%s/pods", c.namespace),
		requestBody: &p,
	}, &retPod)
	return retPod, err
}

// GetLog returns the log of the default container in the specified pod, in the client's specified namespace.
//
// Analogous to kubectl logs pod --namespace=client.namespace
func (c *Client) GetLog(pod string) ([]byte, error) {
	c.log("GetLog", pod)
	return c.requestRetry(&request{
		path: fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/log", c.namespace, pod),
	})
}

// GetLogTail returns the last n bytes of the log of the specified container in the specified pod,
// in the client's specified namespace.
//
// Analogous to kubectl logs pod --tail -1 --limit-bytes n -c container --namespace=client.namespace
func (c *Client) GetLogTail(pod, container string, n int64) ([]byte, error) {
	c.log("GetLogTail", pod, n)
	return c.requestRetry(&request{
		path: fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/log", c.namespace, pod),
		query: map[string]string{ // Because we want last n bytes, we fetch all lines and then limit to n bytes
			"tailLines":  "-1",
			"container":  container,
			"limitBytes": strconv.FormatInt(n, 10),
		},
	})
}

// GetContainerLog returns the log of a container in the specified pod, in the client's specified namespace.
//
// Analogous to kubectl logs pod -c container --namespace=client.namespace
func (c *Client) GetContainerLog(pod, container string) ([]byte, error) {
	c.log("GetContainerLog", pod)
	return c.requestRetry(&request{
		path:  fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/log", c.namespace, pod),
		query: map[string]string{"container": container},
	})
}

// CreateConfigMap creates a configmap, in the client's specified namespace.
//
// Analogous to kubectl create configmap --namespace=client.namespace
func (c *Client) CreateConfigMap(content v1.ConfigMap) (v1.ConfigMap, error) {
	c.log("CreateConfigMap")
	var retConfigMap v1.ConfigMap
	err := c.request(&request{
		method:      http.MethodPost,
		path:        fmt.Sprintf("/api/v1/namespaces/%s/configmaps", c.namespace),
		requestBody: &content,
	}, &retConfigMap)

	return retConfigMap, err
}

// GetConfigMap gets the configmap identified, in the client's specified namespace.
//
// Analogous to kubectl get configmap --namespace=client.namespace
func (c *Client) GetConfigMap(name, namespace string) (v1.ConfigMap, error) {
	c.log("GetConfigMap", name)
	if namespace == "" {
		namespace = c.namespace
	}
	var retConfigMap v1.ConfigMap
	err := c.request(&request{
		path: fmt.Sprintf("/api/v1/namespaces/%s/configmaps/%s", namespace, name),
	}, &retConfigMap)

	return retConfigMap, err
}

// ReplaceConfigMap puts the configmap into name.
//
// Analogous to kubectl replace configmap
//
// If config.Namespace is empty, the client's specified namespace is used.
// Returns the content returned by the apiserver
func (c *Client) ReplaceConfigMap(name string, config v1.ConfigMap) (v1.ConfigMap, error) {
	c.log("ReplaceConfigMap", name)
	namespace := c.namespace
	if config.Namespace != "" {
		namespace = config.Namespace
	}
	var retConfigMap v1.ConfigMap
	err := c.request(&request{
		method:      http.MethodPut,
		path:        fmt.Sprintf("/api/v1/namespaces/%s/configmaps/%s", namespace, name),
		requestBody: &config,
	}, &retConfigMap)

	return retConfigMap, err
}
