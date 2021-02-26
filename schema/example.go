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

package main

import (
	"log"

	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/schema/commands"
	"knative.dev/pkg/schema/example"
	"knative.dev/pkg/schema/registry"
)

// This is a demo of what the CLI looks like, copy and implement your own.
func main() {
	registry.Register("LaremIpsum", example.LaremIpsum{})
	registry.Register("Addressable", duckv1.AddressableType{})
	registry.Register("Binding", duckv1.Binding{})
	registry.Register("Source", duckv1.Source{})
	registry.Register("KResource", duckv1.KResource{})

	if err := commands.New("knative.dev/pkg").Execute(); err != nil {
		log.Fatal("Error during command execution: ", err)
	}
}
