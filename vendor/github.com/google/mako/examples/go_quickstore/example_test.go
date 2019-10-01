// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// see the license for the specific language governing permissions and
// limitations under the license.
//
// An example of the use of Quickstore in a unit test, using Go Modules.
//
// This example assumes the microservice is already up and running on localhost,
// and that the port is specified via the MAKO_PORT environment variable.
//
// See the guide to using Quickstore at https://github.com/google/mako/blob/master/docs/GUIDE.md.
// In particular, before running this test set up authentication
// (https://github.com/google/mako/blob/master/docs/GUIDE.md#setting-up-authentication) and run the
// microservice (https://github.com/google/mako/blob/master/docs/GUIDE.md#quickstore-microservice).
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/google/mako/go/quickstore"

	tpb "github.com/google/mako/clients/proto/analyzers/threshold_analyzer_go_proto"
	qpb "github.com/google/mako/proto/quickstore/quickstore_go_proto"
	mpb "github.com/google/mako/spec/proto/mako_go_proto"
)

const (
	// This example assumes the microservice is already up and running on localhost,
	// and that the port is specified via the MAKO_PORT environment variable.
	benchmarkKey = "5251279936815104"
	resultsFile  = "performance_test_data.json"
)

var microservice string

func init() {
	p := os.Getenv("MAKO_PORT")
	if p == "" {
		panic("This test requires the MAKO_PORT env var set to the microservice's listening port. See https://github.com/google/mako/blob/master/docs/CONCEPTS.md#microservice.")
	}
	port, err := strconv.Atoi(p)
	if err != nil {
		panic(fmt.Sprintf("Could not parse MAKO_PORT env var: %v", err))
	}
	microservice = fmt.Sprintf("localhost:%d", port)
}

