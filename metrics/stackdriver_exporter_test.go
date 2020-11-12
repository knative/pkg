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

package metrics

import (
	"context"
	"path"
	"testing"
	"time"

	sd "contrib.go.opencensus.io/exporter/stackdriver"
	"github.com/google/go-cmp/cmp"
	"go.opencensus.io/metric/metricdata"
	"go.opencensus.io/metric/metricexport"
	"go.opencensus.io/resource"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	. "knative.dev/pkg/logging/testing"
	"knative.dev/pkg/metrics/metricskey"
)

// TODO UTs should move to eventing and serving, as appropriate.
// 	See https://github.com/knative/pkg/issues/608

var testGcpMetadata = gcpMetadata{
	project:  "test-project",
	location: "test-location",
	cluster:  "test-cluster",
}

func fakeGcpMetadataFunc() *gcpMetadata {
	// the caller of this function could modify the struct, so we need a copy if we don't want the original modified.
	newTestGCPMetadata := testGcpMetadata
	return &newTestGCPMetadata
}

type fakeExporter struct{}

func (fe *fakeExporter) ExportView(vd *view.Data) {}
func (fe *fakeExporter) Flush()                   {}

func newFakeExporter(o sd.Options) (view.Exporter, error) {
	return &fakeExporter{}, nil
}

func makeResourceLabels(kv ...string) map[string]string {
	retval := map[string]string{
		metricskey.LabelProject:       testGcpMetadata.project,
		metricskey.LabelLocation:      testGcpMetadata.location,
		metricskey.LabelClusterName:   testGcpMetadata.cluster,
		metricskey.LabelNamespaceName: testNS,
	}
	for i := 0; i+1 < len(kv); i += 2 {
		retval[kv[i]] = kv[i+1]
	}
	return retval
}

type metricExtractor struct {
	data []*metricdata.Metric
}

func (me *metricExtractor) ExportMetrics(ctx context.Context, data []*metricdata.Metric) error {
	me.data = data
	return nil
}

