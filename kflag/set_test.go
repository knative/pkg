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

package kflag

import (
	"flag"
	"testing"
)

func TestSet(t *testing.T) {
	var f StringSet

	fs := flag.NewFlagSet("blah", flag.ContinueOnError)
	fs.Var(&f, "flag-name", "usage description")

	if err := fs.Parse([]string{
		"-flag-name=val1",
		"-flag-name=val1",
		"-flag-name=val2",
		"-flag-name=val3",
		"-flag-name=val1",
	}); err != nil {
		t.Fatalf("Parse() = %v", err)
	}

	if got, want := f.Value.Len(), 3; got != want {
		t.Errorf("Len() = %d, wanted %d", want, got)
	}
	if !f.Value.Has("val1") {
		t.Error("Has(val1) = false, wanted true")
	}
	if !f.Value.Has("val2") {
		t.Error("Has(val1) = false, wanted true")
	}
	if !f.Value.Has("val3") {
		t.Error("Has(val1) = false, wanted true")
	}
}

func TestEmptySet(t *testing.T) {
	var f StringSet

	fs := flag.NewFlagSet("blah", flag.ContinueOnError)
	fs.Var(&f, "flag-name", "usage description")

	if err := fs.Parse([]string{}); err != nil {
		t.Fatalf("Parse() = %v", err)
	}

	if got, want := f.Value.Len(), 0; got != want {
		t.Errorf("Len() = %d, wanted %d", want, got)
	}
	if f.Value.Has("val1") {
		t.Error("Has(val1) = true, wanted false")
	}
}
