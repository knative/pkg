/*
Copyright 2018 The Kubernetes Authors.

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

// Package junit provides a junit viewer for Spyglass
package junit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"path/filepath"
	"sort"
	"time"

	"github.com/sirupsen/logrus"

	"k8s.io/test-infra/prow/spyglass/lenses"
	"k8s.io/test-infra/testgrid/metadata/junit"
)

const (
	name     = "junit"
	title    = "JUnit"
	priority = 5
)

func init() {
	lenses.RegisterLens(Lens{})
}

// Lens is the implementation of a JUnit-rendering Spyglass lens.
type Lens struct{}

// Config returns the lens's configuration.
func (lens Lens) Config() lenses.LensConfig {
	return lenses.LensConfig{
		Name:     name,
		Title:    title,
		Priority: priority,
	}
}

// Header renders the content of <head> from template.html.
func (lens Lens) Header(artifacts []lenses.Artifact, resourceDir string, config json.RawMessage) string {
	t, err := template.ParseFiles(filepath.Join(resourceDir, "template.html"))
	if err != nil {
		return fmt.Sprintf("<!-- FAILED LOADING HEADER: %v -->", err)
	}
	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, "header", nil); err != nil {
		return fmt.Sprintf("<!-- FAILED EXECUTING HEADER TEMPLATE: %v -->", err)
	}
	return buf.String()
}

// Callback does nothing.
func (lens Lens) Callback(artifacts []lenses.Artifact, resourceDir string, data string, config json.RawMessage) string {
	return ""
}

type JunitResult struct {
	junit.Result
}

func (jr JunitResult) Duration() time.Duration {
	return time.Duration(jr.Time * float64(time.Second)).Round(time.Second)
}

// TestResult holds data about a test extracted from junit output
type TestResult struct {
	Junit JunitResult
	Link  string
}

// Body renders the <body> for JUnit tests
func (lens Lens) Body(artifacts []lenses.Artifact, resourceDir string, data string, config json.RawMessage) string {
	type testResults struct {
		junit []junit.Result
		link  string
		path  string
		err   error
	}
	resultChan := make(chan testResults)
	for _, artifact := range artifacts {
		go func(artifact lenses.Artifact) {
			result := testResults{
				link: artifact.CanonicalLink(),
				path: artifact.JobPath(),
			}
			var contents []byte
			contents, result.err = artifact.ReadAll()
			if result.err != nil {
				logrus.WithError(result.err).WithField("artifact", artifact.CanonicalLink()).Warn("Error reading artifact")
				resultChan <- result
				return
			}
			var suites junit.Suites
			suites, result.err = junit.Parse(contents)
			if result.err != nil {
				logrus.WithError(result.err).WithField("artifact", artifact.CanonicalLink()).Info("Error parsing junit file.")
				resultChan <- result
				return
			}
			var record func(suite junit.Suite)
			record = func(suite junit.Suite) {
				for _, subSuite := range suite.Suites {
					record(subSuite)
				}
				for _, test := range suite.Results {
					result.junit = append(result.junit, test)
				}
			}
			for _, suite := range suites.Suites {
				record(suite)
			}
			resultChan <- result
		}(artifact)
	}
	results := make([]testResults, 0, len(artifacts))
	for range artifacts {
		results = append(results, <-resultChan)
	}
	sort.Slice(results, func(i, j int) bool { return results[i].path < results[j].path })

	jvd := struct {
		NumTests int
		Passed   []TestResult
		Failed   []TestResult
		Skipped  []TestResult
	}{}
	for _, result := range results {
		if result.err != nil {
			continue
		}
		for _, test := range result.junit {
			if test.Failure != nil {
				jvd.Failed = append(jvd.Failed, TestResult{
					Junit: JunitResult{test},
					Link:  result.link,
				})
			} else if test.Skipped != nil {
				jvd.Skipped = append(jvd.Skipped, TestResult{
					Junit: JunitResult{test},
					Link:  result.link,
				})
			} else {
				jvd.Passed = append(jvd.Passed, TestResult{
					Junit: JunitResult{test},
					Link:  result.link,
				})
			}
		}
	}

	jvd.NumTests = len(jvd.Passed) + len(jvd.Failed) + len(jvd.Skipped)

	junitTemplate, err := template.ParseFiles(filepath.Join(resourceDir, "template.html"))
	if err != nil {
		logrus.WithError(err).Error("Error executing template.")
		return fmt.Sprintf("Failed to load template file: %v", err)
	}

	var buf bytes.Buffer
	if err := junitTemplate.ExecuteTemplate(&buf, "body", jvd); err != nil {
		logrus.WithError(err).Error("Error executing template.")
	}

	return buf.String()
}
