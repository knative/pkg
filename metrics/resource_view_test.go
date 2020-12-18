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
	"time"

	"go.opencensus.io/resource"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"k8s.io/apimachinery/pkg/util/clock"
)

var (
	r = resource.Resource{Labels: map[string]string{"foo": "bar"}}
)

func TestRegisterResourceView(t *testing.T) {
	meter := meterExporterForResource(&r).m

	m := stats.Int64("testView_sum", "", stats.UnitDimensionless)
	view := view.View{Name: "testView", Measure: m, Aggregation: view.Sum()}

	err := RegisterResourceView(&view)
	if err != nil {
		t.Fatal("RegisterResourceView =", err)
	}
	t.Cleanup(func() { UnregisterResourceView(&view) })

	viewToFind := defaultMeter.m.Find("testView")
	if viewToFind == nil || viewToFind.Name != "testView" {
		t.Error("Registered view should be found in default meter, instead got", viewToFind)
	}

	viewToFind = meter.Find("testView")
	if viewToFind == nil || viewToFind.Name != "testView" {
		t.Error("Registered view should be found in new meter, instead got", viewToFind)
	}
}

func TestOptionForResource(t *testing.T) {
	option, err1 := optionForResource(&r)
	if err1 != nil {
		t.Error("Should succeed getting option, instead got error", err1)
	}
	optionAgain, err2 := optionForResource(&r)
	if err2 != nil {
		t.Error("Should succeed getting option, instead got error", err2)
	}

	if fmt.Sprintf("%v", optionAgain) != fmt.Sprintf("%v", option) {
		t.Errorf("Option for the same resource should not be recreated, instead got %v and %v", optionAgain, option)
	}
}

type testExporter struct {
	view.Exporter
	id string
}

func TestSetFactory(t *testing.T) {
	var oldFactory ResourceExporterFactory
	func() {
		allMeters.lock.Lock()
		defer allMeters.lock.Unlock()

		oldFactory = allMeters.factory
	}()

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
		t.Error("Should succeed getting option, instead got error", err)
	}

	// Now get the exporter and verify the id
	me := meterExporterForResource(&resource123)
	e := me.e.(*testExporter)
	if e.id != "123" {
		t.Error("Expect id to be 123, instead got", e.id)
	}

	resource456 := r
	resource456.Labels["id"] = "456"
	// Create the new meter and apply the factory
	_, err = optionForResource(&resource456)
	if err != nil {
		t.Error("Should succeed getting option, instead got error", err)
	}

	me = meterExporterForResource(&resource456)
	e = me.e.(*testExporter)
	if e.id != "456" {
		t.Error("Expect id to be 456, instead got", e.id)
	}

	setFactory(oldFactory)
}

func TestAllMetersExpiration(t *testing.T) {
	allMeters.clock = clock.Clock(clock.NewFakeClock(time.Now()))
	var fakeClock *clock.FakeClock = allMeters.clock.(*clock.FakeClock)
	ClearMetersForTest() // t+0m

	// Add resource123
	resource123 := r
	resource123.Labels["id"] = "123"
	_, err := optionForResource(&resource123)
	if err != nil {
		t.Error("Should succeed getting option, instead got error ", err)
	}
	// (123=0m, 456=Inf)

	// Bump time to make resource123's expiry offset from resource456
	fakeClock.Step(90 * time.Second) // t+1.5m
	// (123=0m, 456=Inf)

	// Add 456
	resource456 := r
	resource456.Labels["id"] = "456"
	_, err = optionForResource(&resource456)
	if err != nil {
		t.Error("Should succeed getting option, instead got error ", err)
	}
	allMeters.lock.Lock()
	if len(allMeters.meters) != 3 {
		t.Errorf("len(allMeters)=%d, want: 3", len(allMeters.meters))
	}
	allMeters.lock.Unlock()
	// (123=1.5m, 456=0m)

	// Warm up the older entry
	fakeClock.Step(90 * time.Second) //t+3m
	// (123=4.5m, 456=3m)

	// Refresh the first entry
	_, err = optionForResource(&resource123)
	if err != nil {
		t.Error("Should succeed getting option, instead got error ", err)
	}
	// (123=0, 456=1.5m)

	// Expire the second entry
	fakeClock.Step(9 * time.Minute) // t+12m
	time.Sleep(time.Second)         // Wait a second on the wallclock, so that the cleanup thread has time to finish a loop
	allMeters.lock.Lock()
	if len(allMeters.meters) != 2 {
		t.Errorf("len(allMeters)=%d, want: 2", len(allMeters.meters))
	}
	allMeters.lock.Unlock()
	// (123=9m, 456=10.5m)
	// non-expiring defaultMeter was just tested

	// Add resource789
	resource789 := r
	resource789.Labels["id"] = "789"
	_, err = optionForResource(&resource789)
	if err != nil {
		t.Error("Should succeed getting option, instead got error ", err)
	}
	// (123=9m, 456=evicted, 789=0m)
}

func TestResourceAsString(t *testing.T) {
	r1 := &resource.Resource{Type: "foobar", Labels: map[string]string{"k1": "v1", "k3": "v3", "k2": "v2"}}
	r2 := &resource.Resource{Type: "foobar", Labels: map[string]string{"k2": "v2", "k3": "v3", "k1": "v1"}}
	r3 := &resource.Resource{Type: "foobar", Labels: map[string]string{"k1": "v1", "k2": "v2", "k4": "v4"}}

	// Test 5 time since the iteration could be random.
	for i := 0; i < 5; i++ {

		if s1, s2 := resourceToKey(r1), resourceToKey(r2); s1 != s2 {
			t.Errorf("Expect same resources, but got %q and %q", s1, s2)
		}
	}

	if s1, s3 := resourceToKey(r1), resourceToKey(r3); s1 == s3 {
		t.Error("Expect different resources, but got the same", s1)
	}
}

func BenchmarkResourceToKey(b *testing.B) {
	for _, count := range []int{0, 1, 5, 10} {
		labels := make(map[string]string, count)
		for i := 0; i < count; i++ {
			labels[fmt.Sprint("key", i)] = fmt.Sprint("value", i)
		}
		r := &resource.Resource{Type: "foobar", Labels: labels}

		b.Run(fmt.Sprintf("%d-labels", count), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				resourceToKey(r)
			}
		})
	}
}
