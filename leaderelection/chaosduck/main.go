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

// The chaosduck binary is an e2e testing tool for leader election, which loads
// the leader election configuration within the system namespace and
// periodically kills one of the leader pods for each HA component.
package main

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/injection"
	"knative.dev/pkg/injection/sharedmain"
	"knative.dev/pkg/signals"
	"knative.dev/pkg/system"
)

// components is a mapping from component name to the collection of leader pod names.
type components map[string]sets.String

// buildComponents crawls the list of leases and builds a mapping from component names
// to the set pod names that hold one or more leases.
func buildComponents(kc kubernetes.Interface) (components, error) {
	leases, err := kc.CoordinationV1().Leases(system.Namespace()).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	cs := make(components, 5)
	for _, lease := range leases.Items {
		if lease.Spec.HolderIdentity == nil {
			log.Printf("Found lease %q held by nobody!", lease.Name)
			continue
		}
		pod := strings.SplitN(*lease.Spec.HolderIdentity, "_", 2)[0]

		parts := strings.Split(pod, "-")
		if len(parts) < 3 {
			continue
		}
		deploymentName := strings.Join(parts[:len(parts)-2], "-")

		set, ok := cs[deploymentName]
		if !ok {
			set = sets.NewString()
			cs[deploymentName] = set
		}
		set.Insert(pod)
	}
	return cs, nil
}

// quack will kill one of the components leader pods.
func quack(ctx context.Context, kc kubernetes.Interface, component string, leaders sets.String) error {
	tribute, ok := leaders.PopAny()
	if !ok {
		return errors.New("this should not be possible, since components are only created when they have components.")
	}
	log.Printf("Quacking at %q leader %q", component, tribute)

	err := kc.CoreV1().Pods(system.Namespace()).Delete(tribute, &metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	return nil
}

// frequency is the frequency with which we kill off leaders.
const frequency = 30 * time.Second

func main() {
	ctx := signals.NewContext()

	// We don't expect informers to be set up, but we do expect the client to get attached to ctx.
	ctx, informers := injection.Default.SetupInformers(ctx, sharedmain.ParseAndGetConfigOrDie())
	if err := controller.StartInformers(ctx.Done(), informers...); err != nil {
		log.Fatalf("Failed to start informers %v", err)
	}
	kc := kubeclient.Get(ctx)

	// Until we are shutdown, build up an index of components and kill
	// of a leader at the specified frequency.
	for {
		select {
		case <-time.After(frequency):
			components, err := buildComponents(kc)
			if err != nil {
				log.Printf("Error building components: %v", err)
			}
			log.Printf("Got components: %#v", components)

			eg, ctx := errgroup.WithContext(ctx)
			for name, leaders := range components {
				name, leaders := name, leaders
				eg.Go(func() error {
					return quack(ctx, kc, name, leaders)
				})
			}
			if err := eg.Wait(); err != nil {
				log.Printf("Ended iteration with err: %v", err)
			}

		case <-ctx.Done():
			return
		}
	}
}
