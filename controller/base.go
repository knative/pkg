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
	"go.uber.org/zap"

	sharedclientset "github.com/knative/pkg/client/clientset/versioned"
	"github.com/knative/pkg/logging/logkey"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
)

// Base implements the core controller logic, given a Reconciler.
type Base struct {
	// KubeClientSet allows us to talk to the k8s for core APIs
	KubeClientSet kubernetes.Interface

	// SharedClientSet allows us to configure shared objects
	SharedClientSet sharedclientset.Interface

	// DynamicClientSet allows us to configure pluggable Build objects
	DynamicClientSet dynamic.Interface

	// ConfigMapWatcher allows us to watch for ConfigMap changes.
	// ConfigMapWatcher configmap.Watcher <-- TODO: we might want to plumb the config map watcher.

	// Recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	Recorder record.EventRecorder

	// Sugared logger is easier to use but is not as performant as the
	// raw logger. In performance critical paths, call logger.Desugar()
	// and use the returned raw logger instead. In addition to the
	// performance benefits, raw logger also preserves type-safety at
	// the expense of slightly greater verbosity.
	Logger *zap.SugaredLogger
}

// NewBase instantiates a new instance of Base implementing
// the common & boilerplate code between Knative reconcilers.
func NewBase(opt Options, controllerAgentName string) *Base {
	// Enrich the logs with controller name
	logger := opt.Logger.Named(controllerAgentName).With(zap.String(logkey.ControllerType, controllerAgentName))

	recorder := opt.Recorder
	if recorder == nil {
		// Create event broadcaster
		logger.Debug("Creating event broadcaster")
		eventBroadcaster := record.NewBroadcaster()
		watches := []watch.Interface{
			eventBroadcaster.StartLogging(logger.Named("event-broadcaster").Infof),
			eventBroadcaster.StartRecordingToSink(
				&typedcorev1.EventSinkImpl{Interface: opt.KubeClientSet.CoreV1().Events("")}),
		}
		recorder = eventBroadcaster.NewRecorder(
			scheme.Scheme, corev1.EventSource{Component: controllerAgentName})
		go func() {
			<-opt.StopChannel
			for _, w := range watches {
				w.Stop()
			}
		}()
	}

	base := &Base{
		KubeClientSet:    opt.KubeClientSet,
		SharedClientSet:  opt.SharedClientSet,
		DynamicClientSet: opt.DynamicClientSet,
		//ConfigMapWatcher:  opt.ConfigMapWatcher, <-- TODO: we might want to plumb the config map watcher.
		Recorder: recorder,
		Logger:   logger,
	}

	return base
}
