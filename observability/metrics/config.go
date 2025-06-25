/*
Copyright 2025 The Knative Authors

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

package metrics

import (
	"fmt"
	"time"

	configmap "knative.dev/pkg/configmap/parser"
)

const (
	ProtocolGRPC         = "grpc"
	ProtocolHTTPProtobuf = "http/protobuf"
	ProtocolPrometheus   = "prometheus"
	ProtocolNone         = "none"
)

type Config struct {
	Protocol string
	Endpoint string

	ExportInterval time.Duration
}

func (c *Config) Validate() error {
	switch c.Protocol {
	case ProtocolGRPC,
		ProtocolHTTPProtobuf,
		ProtocolNone,
		ProtocolPrometheus:
	default:
		return fmt.Errorf("unsupported protocol %q", c.Protocol)
	}

	if c.Protocol == ProtocolNone || c.Protocol == ProtocolPrometheus {
		if len(c.Endpoint) > 0 {
			return fmt.Errorf("endpoint should not be set when protocol is %q", c.Protocol)
		}
	} else if len(c.Endpoint) == 0 {
		return fmt.Errorf("endpoint should be set when protocol is %q", c.Protocol)
	}

	if c.ExportInterval < 0 {
		return fmt.Errorf("export interval %q should be greater than zero", c.ExportInterval)
	}
	return nil
}

func DefaultConfig() Config {
	return Config{
		Protocol: ProtocolNone,
	}
}

func NewFromMap(m map[string]string) (Config, error) {
	return NewFromMapWithPrefix("", m)
}

func NewFromMapWithPrefix(prefix string, m map[string]string) (Config, error) {
	c := DefaultConfig()

	err := configmap.Parse(m,
		configmap.As(prefix+"metrics-protocol", &c.Protocol),
		configmap.As(prefix+"metrics-endpoint", &c.Endpoint),
		configmap.As(prefix+"metrics-export-interval", &c.ExportInterval),
	)
	if err != nil {
		return c, err
	}

	return c, c.Validate()
}