func TestSdRecordWithResources(t *testing.T) {
	testCases := []struct {
		name               string
		domain             string
		component          string
		metricName         string
		allowCustomMetrics bool
		metricTags         map[string]string
		resource           resource.Resource
		expectedLabels     map[string]string
		expectedResource   map[string]string
	}{{
		name:       "Serving resource and metric labels",
		domain:     internalServingDomain,
		component:  "activator",
		metricName: "request_count",
		metricTags: map[string]string{
			metricskey.ContainerName:          testContainer,
			metricskey.PodName:                testPod,
			metricskey.LabelResponseCodeClass: "2xx",
			metricskey.LabelResponseCode:      "200",
		},
		resource: resource.Resource{
			Labels: map[string]string{
				metricskey.LabelConfigurationName: testConfiguration,
				metricskey.LabelNamespaceName:     testNS,
				metricskey.LabelServiceName:       testService,
				metricskey.LabelRevisionName:      testRevision,
			},
		},
		expectedLabels: map[string]string{
			metricskey.ContainerName:          testContainer,
			metricskey.PodName:                testPod,
			metricskey.LabelResponseCodeClass: "2xx",
			metricskey.LabelResponseCode:      "200",
		},
		expectedResource: makeResourceLabels(metricskey.LabelServiceName, testService,
			metricskey.LabelConfigurationName, testConfiguration,
			metricskey.LabelRevisionName, testRevision),
	}, {
		name:       "Serving only resource labels",
		domain:     internalServingDomain,
		component:  "activator",
		metricName: "request_count",
		resource: resource.Resource{Labels: map[string]string{
			metricskey.LabelConfigurationName: testConfiguration,
			metricskey.LabelNamespaceName:     testNS,
			metricskey.LabelServiceName:       testService,
			metricskey.LabelRevisionName:      testRevision,
		}},
		expectedResource: makeResourceLabels(metricskey.LabelServiceName, testService,
			metricskey.LabelConfigurationName, testConfiguration,
			metricskey.LabelRevisionName, testRevision),
	}, {
		name:       "Serving resource labels overwrite metric labels",
		domain:     internalServingDomain,
		component:  "activator",
		metricName: "request_count",
		metricTags: map[string]string{
			metricskey.LabelNamespaceName: testNS,
			metricskey.LabelServiceName:   testService,
		},
		resource: resource.Resource{Labels: map[string]string{
			metricskey.LabelNamespaceName: "foo",
			metricskey.LabelServiceName:   "bar",
			metricskey.LabelRevisionName:  testRevision,
		}},
		expectedResource: makeResourceLabels(metricskey.LabelNamespaceName, "foo",
			metricskey.LabelServiceName, "bar",
			metricskey.LabelConfigurationName, metricskey.ValueUnknown,
			metricskey.LabelRevisionName, testRevision),
	}, {
		name:       "Serving only metric labels",
		domain:     internalServingDomain,
		component:  "activator",
		metricName: "request_count",
		metricTags: map[string]string{
			metricskey.LabelNamespaceName:     testNS,
			metricskey.LabelServiceName:       testService,
			metricskey.LabelRevisionName:      testRevision,
			metricskey.ContainerName:          testContainer,
			metricskey.PodName:                testPod,
			metricskey.LabelResponseCodeClass: "2xx",
			metricskey.LabelResponseCode:      "200",
		},
		expectedLabels: map[string]string{
			metricskey.ContainerName:          testContainer,
			metricskey.PodName:                testPod,
			metricskey.LabelResponseCodeClass: "2xx",
			metricskey.LabelResponseCode:      "200",
		},
		expectedResource: makeResourceLabels(metricskey.LabelServiceName, testService,
			metricskey.LabelConfigurationName, metricskey.ValueUnknown,
			metricskey.LabelRevisionName, testRevision),
	}, {
		name:               "Serving only metric labels with allowCustomMetrics",
		domain:             internalServingDomain,
		component:          "activator",
		metricName:         "request_count",
		allowCustomMetrics: true,
		metricTags: map[string]string{
			metricskey.LabelNamespaceName:     testNS,
			metricskey.LabelServiceName:       testService,
			metricskey.LabelRevisionName:      testRevision,
			metricskey.ContainerName:          testContainer,
			metricskey.PodName:                testPod,
			metricskey.LabelResponseCodeClass: "2xx",
			metricskey.LabelResponseCode:      "200",
		},
		expectedLabels: map[string]string{
			metricskey.ContainerName:          testContainer,
			metricskey.PodName:                testPod,
			metricskey.LabelResponseCodeClass: "2xx",
			metricskey.LabelResponseCode:      "200",
		},
		expectedResource: makeResourceLabels(metricskey.LabelServiceName, testService,
			metricskey.LabelConfigurationName, metricskey.ValueUnknown,
			metricskey.LabelRevisionName, testRevision),
	}, {
		name:       "Eventing broker metrics",
		domain:     internalEventingDomain,
		component:  "broker",
		metricName: "event_count",
	}, {
		name:       "Eventing trigger metrics",
		domain:     internalEventingDomain,
		component:  "trigger",
		metricName: "event_processing_latencies",
	}, {
		name:       "Eventing source metrics",
		domain:     eventingDomain,
		component:  "source",
		metricName: "event_count",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			recordFunc := sdCustomMetricsRecorder(metricsConfig{
				stackdriverMetricTypePrefix: path.Join(tc.domain, tc.component),
			}, tc.allowCustomMetrics)
			m := stats.Int64(tc.metricName, "", "1")
			v := &view.View{
				Name:    "test_" + tc.metricName,
				Measure: m,

				Aggregation: view.Count(),
			}
			for k := range tc.metricTags {
				v.TagKeys = append(v.TagKeys, tag.MustNewKey(k))
			}
			if err := RegisterResourceView(v); err != nil {
				t.Error("Unable to register view:", err)
			}
			defer UnregisterResourceView(v)

			ctx := context.Background()
			ctx = metricskey.WithResource(ctx, tc.resource)
			tags := make([]tag.Mutator, 0, len(tc.metricTags))
			for k, v := range tc.metricTags {
				tags = append(tags, tag.Upsert(tag.MustNewKey(k), v))
			}
			ctx, err := tag.New(ctx, tags...)
			if err != nil {
				t.Error("Unable to set tags:", err)
			}

			if err := recordFunc(ctx, []stats.Measurement{m.M(1)}); err != nil {
				t.Errorf("Record %q failed: %v", tc.metricName, err)
			}

			// We need to sleep for a moment because stats.Record happens on a
			// background thread, and ReadAndExport happens on the local thread.
			// (This is probably an opencensus bug!)
			time.Sleep(1 * time.Millisecond)

			me := metricExtractor{}
			metricexport.NewReader().ReadAndExport(&me)

			if len(me.data) != 1 {
				t.Fatalf("Expected exactly one metric: %+v", me.data)
			}
			if len(me.data[0].TimeSeries) != 1 {
				t.Errorf("Expected exactly one row: %+v", me.data[0].TimeSeries)
			}

			if tc.expectedResource != nil {
				if diff := cmp.Diff(tc.expectedResource, me.data[0].Resource.Labels); diff != "" {
					t.Errorf("Wrong resource for %s (-want +got):\n%s", tc.name, diff)
				}
			}

			if tc.expectedLabels != nil {
				labels := make(map[string]string, len(me.data[0].Descriptor.LabelKeys))
				for i, k := range me.data[0].Descriptor.LabelKeys {
					if me.data[0].TimeSeries[0].LabelValues[i].Present {
						labels[k.Key] = me.data[0].TimeSeries[0].LabelValues[i].Value
					}
				}
				if diff := cmp.Diff(tc.expectedLabels, labels); diff != "" {
					t.Errorf("Wrong labels for %s (-want + got):\n%s\n\n%+v", tc.name, diff, me.data[0].Resource.Labels)
				}
			}
		})
	}
}

