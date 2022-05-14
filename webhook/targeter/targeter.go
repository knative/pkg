/*
Copyright 2022 The Knative Authors

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

package targeter

import (
	"context"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	apixv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// Interface encapsulates how webhooks target us.
type Interface interface {
	// BasePath returns the base path on which we serve.
	// If the path ends in a '/' then it will receive requests for all paths
	// with that prefix.
	// If the path does not end in a '/' then it will match exactly this path.
	BasePath() string

	// WebhookClientConfig returns the targeting config for this webhook.
	// The path in the configured WebhookClientConfig should match, or have a
	// prefix matching the value returned by BasePath above.
	WebhookClientConfig(context.Context) (*admissionregistrationv1.WebhookClientConfig, error)

	// AddEventHandler registers informer events as needed by this Interface
	AddEventHandlers(context.Context, func(interface{}))
}

// SwitchClientConfig converts an admission webhook config to an API extension
// webhook config.  These are effectively type aliases, but Kubernetes redefines
// the type for unknown reasons.
func SwitchClientConfig(in *admissionregistrationv1.WebhookClientConfig) *apixv1.WebhookClientConfig {
	out := &apixv1.WebhookClientConfig{
		URL:      in.URL,
		CABundle: in.CABundle,
	}
	if in.Service != nil {
		out.Service = &apixv1.ServiceReference{
			Namespace: in.Service.Namespace,
			Name:      in.Service.Name,
			Path:      in.Service.Path,
			Port:      in.Service.Port,
		}
	}
	return out
}
