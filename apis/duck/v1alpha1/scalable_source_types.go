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

package v1alpha1

import (
	"context"
	"fmt"
	"knative.dev/pkg/apis/duck/v1"
	"math"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"knative.dev/pkg/ptr"

	"knative.dev/pkg/apis"
	"knative.dev/pkg/apis/duck"
)

// KedaSource is an Implementable "duck type".
var _ duck.Implementable = (*KedaSource)(nil)

// SourceScaler specifies the minimum set of options that any scalable Source should support.
type SourceScaler struct {
	// MinScale defines the minimum scale for the source.
	// If not specified, defaults to zero.
	// +optional
	MinScale *int32 `json:"minScale,omitempty"`

	// MaxScale defines the maximum scale for the source.
	// If not specified, defaults to one.
	// +optional
	MaxScale *int32 `json:"maxScale,omitempty"`
}

const (
	// SourceScalerProvided has status True when the Source
	// has been configured with an SourceScaler.
	SourceScalerProvided apis.ConditionType = "ScalerProvided"

	// SourceScalerAnnotationKey is the annotation for the explicit class of
	// source scaler that a particular resource has opted into. For example,
	// sources.knative.dev/scaler: foo
	SourceScalerAnnotationKey = "sources.knative.dev/scaler"
)

// +genduck
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KedaSource is the minimum resource shape to adhere to in order to construct
// a scalable Source using Keda (https://github.com/kedacore/keda).
// This duck type is intended to allow implementors of KedaSources to verify
// their own resources meet the expectations.
// This is not a real resource.
type KedaSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KedaSourceSpec   `json:"spec"`
	Status KedaSourceStatus `json:"status"`
}

type KedaSourceSpec struct {
	// inherits duck.v1 SourceSpec
	v1.SourceSpec `json:",inline"`

	// Scaler defines the Keda configuration.
	// If not specified, the Source should still behave as a regular Source (i.e., be non-scalable).
	Scaler *KedaScalerSpec `json:"scaler,omitempty"`
}

// KedaScalerSpec specifies the scaler options that can be configured
// while creating a scalable Source backed by Keda.
type KedaScalerSpec struct {
	// inherits duck.v1alpha SourceScaler.
	SourceScaler `json:",inline"`

	// Type corresponds to the Keda scaler implementation. For example, for Pub/Sub is "gcp-pubsub",
	// for Kafka is "kafka", etc. This correponds to Keda's ScaledObject Trigger Type.
	// Refer to the official Keda documentation for more details: https://github.com/kedacore/keda.
	Type string `json:"type"`

	// PollingInterval refers to the interval in seconds Keda uses to poll metrics in order to inform
	// its scaling decisions. If not specified, default to 30 seconds.
	// +optional
	PollingInterval *int32 `json:"pollingInterval,omitempty"`

	// CooldownPeriod refers to the period Keda waits until it scales a Deployment down to MinScale
	// If not specified, defaults to 300 seconds (5 minutes).
	// +optional
	CooldownPeriod *int32 `json:"cooldownPeriod,omitempty"`

	// Metadata defines additional information needed to properly configure the Keda scaler.
	// This corresponds to Keda's ScaledObject Trigger Metadata.
	// Refer to the official Keda documentation for more details: https://github.com/kedacore/keda.
	Metadata map[string]string `json:"metadata"`
}

// KedaSourceStatus shows how we expect folks to embed information in
// their Status field.
type KedaSourceStatus struct {
	// inherits duck/v1 Status
	v1.SourceStatus `json:",inline"`
}

const (
	// KEDA is Keda Scaler
	KEDA = "keda.sources.knative.dev"

	// defaultMinScale is the default minimum set of Pods the SourceScaler should
	// downscale the source to.
	defaultMinScale int32 = 0
	// defaultMaxScale is the default maximum set of Pods the SourceScaler should
	// upscale the source to.
	defaultMaxScale int32 = 1
)