func TestGetMetricPrefixFunc_UseKnativeDomain(t *testing.T) {
	testCases := []struct {
		name       string
		domain     string
		component  string
		metricName string
	}{{
		name:       "both resource and metric labels",
		domain:     internalServingDomain,
		component:  "activator",
		metricName: "request_count",
	}, {
		name:       "only resource labels",
		domain:     internalServingDomain,
		component:  "activator",
		metricName: "request_count",
	}, {
		name:       "resource labels overwrite metric labels",
		domain:     internalServingDomain,
		component:  "activator",
		metricName: "request_count",
	}, {
		name:       "only metric labels",
		domain:     internalServingDomain,
		component:  "activator",
		metricName: "request_count",
	}}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			knativePrefix := path.Join(testCase.domain, testCase.component)
			customPrefix := path.Join(defaultCustomMetricSubDomain, testCase.component)
			mpf := getMetricPrefixFunc(knativePrefix, customPrefix)

			if got, want := mpf(testCase.metricName), knativePrefix; got != want {
				t.Fatalf("getMetricPrefixFunc=%v, want %v", got, want)
			}
		})
	}
}

func TestGetMetricPrefixFunc_UseCustomDomain(t *testing.T) {
	testCases := []struct {
		name       string
		domain     string
		component  string
		metricName string
	}{{
		name:       "unsupported domain",
		domain:     "unsupported",
		component:  "activator",
		metricName: "request_count",
	}, {
		name:       "unsupported component",
		domain:     servingDomain,
		component:  "unsupported",
		metricName: "request_count",
	}, {
		name:       "unsupported metric",
		domain:     servingDomain,
		component:  "activator",
		metricName: "unsupported",
	}, {
		name:       "unsupported component",
		domain:     internalEventingDomain,
		component:  "unsupported",
		metricName: "event_count",
	}, {
		name:       "unsupported metric",
		domain:     internalEventingDomain,
		component:  "broker",
		metricName: "unsupported",
	}}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			knativePrefix := path.Join(testCase.domain, testCase.component)
			customPrefix := path.Join(defaultCustomMetricSubDomain, testCase.component)
			mpf := getMetricPrefixFunc(knativePrefix, customPrefix)

			if got, want := mpf(testCase.metricName), customPrefix; got != want {
				t.Fatalf("getMetricPrefixFunc=%v, want %v", got, want)
			}
		})
	}
}

