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
	"k8s.io/utils/pointer"
	"time"
)

// Int32 is a helper for turning integers into pointers for use in
// API types that want *int32.
//
// Deprecated: Use k8s.io/utils/pointer.Int32Ptr instead.
var Int32 = pointer.Int32Ptr

// Int64 is a helper for turning integers into pointers for use in
// API types that want *int64.
//
// Deprecated: Use k8s.io/utils/pointer.Int64Ptr instead.
var Int64 = pointer.Int64Ptr

// Float32 is a helper for turning floats into pointers for use in
// API types that want *float32.
//
// Deprecated: Use k8s.io/utils/pointer.Float32Ptr instead.
var Float32 = pointer.Float32Ptr

// Float64 is a helper for turning floats into pointers for use in
// API types that want *float64.
//
// Deprecated: Use k8s.io/utils/pointer.Float64Ptr instead.
var Float64 = pointer.Float64Ptr

// Bool is a helper for turning bools into pointers for use in
// API types that want *bool.
//
// Deprecated: Use k8s.io/utils/pointer.BoolPtr instead.
var Bool = pointer.BoolPtr

// String is a helper for turning strings into pointers for use in
// API types that want *string.
//
// Deprecated: Use k8s.io/utils/pointer.StringPtr instead.
var String = pointer.StringPtr

// Duration is a helper for turning time.Duration into pointers for use in
// API types that want *time.Duration.
//
// Deprecated: Use k8s.io/utils/pointer.DurationPtr instead.
var Duration = pointer.Duration

// Time is a helper for turning a const time.Time into a pointer for use in
// API types that want *time.Time.
//
// Deprecated: Use k8s.io/utils/pointer instead.
func Time(t time.Time) *time.Time {
	return &t
}
