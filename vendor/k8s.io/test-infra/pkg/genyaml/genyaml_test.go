/*
Copyright 2019 The Kubernetes Authors.

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

package genyaml

import (
	"bytes"
	"encoding/json"
	yaml3 "gopkg.in/yaml.v3"
	"io/ioutil"
	aliases "k8s.io/test-infra/pkg/genyaml/testdata/alias_types"
	embedded "k8s.io/test-infra/pkg/genyaml/testdata/embedded_structs"
	interfaces "k8s.io/test-infra/pkg/genyaml/testdata/interface_types"
	multiline "k8s.io/test-infra/pkg/genyaml/testdata/multiline_comments"
	nested "k8s.io/test-infra/pkg/genyaml/testdata/nested_structs"
	tags "k8s.io/test-infra/pkg/genyaml/testdata/no_tags"
	omit "k8s.io/test-infra/pkg/genyaml/testdata/omit_if_empty"
	pointers "k8s.io/test-infra/pkg/genyaml/testdata/pointer_types"
	primitives "k8s.io/test-infra/pkg/genyaml/testdata/primitive_types"
	private "k8s.io/test-infra/pkg/genyaml/testdata/private_members"
	"path/filepath"
	"strings"
	"testing"
)

const (
	testDir = "testdata"
)

func resolvePath(t *testing.T, filename string) string {
	name := filepath.Base(t.Name())
	return strings.ToLower(filepath.Join(testDir, name, filename))
}

func readFile(t *testing.T, extension string) []byte {
	name := filepath.Base(t.Name())
	data, err := ioutil.ReadFile(strings.ToLower(filepath.Join(testDir, name, name+"."+extension)))
	if err != nil {
		t.Errorf("Failed reading .%s file: %v.", extension, err)
	}

	return data
}

func TestFmtRawDoc(t *testing.T) {
	tests := []struct {
		name     string
		rawDoc   string
		expected string
	}{
		{
			name:     "Single line comment",
			rawDoc:   "Owners of the cat.",
			expected: "Owners of the cat.",
		},
		{
			name:   "Multi line comment",
			rawDoc: "StringField comment\nsecond line\nthird line",
			expected: `StringField comment
second line
third line`,
		},
		{
			name:     "Delete trailing space(s)",
			rawDoc:   "Some comment    ",
			expected: "Some comment",
		},
		{
			name:     "Delete trailing newline(s)",
			rawDoc:   "Some comment\n\n\n\n",
			expected: "Some comment",
		},
		{
			name:     "Escape double quote(s)",
			rawDoc:   `"Some comment"`,
			expected: `"Some comment"`,
		},
		{
			name: "Convert tab to space",
			rawDoc: "tab	tab		tabtab",
			expected: "tab tab tabtab",
		},
		{
			name:     "Strip TODO prefixed comment",
			rawDoc:   "TODO: some future work",
			expected: "",
		},
		{
			name:     "Strip + prefixed comment",
			rawDoc:   "+: some future work",
			expected: "",
		},
		{
			name:     "Strip TODO prefixed comment from multi line comment",
			rawDoc:   "TODO: some future work\nmore comment",
			expected: "more comment",
		},
		{
			name:     "Strip + prefixed comment from multi line comment",
			rawDoc:   "+: some future work\nmore comment",
			expected: "more comment",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actualFormattedRawDoc := fmtRawDoc(test.rawDoc)

			if actualFormattedRawDoc != test.expected {
				t.Fatalf("Expected %q, but got result %q", test.expected, actualFormattedRawDoc)
			}
		})
	}
}

func TestInjectComment(t *testing.T) {
	tests := []struct {
		name         string
		typeSpec     string
		actualNode   *yaml3.Node
		expectedNode *yaml3.Node
	}{
		{
			name:     "Inject comments",
			typeSpec: "ExampleStruct",
			actualNode: &yaml3.Node{
				Kind: yaml3.DocumentNode,
				Content: []*yaml3.Node{
					{
						Kind: yaml3.MappingNode,
						Tag:  "!!map",
						Content: []*yaml3.Node{
							{
								Kind:  yaml3.ScalarNode,
								Tag:   "!!str",
								Value: "exampleKey",
							},
							{
								Kind:  yaml3.ScalarNode,
								Tag:   "!!bool",
								Value: "true",
							},
						},
					},
				},
			},
			expectedNode: &yaml3.Node{
				Kind: yaml3.DocumentNode,
				Content: []*yaml3.Node{
					{
						Kind: yaml3.MappingNode,
						Tag:  "!!map",
						Content: []*yaml3.Node{
							{
								Kind:        yaml3.ScalarNode,
								Tag:         "!!str",
								Value:       "exampleKey",
								HeadComment: "Some comment",
							},
							{
								Kind:  yaml3.ScalarNode,
								Tag:   "!!bool",
								Value: "true",
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cm := NewCommentMap()

			err := json.Unmarshal(readFile(t, "json"), &cm.comments)
			if err != nil {
				t.Errorf("Unexpected error unmarshalling JSON to comments: %v.", err)
			}

			cm.injectComment(test.actualNode, []string{test.typeSpec}, 0)

			expectedYaml, err := yaml3.Marshal(test.expectedNode)
			if err != nil {
				t.Errorf("Unexpected error marshalling Node to YAML: %v.", err)
			}

			actualYaml, err := yaml3.Marshal(test.actualNode)
			if err != nil {
				t.Errorf("Unexpected error marshalling Node to YAML: %v.", err)
			}

			if !bytes.Equal(expectedYaml, actualYaml) {
				t.Error("Expected yaml snippets to not be equal.")
			}
		})
	}
}

func TestAddPath(t *testing.T) {
	tests := []struct {
		name     string
		paths    []string
		expected bool
	}{
		{
			name:     "Single path",
			paths:    []string{"example_config.go"},
			expected: true,
		},
		{
			name:     "Multiple paths",
			paths:    []string{"example_config1.go", "example_config2.go"},
			expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cm := NewCommentMap()

			for _, f := range test.paths {
				cm.AddPath(resolvePath(t, f))
			}

			expectedComments := readFile(t, "json")
			actualComments, err := json.MarshalIndent(cm.comments, "", "  ")

			if err != nil {
				t.Errorf("Unexpected error generating JSON from comments: %v.", err)
			}

			equal := bytes.Equal(expectedComments, actualComments)

			if equal != test.expected {
				t.Errorf("Expected comments equality to be: %t.", test.expected)
			}
		})
	}

}

func TestSetPath(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(cm *CommentMap)
		path     string
		expected bool
	}{
		{
			name: "Single path",
			setup: func(cm *CommentMap) {
				cm.comments = make(map[string]map[string]Comment)
			},
			path:     "example_config.go",
			expected: true,
		},
		{
			name: "Set path overwrite",
			setup: func(cm *CommentMap) {
				cm.comments["dummy_key"] = make(map[string]Comment)
				cm.comments["dummy_key"]["dummy_sub_key"] = Comment{
					Type:  "string",
					IsObj: false,
					Doc:   "Some preloaded comments",
				}
			},
			path:     "example_config.go",
			expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cm := NewCommentMap()
			test.setup(cm)

			cm.SetPath(resolvePath(t, test.path))

			expectedComments := readFile(t, "json")
			actualComments, err := json.MarshalIndent(cm.comments, "", "  ")

			if err != nil {
				t.Errorf("Unexpected error generating JSON from comments: %v.", err)
			}

			equal := bytes.Equal(expectedComments, actualComments)

			if equal != test.expected {
				t.Errorf("Expected comments equality to be: %t.", test.expected)
			}
		})
	}

}

func TestGenYAML(t *testing.T) {
	tests := []struct {
		name      string
		structObj interface{}
		expected  bool
	}{
		{
			name: "alias types",
			structObj: &aliases.Alias{
				StringField: "string",
			},
			expected: true,
		},
		{
			name: "primitive types",
			structObj: &primitives.Primitives{
				StringField:  "string",
				BooleanField: true,
				IntegerField: 1,
			},
			expected: true,
		},
		{
			name: "multiline comments",
			structObj: &multiline.Multiline{
				StringField1: "string1",
				StringField2: "string2",
				StringField3: "string3",
				StringField4: "string4",
				StringField5: "string5",
				StringField6: "string6",
			},
			expected: true,
		},
		{
			name: "nested structs",
			structObj: &nested.Parent{
				Age: 35,
				Children: []nested.Child{
					{Name: "Jimbo", Age: 4},
					{Name: "Jenny", Age: 5},
				},
				Name: "Mildred",
			},
			expected: true,
		},
		{
			name: "embedded structs",
			structObj: &embedded.Building{
				Address:  "123 North Main Street",
				Bathroom: embedded.Bathroom{Width: 100, Height: 200},
				Bedroom:  embedded.Bedroom{Width: 100, Height: 200},
			},
			expected: true,
		},
		{
			name: "no tags",
			structObj: &tags.Tagless{
				StringField:  "string",
				BooleanField: true,
				IntegerField: 1,
			},
			expected: true,
		},
		{
			name: "omit if empty",
			structObj: &omit.OmitEmptyStrings{
				StringFieldOmitEmpty: "",
				StringFieldKeepEmpty: "",
				BooleanField:         true,
				IntegerField:         1,
			},
			expected: true,
		},
		{
			name: "pointer types",
			structObj: &pointers.Zoo{
				Employees: []*pointers.Employee{
					{
						Name: "Jim",
						Age:  22,
					},
					{
						Name: "Jane",
						Age:  21,
					},
				},
			},
			expected: true,
		},
		{
			name:      "private members",
			structObj: private.NewPerson("gamer123", "password123"),
			expected:  true,
		},
		{
			name: "interface types",
			structObj: &interfaces.Zoo{
				Animals: []interfaces.Animal{
					&interfaces.Lion{
						Name: "Leo",
					},
					&interfaces.Cheetah{
						Name: "Charles",
					},
				},
			},
			// INFO: Interface type comments are not implemented.
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cm := NewCommentMap(resolvePath(t, "example_config.go"))
			expectedYaml := readFile(t, "yaml")

			actualYaml, err := cm.GenYaml(test.structObj)

			if err != nil {
				t.Errorf("Unexpected error generating YAML from struct: %v.", err)
			}

			equal := bytes.Equal(expectedYaml, []byte(actualYaml))

			if equal != test.expected {
				t.Errorf("Expected yaml snippets equality to be: %t.", test.expected)
			}
		})
	}
}