func TestNewStackdriverExporterWithMetadata(t *testing.T) {
	tests := []struct {
		name          string
		config        *metricsConfig
		expectSuccess bool
	}{{
		name: "standardCase",
		config: &metricsConfig{
			domain:             servingDomain,
			component:          "autoscaler",
			backendDestination: stackdriver,
			stackdriverClientConfig: StackdriverClientConfig{
				ProjectID: testProj,
			},
		},
		expectSuccess: true,
	}, {
		name: "stackdriverClientConfigOnly",
		config: &metricsConfig{
			stackdriverClientConfig: StackdriverClientConfig{
				ProjectID:   "project",
				GCPLocation: "us-west1",
				ClusterName: "cluster",
				UseSecret:   true,
			},
		},
		expectSuccess: true,
	}, {
		name: "fullValidConfig",
		config: &metricsConfig{
			domain:                            servingDomain,
			component:                         testComponent,
			backendDestination:                stackdriver,
			reportingPeriod:                   60 * time.Second,
			isStackdriverBackend:              true,
			stackdriverMetricTypePrefix:       path.Join(servingDomain, testComponent),
			stackdriverCustomMetricTypePrefix: path.Join(customMetricTypePrefix, defaultCustomMetricSubDomain, testComponent),
			stackdriverClientConfig: StackdriverClientConfig{
				ProjectID:   "project",
				GCPLocation: "us-west1",
				ClusterName: "cluster",
				UseSecret:   true,
			},
		},
		expectSuccess: true,
	}, {
		name: "invalidStackdriverGcpLocation",
		config: &metricsConfig{
			domain:                            servingDomain,
			component:                         testComponent,
			backendDestination:                stackdriver,
			reportingPeriod:                   60 * time.Second,
			isStackdriverBackend:              true,
			stackdriverMetricTypePrefix:       path.Join(servingDomain, testComponent),
			stackdriverCustomMetricTypePrefix: path.Join(customMetricTypePrefix, defaultCustomMetricSubDomain, testComponent),
			stackdriverClientConfig: StackdriverClientConfig{
				ProjectID:   "project",
				GCPLocation: "narnia",
				ClusterName: "cluster",
				UseSecret:   true,
			},
		},
		expectSuccess: true,
	}, {
		name: "missingProjectID",
		config: &metricsConfig{
			domain:                            servingDomain,
			component:                         testComponent,
			backendDestination:                stackdriver,
			reportingPeriod:                   60 * time.Second,
			isStackdriverBackend:              true,
			stackdriverMetricTypePrefix:       path.Join(servingDomain, testComponent),
			stackdriverCustomMetricTypePrefix: path.Join(customMetricTypePrefix, defaultCustomMetricSubDomain, testComponent),
			stackdriverClientConfig: StackdriverClientConfig{
				GCPLocation: "narnia",
				ClusterName: "cluster",
				UseSecret:   true,
			},
		},
		expectSuccess: true,
	}, {
		name: "partialStackdriverConfig",
		config: &metricsConfig{
			domain:                            servingDomain,
			component:                         testComponent,
			backendDestination:                stackdriver,
			reportingPeriod:                   60 * time.Second,
			isStackdriverBackend:              true,
			stackdriverMetricTypePrefix:       path.Join(servingDomain, testComponent),
			stackdriverCustomMetricTypePrefix: path.Join(customMetricTypePrefix, defaultCustomMetricSubDomain, testComponent),
			stackdriverClientConfig: StackdriverClientConfig{
				ProjectID: "project",
			},
		},
		expectSuccess: true,
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			e, _, err := newStackdriverExporter(test.config, TestLogger(t))

			succeeded := e != nil && err == nil
			if test.expectSuccess != succeeded {
				t.Errorf("Unexpected test result. Expected success? [%v]. Error: [%v]", test.expectSuccess, err)
			}
		})
	}
}

func TestEnsureKubeClient(t *testing.T) {
	// Even though ensureKubeclient uses sync.Once, make sure if the first run failed, it returns an error on subsequent calls.
	for i := 0; i < 3; i++ {
		err := ensureKubeclient()
		if err == nil {
			t.Error("Expected ensureKubeclient to fail due to not being in a Kubernetes cluster. Did the function run?")
		}
	}
}

func assertStringsEqual(t *testing.T, description string, expected string, actual string) {
	if expected != actual {
		t.Errorf("Expected %v to be set correctly. Want [%v], Got [%v]", description, expected, actual)
	}
}

func TestSetStackdriverSecretLocation(t *testing.T) {
	// Prevent pollution from other tests
	useStackdriverSecretEnabled = false
	// Reset global state after test
	defer func() {
		secretName = StackdriverSecretNameDefault
		secretNamespace = StackdriverSecretNamespaceDefault
		useStackdriverSecretEnabled = false
	}()

	const testName, testNamespace = "test-name", "test-namespace"
	secretFetcher := func(name string) (*corev1.Secret, error) {
		return &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testName,
				Namespace: testNamespace,
			},
		}, nil
	}

	ctx := context.Background()

	// Default checks
	assertStringsEqual(t, "DefaultSecretName", secretName, StackdriverSecretNameDefault)
	assertStringsEqual(t, "DefaultSecretNamespace", secretNamespace, StackdriverSecretNamespaceDefault)
	sec, err := getStackdriverSecret(ctx, secretFetcher)
	if err != nil {
		t.Error("Got unexpected error when getting secret:", err)
	}
	if sec != nil {
		t.Errorf("Stackdriver secret should not be fetched unless SetStackdriverSecretLocation has been called")
	}

	// Once SetStackdriverSecretLocation has been called, attempts to get the secret should complete.
	SetStackdriverSecretLocation(testName, testNamespace)
	sec, err = getStackdriverSecret(ctx, secretFetcher)
	if err != nil {
		t.Error("Got unexpected error when getting secret:", err)
	}
	if sec == nil {
		t.Error("expected secret to be non-nil if there is no error and SetStackdriverSecretLocation has been called")
	}
	assertStringsEqual(t, "secretName", secretName, testName)
	assertStringsEqual(t, "secretNamespace", secretNamespace, testNamespace)
}
