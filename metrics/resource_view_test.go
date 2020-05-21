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
	"fmt"
	"testing"

	"go.opencensus.io/resource"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	//_ "knative.dev/pkg/metrics/testing"
)

var (
	r = resource.Resource{Labels: map[string]string{"foo": "bar"}}
)

func TestRegisterResourceView(t *testing.T) {
	meter := meterForResource(&r)

	m := stats.Int64("testView_sum", "", stats.UnitDimensionless)
	view := view.View{Name: "testView", Measure: m, Aggregation: view.Sum()}
	err := RegisterResourceView(&view)

	if err != nil {
		t.Errorf("RegisterResourceView with error %v.", err)
	}

	viewToFind := defaultMeter.m.Find("testView")
	if viewToFind == nil || viewToFind.Name != "testView" {
		t.Errorf("Registered view should be found in default meter, instead got %v", viewToFind)
	}

	viewToFind = meter.Find("testView")
	if viewToFind == nil || viewToFind.Name != "testView" {
		t.Errorf("Registered view should be found in new meter, instead got %v", viewToFind)
	}
}

func TestMeterForResource(t *testing.T) {
	meter := meterForResource(&r)
	if meter == nil {
		t.Error("Should succeed getting meter, instead got nil")
	}
	meterAgain := meterForResource(&r)
	if meterAgain == nil {
		t.Error("Should succeed getting meter, instead got nil")
	}

	if meterAgain != meter {
		t.Error("Meter for the same resource should not be recreated")
	}
}

func TestOptionForResource(t *testing.T) {
	option, err1 := optionForResource(&r)
	if err1 != nil {
		t.Errorf("Should succeed getting option, instead got error %v", err1)
	}
	optionAgain, err2 := optionForResource(&r)
	if err2 != nil {
		t.Errorf("Should succeed getting option, instead got error %v", err2)
	}

	if fmt.Sprintf("%v", optionAgain) != fmt.Sprintf("%v", option) {
		t.Errorf("Option for the same resource should not be recreated, instead got %v and %v", optionAgain, option)
	}
}

type testExporter struct {
	id string
}

func (fe *testExporter) ExportView(vd *view.Data) {}
func (fe *testExporter) Flush()                   {}
func TestSetFactor(t *testing.T) {
	fakeFactory := func(rr *resource.Resource) (view.Exporter, error) {
		if rr == nil {
			return &testExporter{}, nil
		}

		return &testExporter{id: rr.Labels["id"]}, nil
	}

	resource123 := r
	resource123.Labels["id"] = "123"

	setFactory(fakeFactory)
	// Create the new meter and apply the factory
	_, err := optionForResource(&resource123)
	if err != nil {
		t.Errorf("Should succeed getting option, instead got error %v", err)
	}

	// Now get the exporter and verify the id
	me := meterExporterForResource(&resource123)
	e := me.e.(*testExporter)
	if e.id != "123" {
		t.Errorf("Expect id to be 123, instead got %v", e.id)
	}

	resource456 := r
	resource456.Labels["id"] = "456"
	// Create the new meter and apply the factory
	_, err = optionForResource(&resource456)
	if err != nil {
		t.Errorf("Should succeed getting option, instead got error %v", err)
	}

	me = meterExporterForResource(&resource456)
	e = me.e.(*testExporter)
	if e.id != "456" {
		t.Errorf("Expect id to be 456, instead got %v", e.id)
	}
}
