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

package logstream_test

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	fakecorev1 "k8s.io/client-go/kubernetes/typed/core/v1/fake"
	restclient "k8s.io/client-go/rest"
	fakerest "k8s.io/client-go/rest/fake"
	"knative.dev/pkg/test/logstream/v2"
)

var pod = &corev1.Pod{
	ObjectMeta: metav1.ObjectMeta{
		Name:      logstream.ChaosDuck,
		Namespace: "default",
	},
	Spec: corev1.PodSpec{
		Containers: []corev1.Container{{
			Name: logstream.ChaosDuck,
		}},
	},
}

var readyStatus = corev1.PodStatus{
	Phase: corev1.PodRunning,
	Conditions: []corev1.PodCondition{{
		Type:   corev1.PodReady,
		Status: corev1.ConditionTrue,
	}},
}

func TestStreamErr(t *testing.T) {
	f := newK8sFake(fake.NewSimpleClientset(), errors.New("lookin' good!"))
	stream := logstream.FromNamespace(context.Background(), f, "a-namespace")
	_, err := stream.StartStream(pod.Name, nil)
	if err == nil {
		t.Fatal("LogStream creation should have failed")
	}
}

func TestNamespaceStream(t *testing.T) {
	f := newK8sFake(fake.NewSimpleClientset(), nil)

	logFuncInvoked := make(chan struct{})
	t.Cleanup(func() { close(logFuncInvoked) })
	logFunc := func(format string, args ...interface{}) {
		logFuncInvoked <- struct{}{}
	}

	ctx, cancel := context.WithCancel(context.Background())
	stream := logstream.FromNamespace(ctx, f, pod.Namespace)
	streamC, err := stream.StartStream(pod.Name, logFunc)
	if err != nil {
		t.Fatal("Failed to start the stream: ", err)
	}
	t.Cleanup(streamC)

	podClient := f.CoreV1().Pods(pod.Namespace)
	if _, err := podClient.Create(context.Background(), pod, metav1.CreateOptions{}); err != nil {
		t.Fatal("CreatePod()=", err)
	}

	select {
	case <-time.After(time.Second):
	case <-logFuncInvoked:
		t.Error("Unready pod should not report logs")
	}

	pod.Status = readyStatus
	if _, err := podClient.Update(context.Background(), pod, metav1.UpdateOptions{}); err != nil {
		t.Fatal("UpdatePod()=", err)
	}

	select {
	case <-time.After(time.Second):
		t.Error("Timed out: log message wasn't received")
	case <-logFuncInvoked:
	}

	if _, err := podClient.Update(context.Background(), pod, metav1.UpdateOptions{}); err != nil {
		t.Fatal("UpdatePod()=", err)
	}

	select {
	case <-time.After(time.Second):
	case <-logFuncInvoked:
		t.Error("Repeat updates to the same pod should not trigger GetLogs")
	}

	if err := podClient.Delete(context.Background(), pod.Name, metav1.DeleteOptions{}); err != nil {
		t.Fatal("UpdatePod()=", err)
	}

	select {
	case <-time.After(time.Second):
	case <-logFuncInvoked:
		t.Error("Deletion should not trigger GetLogs")
	}

	// Create pod with the same name? Why not. And let's make it ready from the get go.
	if _, err := podClient.Create(context.Background(), pod, metav1.CreateOptions{}); err != nil {
		t.Fatal("CreatePod()=", err)
	}

	select {
	case <-time.After(time.Second):
		t.Error("Timed out: log message wasn't received")
	case <-logFuncInvoked:
	}

	// Delete again.
	if err := podClient.Delete(context.Background(), pod.Name, metav1.DeleteOptions{}); err != nil {
		t.Fatal("UpdatePod()=", err)
	}
	// Kill the context.
	cancel()

	// Re-create pod, but the watch cycle must have finished by now.
	if _, err := podClient.Create(context.Background(), pod, metav1.CreateOptions{}); err != nil {
		t.Fatal("CreatePod()=", err)
	}

	select {
	case <-time.After(time.Second):
	case <-logFuncInvoked:
		t.Error("No watching should have happened.")
	}
}

func newK8sFake(c *fake.Clientset, watchErr error) *fakeclient {
	return &fakeclient{
		Clientset:  c,
		FakeCoreV1: &fakecorev1.FakeCoreV1{Fake: &c.Fake},
		watchErr:   watchErr,
	}
}

type fakeclient struct {
	*fake.Clientset
	*fakecorev1.FakeCoreV1
	watchErr error
}

type fakePods struct {
	*fakeclient
	v1.PodInterface
	ns       string
	watchErr error
}

func (f *fakePods) Watch(ctx context.Context, lo metav1.ListOptions) (watch.Interface, error) {
	if f.watchErr == nil {
		return f.PodInterface.Watch(ctx, lo)
	}
	return nil, f.watchErr
}

func (f *fakeclient) CoreV1() v1.CoreV1Interface { return f }

func (f *fakeclient) Pods(ns string) v1.PodInterface {
	return &fakePods{
		f,
		f.FakeCoreV1.Pods(ns),
		ns,
		f.watchErr,
	}
}

func (f *fakePods) GetLogs(name string, opts *corev1.PodLogOptions) *restclient.Request {
	fakeClient := &fakerest.RESTClient{
		Client: fakerest.CreateHTTPClient(func(request *http.Request) (*http.Response, error) {
			resp := &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(strings.NewReader("hello\n")),
			}
			return resp, nil
		}),
		NegotiatedSerializer: scheme.Codecs.WithoutConversion(),
		GroupVersion:         schema.GroupVersion{Version: "v1"},
		VersionedAPIPath:     fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/log", f.ns, name),
	}
	return fakeClient.Request()
}
