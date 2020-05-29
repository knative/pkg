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
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"gopkg.in/yaml.v3"
	"knative.dev/pkg/configmap"
)

func main() {
	fileName := os.Args[1]
	if err := processFile(fileName); err != nil {
		log.Fatal(err)
	}
}

func processFile(fileName string) error {
	in, err := ioutil.ReadFile(fileName)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	out, err := process(in)
	if out == nil || err != nil {
		return err
	}

	if err := ioutil.WriteFile(fileName, out, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	return nil
}

func process(data []byte) ([]byte, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}
	content := doc.Content[0]

	example := traverse(content, "data", configmap.ExampleKey)
	if example == nil {
		return nil, nil
	}

	labels := traverse(content, "metadata", "labels")
	if labels == nil {
		// TODO(markusthoemmes): Potentially handle missing metadata and labels?
		return nil, errors.New("'metadata.labels' not found")
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
		return nil, fmt.Errorf("failed to encode YAML: %w", err)
	}
	return buffer.Bytes(), nil
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
