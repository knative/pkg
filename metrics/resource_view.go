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
	"strings"
	"sync"
	"time"

	"go.opencensus.io/metric/metricdata"
	"go.opencensus.io/metric/metricproducer"
	"go.opencensus.io/resource"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

type storedViews struct {
	views []*view.View
	lock  sync.Mutex
}

type meterExporter struct {
	m view.Meter    // NOTE: DO NOT RETURN THIS DIRECTLY; the view.Meter will not work for the empty Resource
	o stats.Options // Cache the option to reduce allocations
	e view.Exporter
}

// ResourceExporterFactory provides a hook for producing separate view.Exporters
// for each observed Resource. This is needed because OpenCensus support for
// Resources is a bit tacked-on rather than being a first-class component like
// Tags are.
type ResourceExporterFactory func(*resource.Resource) (view.Exporter, error)
type meters struct {
	meters  map[string]*meterExporter
	factory ResourceExporterFactory
	// Cache of Resource pointers from metricskey to Meters, to avoid
	// unnecessary stringify operations
	resourceToKey map[*resource.Resource]string
	lock          sync.Mutex
}

var resourceViews storedViews = storedViews{}
var allMeters meters = meters{
	meters:        map[string]*meterExporter{"": &defaultMeter},
	resourceToKey: map[*resource.Resource]string{nil: ""},
}

// RegisterResourceView is similar to view.Register(), except that it will
// register the view across all Resources tracked by the system, rather than
// simply the default view.
func RegisterResourceView(views ...*view.View) error {
	resourceViews.lock.Lock()
	defer resourceViews.lock.Unlock()

	allMeters.lock.Lock()
	defer allMeters.lock.Unlock()

	errors := make([]error, 0, len(allMeters.meters))

	for _, meter := range allMeters.meters {
		if err := meter.m.Register(views...); err != nil {
			errors = append(errors, err)
		}
	}
	if len(errors) > 0 {
		return errors[0] // The first error is as good as any
	}

	resourceViews.views = append(resourceViews.views, views...)
	return nil
}

func setFactory(f func(*resource.Resource) (view.Exporter, error)) error {
	if f == nil {
		return fmt.Errorf("do not setFactory(nil)!")
	}

	allMeters.lock.Lock()
	defer allMeters.lock.Unlock()

	allMeters.factory = f

	errs := make([]error, len(allMeters.meters))

	for r, meter := range allMeters.meters {
		e, err := f(resourceFromString(r))
		if err != nil {
			errs = append(errs, err)
			continue
		}
		meter.m.RegisterExporter(e)
		meter.m.UnregisterExporter(meter.e)
		meter.e = e
	}
	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

func meterExporterForResource(r *resource.Resource) *meterExporter {
	key, ok := allMeters.resourceToKey[r]
	if !ok {
		key = resourceAsString(r)
		allMeters.resourceToKey[r] = key
	}

	mE := allMeters.meters[key]
	if mE == nil {
		mE = &meterExporter{}
		allMeters.meters[key] = mE
	}
	if mE.o != nil {
		return mE
	}
	mE.m = view.NewMeter()
	mE.m.Start()
	mE.m.Register(resourceViews.views...)
	// Prometheus's default collector uses the global metricproducer to read *all* meters.
	// This confuses the default prometheus adapter, so remove these from the global export.
	metricproducer.GlobalManager().DeleteProducer(mE.m.(metricproducer.Producer))
	mE.o = stats.WithRecorder(mE.m)
	allMeters.meters[key] = mE
	return mE
}

// meterForResource finds or creates a view.Meter for the given resource. If
func meterForResource(r *resource.Resource) view.Meter {
	mE := meterExporterForResource(r)
	if mE == nil {
		return nil
	}
	return mE.m
}

// optionForResource finds or creates a stats.Option indicating which meter to record to.
func optionForResource(r *resource.Resource) (stats.Options, error) {
	allMeters.lock.Lock()
	defer allMeters.lock.Unlock()

	mE := meterExporterForResource(r)
	if mE == nil {
		return nil, fmt.Errorf("unexpectedly failed lookup for resource %v", r)
	}

	if mE.e != nil {
		// Assume the exporter is already started.
		return mE.o, nil
	}

	if allMeters.factory == nil {
		if mE.o != nil {
			// If we can't create exporters but we have a Meter, return that.
			return mE.o, nil
		}
		return nil, fmt.Errorf("whoops, allMeters.factory is nil")
	}
	exporter, err := allMeters.factory(r)
	if err != nil {
		return nil, err
	}

	mE.m.RegisterExporter(exporter)
	mE.e = exporter
	return mE.o, nil
}

func resourceAsString(r *resource.Resource) string {
	var s strings.Builder
	l := len(r.Type)
	for k, v := range r.Labels {
		l += len(k) + len(v) + 2
	}
	s.Grow(l)
	fmt.Fprintf(&s, "%s", r.Type)
	for k, v := range r.Labels {
		fmt.Fprintf(&s, "\x01%s\x02%s", k, v)
	}
	return s.String()
}

func resourceFromString(s string) *resource.Resource {
	if s == "" {
		return nil
	}
	r := resource.Resource{Labels: map[string]string{}}
	parts := strings.Split(s, "\x01")
	r.Type = parts[0]
	for _, label := range parts[1:] {
		keyValue := strings.Split(label, "\x02")
		r.Labels[keyValue[0]] = keyValue[1]
	}
	return &r
}

// defaultMeter is a pass-through to the default worker in OpenCensus. This
// allows legacy code that uses OpenCensus and does not store a Resource in the
// context to continue to interoperate.
type defaultMeterImpl struct {
}

var defaultMeter meterExporter = meterExporter{
	m: &defaultMeterImpl{},
	o: stats.WithRecorder(nil),
	e: nil,
}

func (*defaultMeterImpl) Record(*tag.Map, interface{}, map[string]interface{}) {
	// using an empty option prevents this from being called
}

// Find calls view.Find
func (*defaultMeterImpl) Find(name string) *view.View {
	return view.Find(name)
}

// Register calls view.Register
func (*defaultMeterImpl) Register(views ...*view.View) error {
	return view.Register(views...)
}
func (*defaultMeterImpl) Unregister(views ...*view.View) {
	view.Unregister(views...)
}
func (*defaultMeterImpl) SetReportingPeriod(t time.Duration) {
	view.SetReportingPeriod(t)
}
func (*defaultMeterImpl) RegisterExporter(e view.Exporter) {
	view.RegisterExporter(e)
}
func (*defaultMeterImpl) UnregisterExporter(e view.Exporter) {
	view.UnregisterExporter(e)
}
func (*defaultMeterImpl) Start() {

}
func (*defaultMeterImpl) Stop() {

}
func (*defaultMeterImpl) RetrieveData(viewName string) ([]*view.Row, error) {
	return view.RetrieveData(viewName)
}

// Read is implemented to support casting defaultMeterImpl to a metricproducer.Producer,
// but returns no values because the prometheus exporter (which is the only consumer)
// already has a built in path which collects these metrics via metricexport, which calls
// concat(x.Read() for x in metricproducer.GlobalManager.GetAll()).
func (*defaultMeterImpl) Read() []*metricdata.Metric {
	return []*metricdata.Metric{}
}
