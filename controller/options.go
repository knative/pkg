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
	"github.com/knative/pkg/configmap"
	"github.com/knative/pkg/logging"
	"github.com/knative/pkg/metrics"
	"github.com/knative/pkg/signals"
	"github.com/knative/pkg/system"
	"log"
	"time"

	"go.uber.org/zap"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"

	sharedclientset "github.com/knative/pkg/client/clientset/versioned"
	"github.com/knative/pkg/version"
)

const (
	defaultResyncPeriod    = 10 * time.Hour // Based on controller-runtime default.
	defaultTrackerMultiple = 3              // Based on knative usage.
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

	// AtomicLevel is the atomic level the logger was started with.
	AtomicLevel zap.AtomicLevel

	// ResyncPeriod default informer resync period.
	ResyncPeriod time.Duration

	// TrackerMultiple a multiple of the resync period to use.
	TrackerMultiple int

	ConfigMapWatcher *configmap.InformedWatcher

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

// ConfigMapConfig holds config map names, paths and ObserverDecorator fn for Metrics and Logging.
type ConfigMapConfig struct {
	LoggingConfigPath string
	LoggingConfigName string
	LoggingObserver   logging.ObserverDecorator

	MetricsConfigPath string
	MetricsConfigName string
	MetricsObserver   metrics.ObserverDecorator
}

// NewOptions creates the common to Knative controller options.
// component is the name of the controller component.
// loggingConfigFile is the file path to the logging config map.
func NewOptions(component string, cfg *rest.Config, configCfg ConfigMapConfig) Options {
	loggingConfigMap, err := configmap.Load(configCfg.LoggingConfigPath)
	if err != nil {
		log.Fatalf("Error loading logging configuration: %v", err)
	}
	loggingConfig, err := logging.NewConfigFromMap(loggingConfigMap)
	if err != nil {
		log.Fatalf("Error parsing logging configuration: %v", err)
	}
	logger, atomicLevel := logging.NewLoggerFromConfig(loggingConfig, component)
	defer logger.Sync()

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	opts := Options{
		KubeClientSet:    kubernetes.NewForConfigOrDie(cfg),
		SharedClientSet:  sharedclientset.NewForConfigOrDie(cfg),
		DynamicClientSet: dynamic.NewForConfigOrDie(cfg),
		Logger:           logger,
		AtomicLevel:      atomicLevel,
		ResyncPeriod:     defaultResyncPeriod,
		TrackerMultiple:  defaultTrackerMultiple,
		StopChannel:      stopCh,
	}
	opts.ConfigMapWatcher = configmap.NewInformedWatcher(opts.KubeClientSet, system.Namespace())

	// Watch the logging config map and dynamically update logging levels.
	opts.ConfigMapWatcher.Watch(configCfg.LoggingConfigPath, configCfg.LoggingObserver(logger, atomicLevel))

	// Watch the observability config map and dynamically update metrics exporter.
	opts.ConfigMapWatcher.Watch(configCfg.MetricsConfigName, configCfg.MetricsObserver(logger))

	if err := version.CheckMinimumVersion(opts.KubeClientSet.Discovery()); err != nil {
		logger.Fatalf("Version check failed: %v", err)
	}

	return opts
}
