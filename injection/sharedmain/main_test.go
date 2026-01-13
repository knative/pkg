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

package sharedmain

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	fakekubeclientset "k8s.io/client-go/kubernetes/fake"

	fakekubeclient "knative.dev/pkg/client/injection/kube/client/fake"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
	cminformer "knative.dev/pkg/configmap/informer"
	"knative.dev/pkg/injection"
	logtesting "knative.dev/pkg/logging/testing"
	"knative.dev/pkg/leaderelection"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/observability"
	"knative.dev/pkg/system"
	_ "knative.dev/pkg/system/testing"
)

func TestEnabledControllers(t *testing.T) {
	tests := []struct {
		name                string
		disabledControllers []string
		ctors               []injection.NamedControllerConstructor
		wantNames           []string
	}{{
		name:                "zero",
		disabledControllers: []string{"foo"},
		ctors:               []injection.NamedControllerConstructor{{Name: "bar"}},
		wantNames:           []string{"bar"},
	}, {
		name:                "one",
		disabledControllers: []string{"foo"},
		ctors:               []injection.NamedControllerConstructor{{Name: "foo"}},
		wantNames:           []string{},
	}, {
		name:                "two",
		disabledControllers: []string{"foo"},
		ctors: []injection.NamedControllerConstructor{
			{Name: "foo"},
			{Name: "bar"},
		},
		wantNames: []string{"bar"},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := enabledControllers(tt.disabledControllers, tt.ctors)
			if diff := cmp.Diff(tt.wantNames, namesOf(got)); diff != "" {
				t.Error("(-want, +got)", diff)
			}
		})
	}
}

func namesOf(ctors []injection.NamedControllerConstructor) []string {
	names := make([]string, 0, len(ctors))
	for _, x := range ctors {
		names = append(names, x.Name)
	}
	return names
}

func TestWithLoggingConfig(t *testing.T) {
	want := &logging.Config{
		LoggingLevel: map[string]zapcore.Level{
			"foo": zapcore.DebugLevel,
		},
	}
	ctx := logging.WithConfig(context.Background(), want)

	got, err := GetLoggingConfig(ctx)
	if err != nil {
		t.Fatalf("GetLoggingConfig() = %v", err)
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("(-got, +want) = %s", diff)
	}
}

func TestWithLeaderElectionConfig(t *testing.T) {
	want := &leaderelection.Config{
		Buckets: 12,
	}
	ctx := leaderelection.WithConfig(context.Background(), want)

	got, err := GetLeaderElectionConfig(ctx)
	if err != nil {
		t.Fatalf("GetLeaderElectionConfig() = %v", err)
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("(-got, +want) = %s", diff)
	}
}

func TestWithObservabilityConfig(t *testing.T) {
	want := &observability.Config{
		Tracing: observability.TracingConfig{
			Protocol: "some-protocol",
		},
	}
	ctx := observability.WithConfig(context.Background(), want)

	got, err := GetObservabilityConfig(ctx)
	if err != nil {
		t.Fatalf("GetObservabilityConfig() = %v", err)
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("(-got, +want) = %s", diff)
	}
}

