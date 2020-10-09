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
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	clientcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	fakecorev1 "k8s.io/client-go/kubernetes/typed/core/v1/fake"
	restclient "k8s.io/client-go/rest"
	fakerest "k8s.io/client-go/rest/fake"
	"knative.dev/pkg/test/logstream/v2"
)

var pod = &corev1.Pod{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "chaosduck",
		Namespace: "default",
	},
	Spec: corev1.PodSpec{
		Containers: []corev1.Container{
			{Name: "chaosduck"},
		},
	},
}

var readyStatus = corev1.PodStatus{
	Phase: corev1.PodRunning,
	Conditions: []corev1.PodCondition{
		{Type: corev1.PodReady, Status: corev1.ConditionTrue},
	},
}

func TestNamespaceStream(t *testing.T) {
	f := newK8sFake(fake.NewSimpleClientset())

	logFuncInvoked := make(chan string)
	logFunc := func(format string, args ...interface{}) {
		close(logFuncInvoked)
	}

	stream := logstream.FromNamespace(context.TODO(), f, pod.Namespace)
	stream.StartPodStream(pod.Name, logFunc)

	podClient := f.CoreV1().Pods(pod.Namespace)
	if _, err := podClient.Create(context.TODO(), pod, metav1.CreateOptions{}); err != nil {
		t.Fatal("CreatePod()=", err)
	}

	pod.Status = readyStatus
	if _, err := podClient.Update(context.TODO(), pod, metav1.UpdateOptions{}); err != nil {
		t.Fatal("UpdatePod()=", err)
	}

	if _, err := f.logBuffer.WriteString("test\n"); err != nil {
		t.Fatal("WriteString()=", err)
	}

	select {
	case <-time.After(time.Second):
		t.Error("timed out: log message wasn't received")
	case <-logFuncInvoked:
	}
}

func newK8sFake(c *fake.Clientset) *fakeclient {
	return &fakeclient{
		Clientset:  c,
		FakeCoreV1: &fakecorev1.FakeCoreV1{Fake: &c.Fake},
		logBuffer:  new(bytes.Buffer),
	}
}

type fakeclient struct {
	*fake.Clientset
	*fakecorev1.FakeCoreV1

	logBuffer *bytes.Buffer
}

type fakePods struct {
	*fakeclient
	// *fakecorev1.FakePods
	v1.PodInterface
	ns string
}

func (f *fakeclient) CoreV1() clientcorev1.CoreV1Interface { return f }

func (f *fakeclient) Pods(ns string) v1.PodInterface {
	return &fakePods{
		f,
		f.FakeCoreV1.Pods(ns), //.(*fakecorev1.FakePods),
		ns,
	}
}

func (f *fakePods) GetLogs(name string, opts *corev1.PodLogOptions) *restclient.Request {
	fakeClient := &fakerest.RESTClient{
		Client: fakerest.CreateHTTPClient(func(request *http.Request) (*http.Response, error) {
			resp := &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(f.logBuffer),
			}
			return resp, nil
		}),
		NegotiatedSerializer: scheme.Codecs.WithoutConversion(),
		GroupVersion:         schema.GroupVersion{Version: "v1"},
		VersionedAPIPath:     fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/log", f.ns, name),
	}
	return fakeClient.Request()
}
