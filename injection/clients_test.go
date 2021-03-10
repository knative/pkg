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

package injection

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"k8s.io/client-go/rest"
)

func injectFoo(ctx context.Context, cfg *rest.Config) context.Context {
	return ctx
}

func injectBar(ctx context.Context, cfg *rest.Config) context.Context {
	return ctx
}

func TestRegisterClient(t *testing.T) {
	i := &impl{}

	if want, got := 0, len(i.GetClients()); got != want {
		t.Errorf("GetClients() = %d, wanted %d", want, got)
	}

	i.RegisterClient(injectFoo)

	if want, got := 1, len(i.GetClients()); got != want {
		t.Errorf("GetClients() = %d, wanted %d", want, got)
	}

	i.RegisterClient(injectBar)

	if want, got := 2, len(i.GetClients()); got != want {
		t.Errorf("GetClients() = %d, wanted %d", want, got)
	}
}

type fakeClient struct {
	Name string
}

func TestRegisterClientFetcher(t *testing.T) {
	i := &impl{}

	fakeA := fakeClient{Name: "a"}
	fetchA := func(ctx context.Context) interface{} {
		return fakeA
	}

	fakeB := fakeClient{Name: "b"}
	fetchB := func(ctx context.Context) interface{} {
		return fakeB
	}

	ctx := context.Background()
	if want, got := 0, len(i.FetchAllClients(ctx)); got != want {
		t.Errorf("FetchAllClients() = %d, wanted %d", got, want)
	}

	i.RegisterClientFetcher(fetchA)
	i.RegisterClientFetcher(fetchB)

	got := i.FetchAllClients(ctx)
	want := []interface{}{fakeA, fakeB}
	if !cmp.Equal(got, want) {
		t.Errorf("FetchAllClients() = %v, wanted %v", got, want)
	}
}
