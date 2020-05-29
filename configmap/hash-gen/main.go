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

package main

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"

	"gopkg.in/yaml.v3"
	"knative.dev/pkg/configmap"
)

func main() {
	fileName := os.Args[1]
	file, err := ioutil.ReadFile(fileName)
	if err != nil {
		log.Fatal("Failed to read file: ", err)
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(file, &doc); err != nil {
		log.Fatal("Failed to parse YAML: ", err)
	}

	if len(doc.Content) != 1 {
		log.Fatal("Can only handle singular YAML documents")
	}
	content := doc.Content[0]

	example := traverse(content, "data", configmap.ExampleKey)
	if example == nil {
		log.Print("No example field present")
		os.Exit(0)
	}

	labels := traverse(content, "metadata", "labels")
	if labels == nil {
		// TODO(markusthoemmes): Potentially handle missing metadata and labels?
		log.Fatal("'metadata.labels' not found")
	}

	hash := configmap.ExampleHash(example.Value)
	existingLabel := child(labels, configmap.ExampleHashLabel)
	if existingLabel != nil {
		existingLabel.Value = hash
	} else {
		labels.Content = append(labels.Content,
			strNode(configmap.ExampleHashLabel),
			strNode(hash))
	}

	buffer := &bytes.Buffer{}
	encoder := yaml.NewEncoder(buffer)
	encoder.SetIndent(2)
	if err := encoder.Encode(&doc); err != nil {
		log.Fatal("Failed to encode YAML: ", err)
	}

	if err := ioutil.WriteFile(fileName, buffer.Bytes(), 0644); err != nil {
		log.Fatal("Failed to write file: ", err)
	}
}

func traverse(parent *yaml.Node, path ...string) *yaml.Node {
	if parent == nil {
		return nil
	}
	if len(path) == 0 {
		return parent
	}
	tail := path[1:]
	child := child(parent, path[0])
	return traverse(child, tail...)
}

func child(parent *yaml.Node, key string) *yaml.Node {
	if parent == nil {
		return nil
	}
	for i := range parent.Content {
		if parent.Content[i].Value == key {
			if len(parent.Content) < i+1 {
				return nil
			}
			return parent.Content[i+1]
		}
	}
	return nil
}

func strNode(value string) *yaml.Node {
	return &yaml.Node{
		Kind:  yaml.ScalarNode,
		Tag:   "!!str",
		Value: value,
	}
}