// TestPerformance reads performance_test_data.json, uploads it to
// https://mako.dev, and analyzes the data for performance regressions.
func TestPerformance(t *testing.T) {
	// STEP 1: Collect some performance data. Here we read some data from a
	// serialized format.
	//
	// Read more about the Mako data format at
	// https://github.com/google/mako/blob/master/docs/GUIDE.md#preparing-your-performance-test-data

	fmt.Printf("Reading performance test data...\n")
	data, err := readData()
	if err != nil {
		t.Fatalf("readData() = %s", err)
	}

	ctx := context.Background()
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// STEP 2: Configure run metadata in QuickstoreInput.
	//
	// Read about the run metadata you can set in QuickstoreInput at
	// https://github.com/google/mako/blob/master/docs/GUIDE.md#run-metadata.

	input := &qpb.QuickstoreInput{
		BenchmarkKey:   proto.String(benchmarkKey),
		DurationTimeMs: proto.Float64(float64(data.Metadata.RunDurationMs)),
		HoverText:      proto.String(data.Metadata.GitHash),
		Tags:           data.Metadata.Tags,
		TimestampMs:    proto.Float64(float64(data.Metadata.RunTimestamp)), // Mako will insert current time if left blank.
	}

	// STEP 3: Configure an analyzer.
	//
	// Read more about analyzers at https://github.com/google/mako/blob/master/docs/ANALYZERS.md.
	analysis := &tpb.ThresholdAnalyzerInput{
		Name: proto.String("Mako Go Quickstore Example Analyzer"),
		Configs: []*tpb.ThresholdConfig{
			{
				// Threshold on a metric aggregate (median of WriteLatency).
				ConfigName: proto.String("writes_lt_900"),
				Max:        proto.Float64(900),
				DataFilter: &mpb.DataFilter{
					DataType: mpb.DataFilter_METRIC_AGGREGATE_MEDIAN.Enum(),
					ValueKey: proto.String("wl"),
				},
			},
			{
				// Threshold on a custom aggregate (Throughput).
				ConfigName: proto.String("throughput_gt_4000"),
				Min:        proto.Float64(4000),
				DataFilter: &mpb.DataFilter{
					DataType: mpb.DataFilter_CUSTOM_AGGREGATE.Enum(),
					ValueKey: proto.String("tp"),
				},
			},
		},
	}
	input.ThresholdInputs = append(input.ThresholdInputs, analysis)

	// STEP 4: Instantiate a Mako client connecting to the running microservice.
	//
	// Read about setting up authentication and getting a Mako microservice
	// running at https://github.com/google/mako/blob/master/docs/GUIDE.md#setting-up-authentication
	// and https://github.com/google/mako/blob/master/docs/GUIDE.md#quickstore-microservice.

	// Establish a connection to the microservice running at `microservice`.
	fmt.Printf("Connecting to microservice at %s...\n", microservice)
	q, closeq, err := quickstore.NewAtAddress(ctxWithTimeout, input, microservice)
	if err != nil {
		t.Fatalf("quickstore.NewAtAddress() = %v", err)
	}
	// This tells the microservice to shut down. Don't call it if you want it to
	// stay up for subsequent Quickstore uses.
	defer closeq(ctx)

	// STEP 5: Feed your sample point data to the Mako Quickstore client.

	for _, sample := range data.Samples {
		values := make(map[string]float64)
		if sample.WriteLatency != nil {
			values["wl"] = float64(*sample.WriteLatency)
		}
		if sample.ReadLatency != nil {
			values["rl"] = float64(*sample.ReadLatency)
		}
		values["cpu"] = sample.CPULoad
		q.AddSamplePoint(float64(sample.Timestamp), values)
	}

	// STEP 6: Feed your custom aggregate data to the Mako Quickstore client.

	q.AddRunAggregate("tp", data.Counters.Throughput)
	q.AddRunAggregate("bm", data.Counters.BranchMisses)
	q.AddRunAggregate("pf", float64(data.Counters.PageFaults))

	// STEP 7: Call Store() to instruct Mako to process the data and upload it to
	// https://mako.dev.
	out, err := q.Store()
	if err != nil {
		t.Fatalf("quickstore Store() = %s", err)
	}
	switch out.GetStatus() {
	case qpb.QuickstoreOutput_SUCCESS:
		t.Logf("Done! Run can be found at: %s\n", out.GetRunChartLink())
	case qpb.QuickstoreOutput_ERROR:
		t.Fatalf("quickstore Store() output error: %s\n", out.GetSummaryOutput())
	case qpb.QuickstoreOutput_ANALYSIS_FAIL:
		t.Fatalf("Quickstore analysis failure: %s\nRun can be found at: %s\n", out.GetSummaryOutput(), out.GetRunChartLink())
	}
}

// perfData holds all the data from a performance test.
type perfData struct {
	Metadata metadata `json:"metadata"`
	Counters counter  `json:"counters"`
	Samples  []sample `json:"samples"`
}

// metadata holds the 'run metadata' from a performance test.
type metadata struct {
	GitHash       string   `json:"git_hash"`
	RunTimestamp  int64    `json:"run_timestamp"`
	RunDurationMs int64    `json:"run_duration_ms"`
	Tags          []string `json:"tags"`
}

// counter holds the 'custom aggregate' data from a performance test.
type counter struct {
	Throughput   float64 `json:"throughput"`
	BranchMisses float64 `json:"branch_miss_percentage"`
	PageFaults   int     `json:"page_faults"`
}

// sample holds a single sample point from a performance test.
type sample struct {
	Timestamp    int64   `json:"timestamp"`
	WriteLatency *int    `json:"write_latency"`
	ReadLatency  *int    `json:"read_latency"`
	CPULoad      float64 `json:"cpu_load"`
}

func readData() (*perfData, error) {
	// 'go test' puts the CWD in the package dir.
	f, err := os.Open(resultsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s with error: %s", resultsFile, err)
	}
	defer f.Close()

	perfData := &perfData{}
	data, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s with error: %s", resultsFile, err)
	}
	if err := json.Unmarshal(data, &perfData); err != nil {
		return nil, fmt.Errorf("failed to parse json data error: %s", err)
	}
	return perfData, nil
}
