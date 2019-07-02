/*
Copyright 2018 The Knative Authors

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
	"strconv"
	"time"

	"github.com/knative/pkg/metrics"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
)

const (
	requestCountName     = "request_count"
	requestLatenciesName = "request_latencies"
)

var (
	requestCountM = stats.Int64(
		requestCountName,
		"The number of requests that are routed to webhook",
		stats.UnitDimensionless)
	responseTimeInMsecM = stats.Float64(
		"request_latencies",
		"The response time in milliseconds",
		stats.UnitMilliseconds)

	defaultLatencyDistribution = view.Distribution(0, 5, 10, 20, 40, 60, 80, 100, 150, 200, 250, 300, 350, 400, 450, 500, 600, 700, 800, 900, 1000, 2000, 5000, 10000, 20000, 50000, 100000)

	// Create the tag keys that will be used to add tags to our measurements.
	// Tag keys must conform to the restrictions described in
	// go.opencensus.io/tag/validate.go. Currently those restrictions are:
	// - length between 1 and 255 inclusive
	// - characters are printable US-ASCII
	requestOperationKey  = mustNewTagKey("request_operation")
	kindGroupKey         = mustNewTagKey("kind_group")
	kindVersionKey       = mustNewTagKey("kind_version")
	kindKindKey          = mustNewTagKey("kind_kind")
	resourceGroupKey     = mustNewTagKey("resource_group")
	resourceVersionKey   = mustNewTagKey("resource_version")
	resourceResourceKey  = mustNewTagKey("resource_resource")
	resourceNameKey      = mustNewTagKey("resource_name")
	resourceNamespaceKey = mustNewTagKey("resource_namespace")
	admissionAllowedKey  = mustNewTagKey("admission_allowed")
)

func init() {
	tagKeys := []tag.Key{
		requestOperationKey,
		kindGroupKey,
		kindVersionKey,
		kindKindKey,
		resourceGroupKey,
		resourceVersionKey,
		resourceResourceKey,
		resourceNamespaceKey,
		resourceNameKey,
		admissionAllowedKey}

	err := view.Register(
		&view.View{
			Description: requestCountM.Description(),
			Measure:     requestCountM,
			Aggregation: view.Count(),
			TagKeys:     tagKeys,
		},
		&view.View{
			Description: responseTimeInMsecM.Description(),
			Measure:     responseTimeInMsecM,
			Aggregation: defaultLatencyDistribution,
			TagKeys:     tagKeys,
		},
	)
	if err != nil {
		panic(err)
	}
}

// StatsReporter reports webhook metrics
type StatsReporter interface {
	ReportRequest(request *admissionv1beta1.AdmissionRequest, response *admissionv1beta1.AdmissionResponse, d time.Duration) error
}

// reporter implements StatsReporter interface
type reporter struct {
	ctx context.Context
}

// NewStatsReporter creaters a reporter for webhook metrics
func NewStatsReporter() (StatsReporter, error) {
	ctx, err := tag.New(
		context.Background(),
	)
	if err != nil {
		return nil, err
	}

	return &reporter{ctx: ctx}, nil
}

// Captures req count metric, recording the count and the duration
func (r *reporter) ReportRequest(req *admissionv1beta1.AdmissionRequest, resp *admissionv1beta1.AdmissionResponse, d time.Duration) error {
	ctx, err := tag.New(
		r.ctx,
		tag.Insert(requestOperationKey, string(req.Operation)),
		tag.Insert(kindGroupKey, req.Kind.Group),
		tag.Insert(kindVersionKey, req.Kind.Version),
		tag.Insert(kindKindKey, req.Kind.Kind),
		tag.Insert(resourceGroupKey, req.Resource.Group),
		tag.Insert(resourceVersionKey, req.Resource.Version),
		tag.Insert(resourceResourceKey, req.Resource.Resource),
		tag.Insert(resourceNameKey, req.Name),
		tag.Insert(resourceNamespaceKey, req.Namespace),
		tag.Insert(admissionAllowedKey, strconv.FormatBool(resp.Allowed)),
	)
	if err != nil {
		return err
	}

	metrics.Record(ctx, requestCountM.M(1))
	// Convert time.Duration in nanoseconds to milliseconds
	metrics.Record(ctx, responseTimeInMsecM.M(float64(d/time.Millisecond)))
	return nil
}

func mustNewTagKey(s string) tag.Key {
	tagKey, err := tag.NewKey(s)
	if err != nil {
		panic(err)
	}
	return tagKey
}