var (
	// Verify Source resources meet duck contracts.
	_ duck.Populatable = (*KedaSource)(nil)
	_ apis.Listable    = (*KedaSource)(nil)
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KedaSourceList is a list of KedaSource resources.
type KedaSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []KedaSource `json:"items"`
}

// GetListType implements apis.Listable
func (*KedaSource) GetListType() runtime.Object {
	return &KedaSourceList{}
}

// IsReady returns true if the resource is ready overall.
func (ss *KedaSourceStatus) IsReady() bool {
	return ss.SourceStatus.IsReady()
}

// GetFullType implements duck.Implementable
func (*KedaSource) GetFullType() duck.Populatable {
	return &KedaSource{}
}

// Populate implements duck.Populatable
func (s *KedaSource) Populate() {
	s.Spec.Sink = v1.Destination{
		URI: &apis.URL{
			Scheme:   "https",
			Host:     "tableflip.dev",
			RawQuery: "flip=mattmoor",
		},
	}
	s.Spec.CloudEventOverrides = &v1.CloudEventOverrides{
		Extensions: map[string]string{"boosh": "kakow"},
	}
	s.Spec.Scaler = &KedaScalerSpec{
		SourceScaler: SourceScaler{
			MinScale: ptr.Int32(0),
			MaxScale: ptr.Int32(1),
		},
		PollingInterval: ptr.Int32(30),
		CooldownPeriod:  ptr.Int32(300),
		Type:            "mytype",
		Metadata:        map[string]string{"myoption": "myoptionvalue"},
	}
	s.Status.ObservedGeneration = 42
	s.Status.Conditions = v1.Conditions{{
		// Populate ALL fields
		Type:               v1.SourceConditionSinkProvided,
		Status:             corev1.ConditionTrue,
		LastTransitionTime: apis.VolatileTime{Inner: metav1.NewTime(time.Date(1984, 02, 28, 18, 52, 00, 00, time.UTC))},
	}, {
		Type:               SourceScalerProvided,
		Status:             corev1.ConditionTrue,
		LastTransitionTime: apis.VolatileTime{Inner: metav1.NewTime(time.Date(1984, 02, 28, 18, 52, 00, 00, time.UTC))},
	}}
	s.Status.SinkURI = &apis.URL{
		Scheme:   "https",
		Host:     "tableflip.dev",
		RawQuery: "flip=mattmoor",
	}
}

// Validate the SourceScaler has all the necessary fields.
func (ss *SourceScaler) Validate(ctx context.Context) *apis.FieldError {
	if ss == nil {
		return nil
	}
	var errs *apis.FieldError
	if ss.MinScale == nil {
		errs = errs.Also(apis.ErrMissingField("minScale"))
	} else if *ss.MinScale < 0 {
		errs = errs.Also(apis.ErrOutOfBoundsValue(*ss.MinScale, 0, math.MaxInt32, "minScale"))
	}

	if ss.MaxScale == nil {
		errs = errs.Also(apis.ErrMissingField("maxScale"))
	} else if *ss.MaxScale < 1 {
		errs = errs.Also(apis.ErrOutOfBoundsValue(*ss.MaxScale, 1, math.MaxInt32, "maxScale"))
	}

	if ss.MinScale != nil && ss.MaxScale != nil && *ss.MaxScale < *ss.MinScale {
		errs = errs.Also(&apis.FieldError{
			Message: fmt.Sprintf("maxScale=%d is less than minScale=%d", *ss.MaxScale, *ss.MinScale),
			Paths:   []string{"maxScale", "minScale"},
		})
	}

	return errs
}

// SetDefaults sets the defaults for the SourceScaler.
func (ss *SourceScaler) SetDefault(ctx context.Context) {
	if ss == nil {
		return
	}
	if ss.MinScale == nil {
		ss.MinScale = ptr.Int32(defaultMinScale)
	}
	if ss.MaxScale == nil {
		ss.MaxScale = ptr.Int32(defaultMaxScale)
	}
}
