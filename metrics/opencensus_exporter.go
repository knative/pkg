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

package metrics

import (
	"context"
	"crypto/tls"
	"fmt"
	"sync"

	"contrib.go.opencensus.io/exporter/ocagent"
	"go.opencensus.io/resource"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.uber.org/zap"
	"google.golang.org/grpc/credentials"
	"k8s.io/apimachinery/pkg/api/errors"
	"knative.dev/pkg/metrics/metricskey"
)

func newOpenCensusExporter(config *metricsConfig, logger *zap.SugaredLogger) (view.Exporter, error) {
	opts := []ocagent.ExporterOption{ocagent.WithServiceName(config.component)}
	if config.collectorAddress != "" {
		opts = append(opts, ocagent.WithAddress(config.collectorAddress))
	}
	if config.requireSecure {
		opts = append(opts, ocagent.WithTLSCredentials(credentialFetcher(config.component, config.secretFetcher, logger)))
	} else {
		opts = append(opts, ocagent.WithInsecure())
	}
	e, err := ocagent.NewExporter(opts...)
	if err != nil {
		logger.Errorw("Failed to create the OpenCensus exporter.", zap.Error(err))
		return nil, err
	}
	logger.Infof("Created OpenCensus exporter with config: %+v.", *config)
	view.RegisterExporter(e)
	return e, nil
}

// credentialFetcher attempts to locate a secret containing TLS credentials
// for communicating with the OpenCensus Agent. To do this, it first looks
// for a secret named "<component>-opencensus", then for a generic
// "opencensus" secret.
func credentialFetcher(component string, lister SecretFetcher, logger *zap.SugaredLogger) credentials.TransportCredentials {
	if lister == nil {
		logger.Errorf("No secret lister provided for component %q; cannot use requireSecure=true", component)
		return nil
	}
	return credentials.NewTLS(&tls.Config{
		GetClientCertificate: func(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
			// We ignore the CertificateRequestInfo for now, and hand back a single fixed certificate.
			// TODO(evankanderson): maybe do something SPIFFE-ier?
			cert, err := certificateFetcher(component+"-opencensus", lister)
			if errors.IsNotFound(err) {
				cert, err = certificateFetcher("opencensus", lister)
			}
			if err != nil {
				return nil, fmt.Errorf("Unable to fetch opencensus secret for %q, cannot use requireSecure=true: %+v", component, err)
			}
			return &cert, err
		},
	})
}

func certificateFetcher(secretName string, lister SecretFetcher) (tls.Certificate, error) {
	secret, err := lister(secretName)
	if err != nil {
		return tls.Certificate{}, err
	}
	return tls.X509KeyPair(secret.Data["client-cert.pem"], secret.Data["client-key.pem"])
}

// resourceMapping is an internal per-resource data structure
type resourceExporter struct {
	// meter is used to aggregate stats updates
	meter view.Meter
	// recordWith is a `view.WithRecorder(meter)`
	recordWith stats.Options
	// exporter is needed to flush data when shutting down
	exporter *ocagent.Exporter
}

type ocagentStatus struct {
	// resourceToExporter is a map from either of the following to a *resourceExporter:
	// * a `*resource.Resource`
	// * a string in the format resource.
	resourceToExporter sync.Map

	// resourcedViews tracks the views which should be registered on each view.Meter
	resourcedViews []*view.View

	// storedOpts is a cache of the standard arguments used to create a new
	// ocagent.Exporter (each Export exports for only a single Resource, because
	// of how the opencensus view aggregation is structured.)
	storedOpts []ocagent.ExporterOption
}

func (o *ocagentStatus) ocRecord(ctx context.Context, ms []stats.Measurement, opt ...stats.Options) error {
	r := metricskey.GetResource(ctx)
	opt = append(opt, stats.WithMeasurements(ms...))
	if r == nil {
		return stats.RecordWithOptions(ctx, opt...)
	}
	re, err := o.getExporter(r)
	if err != nil {
		return err
	}
	opt = append(opt, re.recordWith)
	return stats.RecordWithOptions(ctx, opt...)
}

// Computes a string representation of the given resource. r must be non-`nil`.
func resourceToStringKey(r *resource.Resource) string {
	// TODO: measure performance and avoid string copies if this ends up a performance hotspot.
	return r.Type + ":" + resource.EncodeLabels(r.Labels)
}

// Finds or constructs a new resourceExporter for the given resource, which must not be `nil`.
func (o *ocagentStatus) getExporter(r *resource.Resource) (*resourceExporter, error) {
	var e interface{}
	if e, _ = o.resourceToExporter.Load(r); e != nil {
		return e.(*resourceExporter), nil
	}
	stringKey := resourceToStringKey(r)
	if e, _ = o.resourceToExporter.Load(stringKey); e != nil {
		return e.(*resourceExporter), nil
	}
	// Construct a new exporter -- note that this may race with another construction,
	// so we may need to discard the value after LoadOrStore.
	opts := append(o.storedOpts, ocagent.WithResourceDetector(
		func(context.Context) (*resource.Resource, error) {
			return r, nil
		}))
	exporter, err := ocagent.NewExporter(opts...)
	if err != nil {
		return nil, err
	}
	newExporter := resourceExporter{
		meter:    view.NewMeter(),
		exporter: exporter,
	}
	newExporter.recordWith = stats.WithRecorder(newExporter.meter)

	e, existing := o.resourceToExporter.LoadOrStore(stringKey, newExporter)
	re := e.(*resourceExporter)
	if existing {
		// This is the race; discard the exporter we created.
		newExporter.exporter.Stop()
	} else {
		// Get stats export started
		re.meter.RegisterExporter(re.exporter)
		err = re.meter.Register(o.resourcedViews...)
		if err != nil {
			return nil, err
		}
	}
	return re, nil
}
