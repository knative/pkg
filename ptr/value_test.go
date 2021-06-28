/*
Copyright 2021 The Knative Authors

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

func TestInt32Value(t *testing.T) {
	want := int32(55)
	gotValue := Int32Value(&want)
	if want != gotValue {
		t.Errorf("Int32() = &%v, wanted %v", gotValue, want)
	}
}

func TestInt32Value_Nil(t *testing.T) {
	want := int32(0)
	gotValue := Int32Value(nil)
	if want != gotValue {
		t.Errorf("Int32() = &%v, wanted %v", gotValue, want)
	}
}

func TestInt64Value(t *testing.T) {
	want := int64(55)
	gotValue := Int64Value(&want)
	if want != gotValue {
		t.Errorf("Int64() = &%v, wanted %v", gotValue, want)
	}
}

func TestInt64Value_Nil(t *testing.T) {
	want := int64(0)
	gotValue := Int64Value(nil)
	if want != gotValue {
		t.Errorf("Int64() = &%v, wanted %v", gotValue, want)
	}
}

func TestFloat32Value(t *testing.T) {
	want := float32(1.25)
	gotValue := Float32Value(&want)
	if want != gotValue {
		t.Errorf("Float32() = &%v, wanted %v", gotValue, want)
	}
}

func TestFloat32Value_Nil(t *testing.T) {
	want := float32(0)
	gotValue := Float32Value(nil)
	if want != gotValue {
		t.Errorf("Float32() = &%v, wanted %v", gotValue, want)
	}
}

func TestFloat64Value(t *testing.T) {
	want := 1.25
	gotValue := Float64Value(&want)
	if want != gotValue {
		t.Errorf("Float64() = &%v, wanted %v", gotValue, want)
	}
}

func TestFloat64Value_Nil(t *testing.T) {
	want := float64(0)
	gotValue := Float64Value(nil)
	if want != gotValue {
		t.Errorf("Float64() = &%v, wanted %v", gotValue, want)
	}
}


func TestBoolValue(t *testing.T) {
	want := false
	gotValue := BoolValue(nil)
	if want != gotValue {
		t.Errorf("Bool() = &%v, wanted %v", gotValue, want)
	}
}

func TestStringValue(t *testing.T) {
	want := ""
	gotValue := StringValue(nil)
	if want != gotValue {
		t.Errorf("String() = &%v, wanted %v", gotValue, want)
	}
}

func TestTimeValue(t *testing.T) {
	want := time.Time{}
	if got, want := TimeValue(nil), want; got != want {
		t.Errorf("got = %v, want: %v", got, want)
	}
}

func TestDurationValue(t *testing.T) {
	want := time.Duration(0)
	gotValue := DurationValue(nil)
	if want != gotValue {
		t.Errorf("Duration() = &%v, wanted %v", gotValue, want)
	}
}
