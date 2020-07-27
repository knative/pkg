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

package ptr

import (
	"testing"
	"time"
)

func TestInt32(t *testing.T) {
	const want = 55
	gotPtr := Int32(want)
	if want != *gotPtr {
		t.Errorf("Int32() = &%v, wanted %v", *gotPtr, want)
	}
}

func TestInt64(t *testing.T) {
	const want = 55
	gotPtr := Int64(want)
	if want != *gotPtr {
		t.Errorf("Int64() = &%v, wanted %v", *gotPtr, want)
	}
}

func TestFloat32(t *testing.T) {
	const want = 1.25
	gotPtr := Float32(want)
	if want != *gotPtr {
		t.Errorf("Float32() = &%v, wanted %v", *gotPtr, want)
	}
}

func TestFloat64(t *testing.T) {
	const want = 1.25
	gotPtr := Float64(want)
	if want != *gotPtr {
		t.Errorf("Float64() = &%v, wanted %v", *gotPtr, want)
	}
}

func TestBool(t *testing.T) {
	const want = true
	gotPtr := Bool(want)
	if want != *gotPtr {
		t.Errorf("Bool() = &%v, wanted %v", *gotPtr, want)
	}
}

func TestString(t *testing.T) {
	const want = "should be a pointer"
	gotPtr := String(want)
	if want != *gotPtr {
		t.Errorf("String() = &%v, wanted %v", *gotPtr, want)
	}
}

func TestTime(t *testing.T) {
	want := time.Now().Add(time.Minute)
	if got, want := *Time(want), want; got != want {
		t.Errorf("got = %v, want: %v", got, want)
	}
}

func TestDurationWithConst(t *testing.T) {
	const want = 42 * time.Second
	gotPtr := Duration(want)
	if want != *gotPtr {
		t.Errorf("Duration() = &%v, wanted %v", *gotPtr, want)
	}
}
