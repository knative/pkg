/*
Copyright 2019 The Knative Authors

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

package controller

import (
	"context"
	"time"

	"go.uber.org/zap"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"

	sharedclientset "github.com/knative/pkg/client/clientset/versioned"
	"github.com/knative/pkg/logging"
	"github.com/knative/pkg/version"
)

// Options defines the common reconciler options.
// We define this to reduce the boilerplate argument list when
// creating our controllers.
type Options struct {
	// KubeClientSet is the core k8s clientset.
	KubeClientSet kubernetes.Interface

	// SharedClientSet are shared dependency clientsets.
	SharedClientSet sharedclientset.Interface

	// DynamicClientSet is the dynamic k8s client using unstructured.Unstructured.
	DynamicClientSet dynamic.Interface

	// Recorder is the k8s event recorder.
	Recorder record.EventRecorder

	// Logger logs to the configured log system.
	Logger *zap.SugaredLogger

	// ResyncPeriod default informer resync period.
	ResyncPeriod time.Duration

	// TrackerMultiple a multiple of the resync period to use.
	TrackerMultiple int

	// StopChannel is the shared stop channel to end the process.
	StopChannel <-chan struct{}

	// TODO: We should have a common stats reporter, but these are custom at the moment.
	//StatsReporter StatsReporter
}

// GetTrackerLease returns a multiple of the resync period to use as the
// duration for tracker leases. This attempts to ensure that resyncs happen to
// refresh leases frequently enough that we don't miss updates to tracked
// objects.
func (o Options) GetTrackerLease() time.Duration {
	return o.ResyncPeriod * time.Duration(o.TrackerMultiple)
}

func NewOptions(ctx context.Context, cfg *rest.Config, stopCh <-chan struct{}) Options {
	logger := logging.FromContext(ctx)

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		logger.Fatalw("Error building kubernetes clientset", zap.Error(err))
	}

	sharedClient, err := sharedclientset.NewForConfig(cfg)
	if err != nil {
		logger.Fatalw("Error building shared clientset", zap.Error(err))
	}

	dynamicClient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		logger.Fatalw("Error building dynamic clientset", zap.Error(err))
	}

	if err := version.CheckMinimumVersion(kubeClient.Discovery()); err != nil {
		logger.Fatalf("Version check failed: %v", err)
	}

	opts := Options{
		KubeClientSet:    kubeClient,
		SharedClientSet:  sharedClient,
		DynamicClientSet: dynamicClient,
		Logger:           logger,
		ResyncPeriod:     10 * time.Hour, // Based on controller-runtime default.
		TrackerMultiple:  3,              // Based on knative usage.
		StopChannel:      stopCh,
	}

	return opts
}
