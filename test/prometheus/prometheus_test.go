/*
Copyright 2018 The Knative Authors

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

package prometheus_test

import (
	"context"
	"testing"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"knative.dev/pkg/test/prometheus"
)

const (
	expected = 1.0
	query    = "test"
	duration = 10 * time.Second
)

type testPromAPI struct {
	v1.API
}

// Query performs a query on the prom api
func (*testPromAPI) Query(c context.Context, query string, ts time.Time, opts ...v1.Option) (model.Value, v1.Warnings, error) {

	s := model.Sample{Value: expected}
	var v []*model.Sample
	v = append(v, &s)

	return model.Vector(v), nil, nil
}

// QueryRange performs a query for the given range.
func (*testPromAPI) QueryRange(ctx context.Context, query string, r v1.Range, opts ...v1.Option) (model.Value, v1.Warnings, error) {
	s := model.Sample{Value: expected}
	var v []*model.Sample
	v = append(v, &s)

	return model.Vector(v), nil, nil
}

func TestRunQuery(t *testing.T) {
	r, err := prometheus.RunQuery(context.Background(), t.Logf, &testPromAPI{}, query)
	if err != nil {
		t.Fatal("Error running query:", err)
	}
	if r != expected {
		t.Fatalf("Want: %f Got: %f", expected, r)
	}
}

func TestRunQueryRange(t *testing.T) {
	r := v1.Range{Start: time.Now(), End: time.Now().Add(duration)}
	val, err := prometheus.RunQueryRange(context.Background(), t.Logf, &testPromAPI{}, query, r)
	if err != nil {
		t.Fatal("Error running query:", err)
	}
	if val != expected {
		t.Fatalf("Want: %f Got: %f", expected, val)
	}
}
