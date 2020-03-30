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
	"context"

	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	"k8s.io/client-go/kubernetes"
)

// Context wraps information relevant for validation of the incoming request along with the generic context.Context.
type Context struct {
	// CallCtx is the context from the user-request.
	CallCtx context.Context

	// Operation is the operation being performed. This may be different than the operation
	// requested. e.g. a patch can result in either a CREATE or UPDATE Operation.
	Operation admissionv1beta1.Operation

	// DryRun indicates that this request will not be persisted.
	// Any operations with persistent side-effects should be skipped.
	DryRun bool

	// KubeClient is a client for interacting with k8s APIs.
	KubeClient kubernetes.Interface
}

// NewValidationContext copies relevant objects from the request to a single structure to be read during validation.
func NewValidationContext(ctx context.Context, ac *reconciler, req *admissionv1beta1.AdmissionRequest) (vc *Context) {
	dryRun := false
	if req.DryRun != nil && *req.DryRun {
		dryRun = true
	}

	vc = &Context{
		CallCtx:    ctx,
		Operation:  req.Operation,
		DryRun:     dryRun,
		KubeClient: ac.client,
	}
	return
}