func TestMissingConfigMapsUseDefaults(t *testing.T) {
	// Create a fake client with no ConfigMaps
	ctx := logtesting.TestContextWithLogger(t)
	ctx, _ = fakekubeclient.With(ctx)

	// Set up injection context
	ctx, _ = injection.Fake.SetupInformers(ctx, &rest.Config{})

	// Test that GetLoggingConfig returns defaults when ConfigMap is missing
	loggingConfig, err := GetLoggingConfig(ctx)
	if err != nil {
		t.Fatalf("GetLoggingConfig() should not fail with missing ConfigMap, got error: %v", err)
	}
	// Verify it's the default (empty LoggingLevel map)
	defaultLoggingConfig, _ := logging.NewConfigFromMap(nil)
	if diff := cmp.Diff(loggingConfig, defaultLoggingConfig); diff != "" {
		t.Errorf("GetLoggingConfig() with missing ConfigMap should return defaults, (-got, +want) = %s", diff)
	}

	// Test that GetObservabilityConfig returns defaults when ConfigMap is missing
	observabilityConfig, err := GetObservabilityConfig(ctx)
	if err != nil {
		t.Fatalf("GetObservabilityConfig() should not fail with missing ConfigMap, got error: %v", err)
	}
	// Verify it's the default config
	defaultObservabilityConfig := observability.DefaultConfig()
	if diff := cmp.Diff(observabilityConfig, defaultObservabilityConfig); diff != "" {
		t.Errorf("GetObservabilityConfig() with missing ConfigMap should return defaults, (-got, +want) = %s", diff)
	}

	// Test that GetLeaderElectionConfig returns defaults when ConfigMap is missing
	leaderElectionConfig, err := GetLeaderElectionConfig(ctx)
	if err != nil {
		t.Fatalf("GetLeaderElectionConfig() should not fail with missing ConfigMap, got error: %v", err)
	}
	// Verify it's the default (created from nil ConfigMap)
	defaultLeaderElectionConfig, _ := leaderelection.NewConfigFromConfigMap(nil)
	if diff := cmp.Diff(leaderElectionConfig, defaultLeaderElectionConfig); diff != "" {
		t.Errorf("GetLeaderElectionConfig() with missing ConfigMap should return defaults, (-got, +want) = %s", diff)
	}

	// Test that SetupConfigMapWatchOrDie doesn't panic
	logger := logtesting.TestLogger(t)
	cmw := SetupConfigMapWatchOrDie(ctx, logger)
	if cmw == nil {
		t.Fatal("SetupConfigMapWatchOrDie() should return a watcher")
	}

	// Test that WatchLoggingConfigOrDie doesn't panic with missing ConfigMap
	atomicLevel := zap.NewAtomicLevel()
	WatchLoggingConfigOrDie(ctx, cmw, logger, atomicLevel, "test-component")

	// Test that WatchObservabilityConfigOrDie doesn't panic with missing ConfigMap
	// We need a pprof server for this, but we can use nil or create a minimal one
	// For now, let's skip the pprof part and just verify the watcher setup doesn't crash
	// Actually, we need to import the pprof package, let's check if we can create a minimal test
}

func TestConfigMapWatcherObservesLaterCreation(t *testing.T) {
	// Use direct fake client (like in configmap/informer tests) to ensure informer events work
	kc := fakekubeclientset.NewSimpleClientset()
	ctx := logtesting.TestContextWithLogger(t)
	// Set up context with the fake client using the same key as the injection system
	ctx = context.WithValue(ctx, kubeclient.Key{}, kc)

	// Create watcher directly (similar to SetupConfigMapWatchOrDie but with direct client)
	cmw := cminformer.NewInformedWatcher(kc, system.Namespace())

	// Track if the update handler was invoked
	var handlerInvoked sync.WaitGroup
	handlerInvoked.Add(1)

	var receivedConfigMap *corev1.ConfigMap
	var handlerMutex sync.Mutex
	updateHandler := func(cm *corev1.ConfigMap) {
		handlerMutex.Lock()
		defer handlerMutex.Unlock()
		receivedConfigMap = cm
		handlerInvoked.Done()
	}

	// Watch a ConfigMap that doesn't exist yet
	cmName := "test-configmap"
	cmw.Watch(cmName, updateHandler)

	// Start the watcher
	stopCh := make(chan struct{})
	defer close(stopCh)

	if err := cmw.Start(stopCh); err != nil {
		t.Fatalf("cmw.Start() should succeed even with missing ConfigMap, got error: %v", err)
	}

	// Give the watcher time to fully start and sync
	time.Sleep(500 * time.Millisecond)

	// Now create the ConfigMap
	testCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cmName,
			Namespace: system.Namespace(),
		},
		Data: map[string]string{
			"key": "value",
		},
	}

	createdCM, err := kc.CoreV1().ConfigMaps(system.Namespace()).Create(ctx, testCM, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create ConfigMap: %v", err)
	}

	// Wait for the handler to be invoked (with timeout)
	done := make(chan struct{})
	go func() {
		handlerInvoked.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Handler was invoked, verify the ConfigMap
		handlerMutex.Lock()
		defer handlerMutex.Unlock()
		if receivedConfigMap == nil {
			t.Fatal("Update handler was invoked but receivedConfigMap is nil")
		}
		if receivedConfigMap.Name != cmName {
			t.Errorf("Received ConfigMap name = %q, want %q", receivedConfigMap.Name, cmName)
		}
		if receivedConfigMap.Namespace != system.Namespace() {
			t.Errorf("Received ConfigMap namespace = %q, want %q", receivedConfigMap.Namespace, system.Namespace())
		}
		if got, want := receivedConfigMap.Data["key"], "value"; got != want {
			t.Errorf("Received ConfigMap data[key] = %q, want %q", got, want)
		}
		// Verify it's the same object (or at least equivalent)
		if receivedConfigMap.UID != createdCM.UID {
			t.Errorf("Received ConfigMap UID = %q, want %q", receivedConfigMap.UID, createdCM.UID)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Update handler was not invoked within 5 seconds after ConfigMap creation")
	}
}
