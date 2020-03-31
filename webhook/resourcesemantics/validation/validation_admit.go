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

package validation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"go.uber.org/zap"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/webhook"
	"knative.dev/pkg/webhook/resourcesemantics"
)

var errMissingNewObject = errors.New("the new object may not be nil")

// Callback is a generic function to be called by a consumer of validation
type Callback func(ctx context.Context, unstructured *unstructured.Unstructured, dryRun bool, opVerb admissionv1beta1.Operation) error

var _ webhook.AdmissionController = (*reconciler)(nil)

// Admit implements AdmissionController
func (ac *reconciler) Admit(ctx context.Context, request *admissionv1beta1.AdmissionRequest) *admissionv1beta1.AdmissionResponse {
	if ac.withContext != nil {
		ctx = ac.withContext(ctx)
	}

	kind := request.Kind
	gvk := schema.GroupVersionKind{
		Group:   kind.Group,
		Version: kind.Version,
		Kind:    kind.Kind,
	}

	ctx, resource, err := ac.decodeRequestAndPrepareContext(ctx, request, gvk)
	if err != nil {
		return webhook.MakeErrorStatus("decoding request failed: %v", err)
	}

	err = validate(ctx, resource, request)
	if err != nil {
		return webhook.MakeErrorStatus("validation failed: %v", err)
	}

	if err := ac.callback(ctx, request, gvk); err != nil {
		return webhook.MakeErrorStatus("validation callback failed: %v", err)
	}

	return &admissionv1beta1.AdmissionResponse{Allowed: true}
}

// decodeRequest deserializes the old and new GenericCrds from the incoming request and sets up the context.
// nil oldObj or newObj denote absence of `old` (create) or `new` (delete) objects.
func (ac *reconciler) decodeRequestAndPrepareContext(
	ctx context.Context,
	req *admissionv1beta1.AdmissionRequest,
	gvk schema.GroupVersionKind) (context.Context, resourcesemantics.GenericCRD, error) {

	logger := logging.FromContext(ctx)
	handler, ok := ac.handlers[gvk]
	if !ok {
		logger.Errorf("Unhandled kind: %v", gvk)
		return ctx, nil, fmt.Errorf("unhandled kind: %v", gvk)
	}

	newBytes := req.Object.Raw
	oldBytes := req.OldObject.Raw

	// Decode json to a GenericCRD
	var newObj resourcesemantics.GenericCRD
	if len(newBytes) != 0 {
		newObj = handler.DeepCopyObject().(resourcesemantics.GenericCRD)
		newDecoder := json.NewDecoder(bytes.NewBuffer(newBytes))
		if ac.disallowUnknownFields {
			newDecoder.DisallowUnknownFields()
		}
		if err := newDecoder.Decode(&newObj); err != nil {
			return ctx, nil, fmt.Errorf("cannot decode incoming new object: %v", err)
		}
	}

	var oldObj resourcesemantics.GenericCRD
	if len(oldBytes) != 0 {
		oldObj = handler.DeepCopyObject().(resourcesemantics.GenericCRD)
		oldDecoder := json.NewDecoder(bytes.NewBuffer(oldBytes))
		if ac.disallowUnknownFields {
			oldDecoder.DisallowUnknownFields()
		}
		if err := oldDecoder.Decode(&oldObj); err != nil {
			return ctx, nil, fmt.Errorf("cannot decode incoming old object: %v", err)
		}
	}

	// Set up the context for validation
	if oldObj != nil {
		if req.SubResource == "" {
			ctx = apis.WithinUpdate(ctx, oldObj)
		} else {
			ctx = apis.WithinSubResourceUpdate(ctx, oldObj, req.SubResource)
		}
	} else {
		ctx = apis.WithinCreate(ctx)
	}
	ctx = apis.WithUserInfo(ctx, &req.UserInfo)
	ctx = apis.WithKubeClient(ctx, ac.client)
	return ctx, newObj, nil
}

func validate(ctx context.Context, resource resourcesemantics.GenericCRD, req *admissionv1beta1.AdmissionRequest) error {
	logger := logging.FromContext(ctx)

	// Only run validation for supported create and update validaiton.
	switch req.Operation {
	case admissionv1beta1.Create, admissionv1beta1.Update:
		// Supported verbs
	default:
		logger.Infof("Unhandled webhook validation operation, letting it through %v", req.Operation)
		return nil
	}

	// None of the validators will accept a nil value for newObj.
	if resource == nil {
		return errMissingNewObject
	}

	if err := resource.Validate(ctx); err != nil {
		logger.Errorw("Failed the resource specific validation", zap.Error(err))
		// Return the error message as-is to give the validation callback
		// discretion over (our portion of) the message that the user sees.
		return err
	}

	return nil
}

// callback runs optional callbacks on admission
func (ac *reconciler) callback(ctx context.Context, req *admissionv1beta1.AdmissionRequest, gvk schema.GroupVersionKind) error {
	var toDecode []byte
	if req.Operation == admissionv1beta1.Delete {
		toDecode = req.OldObject.Raw
	} else {
		toDecode = req.Object.Raw
	}
	if toDecode == nil {
		logger := logging.FromContext(ctx)
		logger.Errorf("No incoming object found: %v for verb %v", gvk, req.Operation)
		return nil
	}

	// Generically callback if any are provided for the resource.
	if callback, ok := ac.callbacks[gvk]; ok {
		unstruct := &unstructured.Unstructured{}
		newDecoder := json.NewDecoder(bytes.NewBuffer(toDecode))
		if err := newDecoder.Decode(&unstruct); err != nil {
			return fmt.Errorf("cannot decode incoming new object: %w", err)
		}

		dryRun := false
		if req.DryRun != nil {
			dryRun = *req.DryRun
		}

		if err := callback(ctx, unstruct, dryRun, req.Operation); err != nil {
			return err
		}
	}

	return nil
}
