/*
Copyright 2019 The Knative Authors.

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

package config

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"

	"cloud.google.com/go/compute/metadata"
	corev1 "k8s.io/api/core/v1"
)

const (
	// ConfigName is the name of the configmap
	ConfigName = "config-tracing"

	enableKey                  = "enable"
	backendKey                 = "backend"
	jaegerAgentEndpointKey     = "jaeger-agent-endpoint"
	jaegerCollectorEndpointKey = "jaeger-collector-endpoint"
	zipkinEndpointKey          = "zipkin-endpoint"
	debugKey                   = "debug"
	sampleRateKey              = "sample-rate"
	stackdriverProjectIDKey    = "stackdriver-project-id"
)

// BackendType specifies the backend to use for tracing
type BackendType string

const (
	// None is used for no backend.
	None BackendType = "none"
	// Stackdriver is used for Stackdriver backend.
	Stackdriver BackendType = "stackdriver"
	// Zipkin is used for Zipkin backend.
	Zipkin BackendType = "zipkin"
	// Jaeger is used for a Jaeger collector backend.
	Jaeger BackendType = "jaeger"
)

// Config holds the configuration for tracers
type Config struct {
	Backend                 BackendType
	JaegerAgentEndpoint     string
	JaegerCollectorEndpoint string
	ZipkinEndpoint          string
	StackdriverProjectID    string

	Debug      bool
	SampleRate float64
}

// Equals returns true if two Configs are identical
func (cfg *Config) Equals(other *Config) bool {
	return reflect.DeepEqual(other, cfg)
}

// NewTracingConfigFromMap returns a Config given a map corresponding to a ConfigMap
func NewTracingConfigFromMap(cfgMap map[string]string) (*Config, error) {
	tc := Config{
		Backend:    None,
		Debug:      false,
		SampleRate: 0.1,
	}

	if backend, ok := cfgMap[backendKey]; ok {
		switch bt := BackendType(backend); bt {
		case Stackdriver, Zipkin, Jaeger, None:
			tc.Backend = bt
		default:
			return nil, fmt.Errorf("unsupported tracing backend value %q", backend)
		}
	} else {
		// For backwards compatibility, parse the enabled flag as Zipkin.
		if enable, ok := cfgMap[enableKey]; ok {
			enableBool, err := strconv.ParseBool(enable)
			if err != nil {
				return nil, fmt.Errorf("failed parsing tracing config %q: %v", enableKey, err)
			}
			if enableBool {
				tc.Backend = Zipkin
			}
		}
	}

	if jaegerAgentEndpoint, ok := cfgMap[jaegerAgentEndpointKey]; ok {
		tc.JaegerAgentEndpoint = jaegerAgentEndpoint
	} else if jaegerCollectorEndpoint, ok := cfgMap[jaegerCollectorEndpointKey]; ok && tc.Backend == Jaeger {
		tc.JaegerCollectorEndpoint = jaegerCollectorEndpoint
	} else if tc.Backend == Jaeger {
		return nil, errors.New("jaeger tracing enabled without a jaeger agent or collector endpoint specified")
	}

	if zipkinEndpoint, ok := cfgMap[zipkinEndpointKey]; !ok && tc.Backend == Zipkin {
		return nil, errors.New("zipkin tracing enabled without a zipkin endpoint specified")
	} else {
		tc.ZipkinEndpoint = zipkinEndpoint
	}

	if projectID, ok := cfgMap[stackdriverProjectIDKey]; ok {
		tc.StackdriverProjectID = projectID
	} else if tc.Backend == Stackdriver {
		projectID, err := metadata.ProjectID()
		if err != nil {
			return nil, fmt.Errorf("stackdriver tracing enabled without a project-id specified: %v", err)
		}
		tc.StackdriverProjectID = projectID
	}

	if debug, ok := cfgMap[debugKey]; ok {
		debugBool, err := strconv.ParseBool(debug)
		if err != nil {
			return nil, fmt.Errorf("failed parsing tracing config %q", debugKey)
		}
		tc.Debug = debugBool
	}

	if sampleRate, ok := cfgMap[sampleRateKey]; ok {
		sampleRateFloat, err := strconv.ParseFloat(sampleRate, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse sampleRate in tracing config: %v", err)
		}
		tc.SampleRate = sampleRateFloat
	}

	return &tc, nil
}

// NewTracingConfigFromConfigMap returns a Config for the given configmap
func NewTracingConfigFromConfigMap(config *corev1.ConfigMap) (*Config, error) {
	return NewTracingConfigFromMap(config.Data)
}
