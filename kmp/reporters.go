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

package kmp

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/google/go-cmp/cmp"
)

// FieldListReporter implements the cmp.Reporter interface. It keeps
// track of the field names that differ between two structs and reports
// them through the Fields() function.
type FieldListReporter struct {
	path       cmp.Path
	fieldNames []string
}

// PushStep implements the cmp.Reporter.
func (r *FieldListReporter) PushStep(ps cmp.PathStep) {
	r.path = append(r.path, ps)
}

// fieldName returns the camelCase field name on the root structure based on
// the current path.
func (r *FieldListReporter) fieldName() string {
	var name string
	if len(r.path) < 2 {
		name = r.path.Index(0).String()
	} else {
		name = strings.TrimPrefix(r.path.Index(1).String(), ".")
	}
	return strings.ToLower(string(name[0])) + name[1:len(name)]
}

// Report implements the cmp.Reporter.
func (r *FieldListReporter) Report(rs cmp.Result) {
	if rs.Equal() {
		return
	}
	name := r.fieldName()
	// Only append elements we don't already have.
	for _, v := range r.fieldNames {
		if name == v {
			return
		}
	}
	r.fieldNames = append(r.fieldNames, name)
}

// PopStep implements cmp.Reporter.
func (r *FieldListReporter) PopStep() {
	r.path = r.path[:len(r.path)-1]
}

// Fields returns the field names that differed between the two
// objects after calling cmp.Equal with the FieldListReporter. Field names
// are returned in alphabetical order.
func (r *FieldListReporter) Fields() []string {
	sort.Strings(r.fieldNames)
	return r.fieldNames
}

// ShortDiffReporter implements the cmp.Reporter interface. It reports
// on fields which have diffing values in a short zero-context, unified diff
// format.
type ShortDiffReporter struct {
	path  cmp.Path
	diffs []string
	err   error
}

// PushStep implements the cmp.Reporter.
func (r *ShortDiffReporter) PushStep(ps cmp.PathStep) {
	r.path = append(r.path, ps)
}

// Report implements the cmp.Reporter.
func (r *ShortDiffReporter) Report(rs cmp.Result) {
	if rs.Equal() {
		return
	}
	cur := r.path.Last()
	vx, vy := cur.Values()
	t := cur.Type()
	var diff string
	// Prefix struct values with the types to add clarity in output
	if !vx.IsValid() || !vy.IsValid() {
		r.err = fmt.Errorf("Unable to diff %+v and %+v on path %#v", vx, vy, r.path)
	} else if t.Kind() == reflect.Struct {
		diff = fmt.Sprintf("%#v:\n\t-: %+v: \"%+v\"\n\t+: %+v: \"%+v\"\n", r.path, t, vx, t, vy)
	} else {
		diff = fmt.Sprintf("%#v:\n\t-: \"%+v\"\n\t+: \"%+v\"\n", r.path, vx, vy)
	}
	r.diffs = append(r.diffs, diff)
}

// PopStep implements the cmp.Reporter.
func (r *ShortDiffReporter) PopStep() {
	r.path = r.path[:len(r.path)-1]
}

// Diff returns the generated short diff for this object.
// cmp.Equal should be called before this method.
func (r *ShortDiffReporter) Diff() (string, error) {
	if r.err != nil {
		return "", r.err
	}
	return strings.Join(r.diffs, ""), nil
}
