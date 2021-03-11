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

package webhook

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"

	// Make system.Namespace() work in tests.
	_ "knative.dev/pkg/system/testing"

	. "knative.dev/pkg/reconciler/testing"
)

func newDefaultOptions() Options {
	return Options{
		ServiceName: "webhook",
		Port:        8443,
		SecretName:  "webhook-certs",
	}
}

const (
	testResourceName = "test-resource"
	user1            = "brutto@knative.dev"
)

func newNonRunningTestWebhook(t *testing.T, options Options, acs ...interface{}) (
	ctx context.Context, ac *Webhook, cancel context.CancelFunc) {
	t.Helper()

	// Create fake clients
	ctx, ctxCancel, informers := SetupFakeContextWithCancel(t)
	ctx = WithOptions(ctx, options)

	stopCb, err := RunAndSyncInformers(ctx, informers...)
	if err != nil {
		t.Fatal("StartInformers() =", err)
	}
	cancel = func() {
		ctxCancel()
		stopCb()
	}

	ac, err = New(ctx, acs)
	if err != nil {
		t.Fatal("Failed to create new admission controller:", err)
	}
	ac.gracePeriod = 100 * time.Millisecond
	return
}

func TestRegistrationStopChanFire(t *testing.T) {
	opts := newDefaultOptions()
	_, ac, cancel := newNonRunningTestWebhook(t, opts)
	defer cancel()

	stopCh := make(chan struct{})

	var g errgroup.Group
	g.Go(func() error {
		return ac.Run(stopCh)
	})
	close(stopCh)

	if err := g.Wait(); err != nil {
		t.Fatal("Error during run: ", err)
	}
	conn, err := net.Dial("tcp", fmt.Sprintf(":%d", opts.Port))
	if err == nil {
		conn.Close()
		t.Error("Unexpected success to dial to port", opts.Port)
	}
}
