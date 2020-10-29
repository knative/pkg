/*
Copyright 2020 The Knative Authors

Licensed under the Apache License, Veroute.on 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package reconciler

import (
	"errors"
	"testing"

	apierrs "k8s.io/apimachinery/pkg/api/errors"

	v1 "k8s.io/api/autoscaling/v1"
)

func TestRetryUpdateConflicts(t *testing.T) {
	errAny := errors.New("foo")
	errConflict := apierrs.NewConflict(v1.Resource("foo"), "bar", errAny)

	tests := []struct {
		name         string
		returns      []error
		want         error
		wantAttempts int
	}{{
		name:         "all good",
		returns:      []error{nil},
		want:         nil,
		wantAttempts: 1,
	}, {
		name:         "not retry on non-conflict error",
		returns:      []error{errAny},
		want:         errAny,
		wantAttempts: 1,
	}, {
		name:         "retry up to 5 times on conflicts",
		returns:      []error{errConflict, errConflict, errConflict, errConflict, errConflict, errConflict},
		want:         errConflict,
		wantAttempts: 5,
	}, {
		name:         "eventually succeed",
		returns:      []error{errConflict, errConflict, nil},
		want:         nil,
		wantAttempts: 3,
	}, {
		name:         "eventually fail",
		returns:      []error{errConflict, errConflict, errAny},
		want:         errAny,
		wantAttempts: 3,
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			attempts := 0
			got := RetryUpdateConflicts(func(i int) error {
				attempts++
				return test.returns[i]
			})

			if !errors.Is(got, test.want) {
				t.Errorf("RetryUpdateConflicts() = %v, want %v", got, test.want)
			}
			if attempts != test.wantAttempts {
				t.Errorf("attempts = %d, want %d", attempts, test.wantAttempts)
			}
		})
	}
}

func TestRetryTestErrors(t *testing.T) {
	errAny := errors.New("foo")
	errGKE := errors.New(`operation cannot be fulfilled on resourcequotas "gke-resource-quotas": StorageError: invalid object, Code: 4, Key: /registry/resourcequotas/serving-tests/gke-resource-quotas, ResourceVersion: 0, AdditionalErrorMsg: Precondition failed: UID in precondition: 7aaedbdf-caa8-41e7-94cb-f8c053038e86, UID in object meta`)
	errEtcd := errors.New("etcdserver: request timed out")
	errConflict := apierrs.NewConflict(v1.Resource("foo"), "bar", errAny)

	tests := []struct {
		name         string
		returns      []error
		want         error
		wantAttempts int
	}{{
		name:         "all good",
		returns:      []error{nil},
		want:         nil,
		wantAttempts: 1,
	}, {
		name:         "not retry on non-conflict error",
		returns:      []error{errAny},
		want:         errAny,
		wantAttempts: 1,
	}, {
		name:         "retry up to 5 times",
		returns:      []error{errConflict, errGKE, errEtcd, errConflict, errGKE, errEtcd},
		want:         errGKE,
		wantAttempts: 5,
	}, {
		name:         "eventually succeed",
		returns:      []error{errConflict, errGKE, errEtcd, nil},
		want:         nil,
		wantAttempts: 4,
	}, {
		name:         "eventually fail",
		returns:      []error{errConflict, errConflict, errAny},
		want:         errAny,
		wantAttempts: 3,
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			attempts := 0
			got := RetryTestErrors(func(i int) error {
				attempts++
				return test.returns[i]
			})

			if !errors.Is(got, test.want) {
				t.Errorf("RetryTestErrors() = %v, want %v", got, test.want)
			}
			if attempts != test.wantAttempts {
				t.Errorf("attempts = %d, want %d", attempts, test.wantAttempts)
			}
		})
	}
}
