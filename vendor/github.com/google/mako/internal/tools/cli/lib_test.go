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
package lib

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
	"testing"
	"unicode"

	"flag"
	"github.com/bazelbuild/rules_go/go/tools/bazel"
	"github.com/google/subcommands"

	"github.com/golang/protobuf/proto"

	pgpb "github.com/google/mako/spec/proto/mako_go_proto"

	fakeStorage "github.com/google/mako/clients/go/storage/fakestorage"
)

var ctx context.Context
var fs = fakeStorage.New()

func TestParseProtoPath_NoArgs(t *testing.T) {
	if _, err := parseProtoPathFromCommandLineArgs([]string{}); err == nil {
		t.Error("parseProtoPathFromCommandLineArgs() nil; want err")
	}
}

func TestParseProtoPath_SingleArg(t *testing.T) {
	if _, err := parseProtoPathFromCommandLineArgs([]string{"one"}); err == nil {
		t.Error("parseProtoPathFromCommandLineArgs() nil; want err")
	}
}

func TestParseProtoPath_CorrectArg(t *testing.T) {
	protoPath := "/some/path"
	path, err := parseProtoPathFromCommandLineArgs([]string{"one", protoPath})

	if err != nil {
		t.Errorf("parseProtoPathFromCommandLineArgs() %s; want nil", err.Error())
	} else if path != protoPath {
		t.Errorf("parseProtoPathFromCommandLineArgs() %s; want %s", path, protoPath)
	}
}

func TestReadPathAsString_NoSuchFile(t *testing.T) {
	if _, err := readPathAsString("/tmp/no/such/file"); err == nil {
		t.Error("readPathAsString() nil; want err")
	}
}

func TestReadPathAsString_FoundFile(t *testing.T) {
	expectedFileContents := "yada\nyada\nyada"
	f, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatalf("Failed creating temp file: %v", err)
	}
	defer os.Remove(f.Name())
	if _, err := f.WriteString(expectedFileContents); err != nil {
		t.Fatalf("Failed writing content to file: %v", err)
	}
	if actualFileContents, err := readPathAsString(f.Name()); err != nil {
		t.Errorf("readPathAsString() %s; want nil", err.Error())
	} else if actualFileContents != expectedFileContents {
		t.Errorf("readPathAsString() %s; want %s", actualFileContents, expectedFileContents)
	}
}

func TestUnmarshal_Success(t *testing.T) {
	expectedBi := &pgpb.BenchmarkInfo{BenchmarkKey: proto.String("my_key"), ProjectName: proto.String("Project")}
	actualBi := &pgpb.BenchmarkInfo{}

	if err := unmarshal(actualBi, proto.MarshalTextString(expectedBi)); err != nil {
		t.Errorf("unmarshal() %s; want nil", err.Error())
	} else if !proto.Equal(expectedBi, actualBi) {
		t.Errorf("unmarshal() %s; want %s", actualBi, expectedBi)
	}
}

func TestParsingSampleProto(t *testing.T) {
	samplePath := "examples/benchmark_info/create_benchmark.config"

	textProto, err := readRunFilesFile(samplePath)
	if err != nil {
		t.Fatal("could not find sample at path:" + samplePath)
	}

	bi := &pgpb.BenchmarkInfo{}

	if err := unmarshal(bi, string(textProto)); err != nil {
		t.Errorf("unmarshal() %s; want nil", err.Error())
	}

	// Verify a few things about the benchmarkInfo to make sure it has been
	// parsed correctly.
	if len(bi.GetOwnerList()) == 0 {
		t.Errorf("GetOwnerList() %d; want > 0", len(bi.GetOwnerList()))
	} else if len(bi.GetDescription()) == 0 {
		t.Errorf("GetDescription() %d; want > 0", len(bi.GetDescription()))
	}
}

func TestUpdateBenchmarkFailsWithoutBenchmarkKeySubargOrBenchmarkInfoProtoPath(t *testing.T) {
	status, err := updateBenchmark(ctx, fs)
	if err == nil {
		t.Error("updateBenchmark(ctx, fs) nil; want err")
	}
	if status != subcommands.ExitUsageError {
		t.Errorf("updateBenchmark(ctx, fs) nil; got %s, want status of subcommands.ExitUsageError", status)
	}
}

func TestUpdateRunFailWithoutRunKeySubarg(t *testing.T) {
	status, err := updateRun(ctx, fs)
	if err == nil {
		t.Error("updateRun(ctx, fs) nil; want err")
	}
	if status != subcommands.ExitUsageError {
		t.Errorf("updateRun(ctx, fs) nil; got %s, want status of subcommands.ExitUsageError", status)
	}
}

func TestCreateTempTextProtoFileCreatesCorrectDir(t *testing.T) {
	fpath, err := createTempTextProtoFile()
	defer os.Remove(fpath)

	if err != nil {
		panic(err)
	}

	expect := path.Join(os.TempDir(), makoTempDirName)
	if !strings.HasPrefix(fpath, expect) {
		t.Error("expected prefix of " + expect + " got " + fpath)
	}

	if _, err = os.Stat(expect); err != nil && os.IsNotExist(err) {
		t.Error("expected " + expect + " to be created but not created")
	}
}

func TestCreateTempFileWithExtensionHasCorrectExtension(t *testing.T) {
	fpath, err := createTempTextProtoFile()
	if err != nil {
		panic(err)
	}

	if !strings.HasSuffix(fpath, "."+makoTempProtoFileExtension) {
		t.Error(fpath + " should have suffix of ." + makoTempProtoFileExtension)
	}
}

func TestDefaultEditorReturnsViForEmptyEditorVar(t *testing.T) {
	os.Setenv("EDITOR", "")
	if editor := defaultEditor(); editor != "vi" {
		t.Error("defaultEditor() " + editor + "; want vi")
	}
}

func TestDefaultEditorReturnsCorrectEditor(t *testing.T) {
	de := "emacs"
	os.Setenv("EDITOR", de)
	if editor := defaultEditor(); editor != de {
		t.Error("defaultEditor() " + editor + "; want " + de)
	}
}

// remove whitespace from string
func removeWhitespace(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) && r != '\n' {
			return -1
		}
		return r
	}, s)
}

// compare two strings, returning true if they are equivalent ignoring
// all their whitespace (and false otherwise)
func equalIgnoreWhitespace(s1, s2 string) bool {
	return removeWhitespace(s1) == removeWhitespace(s2)
}

// helper method takes in a mako command line string, runs the string as input
// to the CLI, and then compares the CLI output to the given expected output, erroring
// out if they are different (ignoring whitespace)
func checkCmdLine(ctx context.Context, t *testing.T, cmdLine []string, expectedOutput string,
	expectedExitStatus subcommands.ExitStatus) {
	var buf bytes.Buffer
	t.Logf("Running command %s", cmdLine)

	testCommandLine := flag.NewFlagSet(cmdLine[0], flag.PanicOnError)

	// Define the global flag -nointeractive on this FlagSet in order to properly test that it will work
	// when the binary is run.
	testCommandLine.BoolVar(flagNoInteractive, "nointeractive", false, "Skip all user prompts, assuming user entered \"yes\"")

	testCommandLine.Parse(cmdLine[1:])
	flag.CommandLine = testCommandLine

	writer := io.Writer(&buf)

	commander := subcommands.NewCommander(testCommandLine, path.Base(cmdLine[0]))
	exitStatus := Run(ctx, fs, writer, writer, commander)
	if exitStatus != expectedExitStatus {
		t.Errorf("mako test CLI call returned status %d, expected %d. output: %s",
			int(exitStatus), int(expectedExitStatus), buf.String())
	} else if !equalIgnoreWhitespace(expectedOutput, buf.String()) {
		t.Errorf("expected %q, got %q", expectedOutput, buf.String())
	}
}

func checkCmdLineSucceeded(ctx context.Context, t *testing.T, cmdLine []string, expectedOutput string) {
	checkCmdLine(ctx, t, cmdLine, expectedOutput, subcommands.ExitSuccess)
}

func TestInvalidRunQuery(t *testing.T) {
	checkCmdLine(ctx, t, []string{"mako", "list_runs", "-benchmark_key", "testmark",
		"-run_build_id_min", "5", "-run_build_id_max", "3"},
		"***\n-run_build_id_min 5 must be less than -run_build_id_max 3\n***\n",
		subcommands.ExitFailure)

	checkCmdLine(ctx, t, []string{"mako", "list_runs", "-benchmark_key", "testmark",
		"-run_timestamp_min_ms", "10.0", "-run_timestamp_max_ms", "3.6"},
		"***\n-run_timestamp_min_ms 10.000000 must not be greater than -run_timestamp_max_ms 3.600000\n***\n",
		subcommands.ExitFailure)

	checkCmdLine(ctx, t, []string{"mako", "list_runs", "-benchmark_key", "testmark",
		"-run_timestamp_min_ms", "10.0", "-run_build_id_min", "5"},
		"***\ncannot filter runs based on both timestamp and build id currently\n***\n",
		subcommands.ExitFailure)

	checkCmdLine(ctx, t, []string{"mako", "add_tag", "-tag", "a", "-tag_list", "a,", "-benchmark_key", "testmark"},
		"***\n-tag_list can not have empty tag string\n***\n",
		subcommands.ExitFailure)
}

func TestHelp(t *testing.T) {
	defer fs.FakeClear()
	checkCmdLineSucceeded(ctx, t, []string{"mako", "help"},
		`Usage: mako <flags> <subcommand> <subcommand args>

					Subcommands:
						flags            describe all known top-level flags
						help             describe subcommands and their syntax

					Subcommands for annotations:
						add_annotation   Add an annotation to a run.
						delete_annotation  Delete an annotation.
						list_annotations  Lists the annotations for a run.

					Subcommands for benchmarks:
						create_benchmark  Create a new benchmark.
						delete_benchmark  Deletes a new benchmark.
						display_benchmark  Displays a benchmark.
						list_benchmarks  Lists matching benchmark keys.
						update_benchmark  Update an existing benchmark.

					Subcommands for runs:
						delete_runs      Delete one or more runs and associated sample batch data.
						display_run      Displays a run.
						list_runs        Lists matching run keys.
						update_run       Update single run

					Subcommands for sample batches:
						display_sample_batch  Display a sample batch.
						list_sample_batches  List all sample batches from a run.

					Subcommands for tags:
						add_tag          Add a tag to the specified runs.
						delete_tag       Delete the specified tag.
						list_tags        List all tags from the specified run.


					Top-level flags (use "mako flags" for a full list):
						-nointeractive=false: Skip all user prompts, assuming user entered "yes"
	`)
}

// helper function that writes out a given benchmark info text proto buffer to a
// test file, returning the filename it wrote to
func writeBenchmarkTextProto(benchmarkTxt string) (benchFilePath string) {
	benchFilePath = path.Join(os.Getenv("TEST_TMPDIR"), "test_benchmark.textpb")
	f, err := os.Create(benchFilePath)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	_, err = f.WriteString(benchmarkTxt)
	if err != nil {
		panic(err)
	}
	return
}

func TestCreateListUpdateDeleteBenchmark(t *testing.T) {
	defer fs.FakeClear()
	benchFilePath := writeBenchmarkTextProto(`
benchmark_name: "testmark"
project_name: "testproj"

# allow all users to write to this test benchmark
owner_list: "*"
# owner_list: "<MDB_GROUP>@prod.google.com"

input_value_info: <
  value_key: "t"
  label: "time"
  type: TIMESTAMP
>

metric_info_list: <
  value_key: "tm1"
  label: "TestMetric"
>
metric_info_list: <
  value_key: "tm2"
  label: "OtherTestMetric"
>
`)
	checkCmdLineSucceeded(ctx, t, []string{"mako", "create_benchmark", benchFilePath},
		`Benchmark creation successful. Please add the benchmark_key: '1' to your benchmark text-proto file.
	`)
	checkCmdLineSucceeded(ctx, t, []string{"mako", "list_benchmarks", "-project_name=testproj"}, "1\n")
	checkCmdLineSucceeded(ctx, t, []string{"mako", "list_benchmarks", "-benchmark_name=testmark"}, "1\n")
	// add another benchmark to see how things work with multiple benchmarks in system
	benchFilePath2 := writeBenchmarkTextProto(`
benchmark_name: "testmark2"
project_name: "testproj"

# allow all users to write to this test benchmark
owner_list: "*"
# owner_list: "<MDB_GROUP>@prod.google.com"

input_value_info: <
  value_key: "t"
  label: "time"
  type: TIMESTAMP
>

metric_info_list: <
  value_key: "tm1"
  label: "TestMetric"
>
metric_info_list: <
  value_key: "tm2"
  label: "OtherTestMetric"
>
`)
	checkCmdLineSucceeded(ctx, t, []string{"mako", "create_benchmark", benchFilePath2},
		`Benchmark creation successful. Please add the benchmark_key: '2' to your benchmark text-proto file.
	`)
	checkCmdLineSucceeded(ctx, t, []string{"mako", "list_benchmarks", "-project_name=testproj"}, "1\n2\n")
	checkCmdLineSucceeded(ctx, t, []string{"mako", "list_benchmarks", "-benchmark_name=testmark2"}, "2\n")
	checkCmdLineSucceeded(ctx, t, []string{"mako", "display_benchmark", "-benchmark_key=1"},
		`benchmark_key: "1"
		benchmark_name: "testmark"
		project_name: "testproj"
		owner_list: "*"
		input_value_info: <
			value_key: "t"
			label: "time"
			type: TIMESTAMP
		>
		metric_info_list: <
			value_key: "tm1"
			label: "TestMetric"
		>
		metric_info_list: <
			value_key: "tm2"
			label: "OtherTestMetric"
		>
	`)
	// test querying for a nonexistant benchmark
	checkCmdLineSucceeded(ctx, t, []string{"mako", "list_benchmarks", "-project_name=non_existent"},
		"No results! Check your arguments.\n")
	checkCmdLineSucceeded(ctx, t, []string{"mako", "list_benchmarks", "-benchmark_name=non_existent"},
		"No results! Check your arguments.\n")
	// test update_benchmark
	benchFilePath = writeBenchmarkTextProto(`
benchmark_key: "1"
benchmark_name: "testmarkv2"
project_name: "testproj"

# allow all users to write to this test benchmark
owner_list: "*"
# owner_list: "<MDB_GROUP>@prod.google.com"

input_value_info: <
  value_key: "t"
  label: "time"
  type: TIMESTAMP
>

metric_info_list: <
  value_key: "tm1"
  label: "TestMetric"
>
metric_info_list: <
  value_key: "tm2"
  label: "OtherTestMetric"
>
`)
	checkCmdLineSucceeded(ctx, t, []string{"mako", "update_benchmark", benchFilePath},
		"Benchmark update successful.\n")
	// check to see if we can query based on the updated benchmark, and not on the old value
	checkCmdLineSucceeded(ctx, t, []string{"mako", "list_benchmarks", "-benchmark_name=testmarkv2"}, "1\n")
	checkCmdLineSucceeded(ctx, t, []string{"mako", "list_benchmarks", "-benchmark_name=testmark"},
		"No results! Check your arguments.\n")
	// remove benchmark
	checkCmdLineSucceeded(ctx, t, []string{"mako", "delete_benchmark", "-benchmark_key=1"},
		"Deletion of benchmark successful.\n")
	// make sure it is gone
	checkCmdLineSucceeded(ctx, t, []string{"mako", "list_benchmarks", "-project_name=testproj"},
		"2\n")
	checkCmdLineSucceeded(ctx, t, []string{"mako", "list_benchmarks", "-benchmark_name=testmark"},
		"No results! Check your arguments.\n")
}

func TestRunManipulation(t *testing.T) {
	defer fs.FakeClear()
	ctx := context.Background()
	// make some test runs to query, tag, and delete
	for i := 0; i < 10; i++ {
		run := &pgpb.RunInfo{
			BenchmarkKey: proto.String("testmark"),
			TimestampMs:  proto.Float64(float64(i)),
			BuildId:      proto.Int64(int64(i)),
			// integer divide by two so that each test pass id has 2 runs associated
			TestPassId: proto.String(strconv.Itoa(i / 2)),
		}
		if _, err := fs.CreateRunInfo(ctx, run); err != nil {
			t.Fatalf("fs.CreateRunInfo(ctx, %v) got err %v; want nil", run, err)
		}
	}
	// query runs
	checkCmdLineSucceeded(ctx, t, []string{"mako", "list_runs", "-benchmark_key=testmark"},
		"10\n9\n8\n7\n6\n5\n4\n3\n2\n1\n")
	checkCmdLineSucceeded(ctx, t, []string{"mako", "list_runs", "-benchmark_key=testmark",
		"-run_timestamp_min_ms=1.0", "-run_timestamp_max_ms=4.0"},
		"5\n4\n3\n2\n")
	checkCmdLineSucceeded(ctx, t, []string{"mako", "list_runs", "-benchmark_key=testmark",
		"-run_build_id_min=1", "-run_build_id_max=4"},
		"5\n4\n3\n2\n")
	checkCmdLineSucceeded(ctx, t, []string{"mako", "list_runs", "-benchmark_key=testmark",
		"-test_pass_id=0"}, "2\n1\n")
	checkCmdLineSucceeded(ctx, t, []string{"mako", "list_runs", "-benchmark_key=testmark",
		"-test_pass_id=2"}, "6\n5\n")

	// annotate runs
	checkCmdLineSucceeded(ctx, t, []string{"mako", "add_annotation", "-run_key=1",
		"-label=a", "-value_key=tm1", "-description=testing"}, "Successfully added a new annotation.\n")
	checkCmdLineSucceeded(ctx, t, []string{"mako", "list_annotations", "-run_key=1"}, `
		Annotations List:#0  value_key: "tm1"
    label: "a"
    description: "testing"


    The index listed before each annotation can be used as input to the 'delete_annotation' command.`)
	checkCmdLineSucceeded(ctx, t, []string{"mako", "delete_annotation", "-run_key=1", "-annotation_index=0"},
		`Successfully deleted annotation: value_key: "tm1"
        label: "a"
        description: "testing"

	`)
	checkCmdLineSucceeded(ctx, t, []string{"mako", "list_annotations", "-run_key=1"},
		"Run has no annotations. Add some with the 'add_annotation' command\n")

	// delete all runs using nointeractive
	checkCmdLineSucceeded(ctx, t, []string{"mako", "-nointeractive=true", "delete_runs", "-benchmark_key=testmark"},
		"Done. Deleted 10 runs.\n")
}

func TestUpdateManyRuns(t *testing.T) {
	defer fs.FakeClear()
	ctx := context.Background()
	subargBenchmarkKey = "123456"
	subargTag = "fake_tag"
	numRuns := 5000
	*flagNoInteractive = true
	subargRunKey = ""
	subargTimestampMinMs = -1
	subargTimestampMaxMs = -1
	subargMinBuildID = -1
	subargMaxBuildID = -1
	subargTestPassID = ""
	defer func() {
		*flagNoInteractive = false
		subargTimestampMinMs = -1
		subargTimestampMaxMs = -1
		subargMinBuildID = -1
		subargMaxBuildID = -1
		subargRunKey = ""
	}()
	for i := 0; i < numRuns; i++ {
		run := &pgpb.RunInfo{BenchmarkKey: proto.String(subargBenchmarkKey), TimestampMs: proto.Float64(float64(i))}
		if _, err := fs.CreateRunInfo(ctx, run); err != nil {
			t.Fatalf("fs.CreateRunInfo(ctx, %v) got err %v; want nil", run, err)
		}
	}
	status, err := addTag(ctx, fs)
	if err != nil {
		t.Fatalf("addTag(ctx, fs) got err %v; want nil", err)
	}
	if status != subcommands.ExitSuccess {
		t.Fatalf("addTag(ctx, fs) got status %v; want subcommands.ExitSuccess", status)
	}
	query := pgpb.RunInfoQuery{BenchmarkKey: &subargBenchmarkKey}
	runs, err := queryRunInfo(ctx, fs, query)
	if err != nil {
		t.Fatalf("queryRunInfo(ctx, fs, %v) got err %v; want nil", query, err)
	}
	if got := len(runs); got != numRuns {
		t.Errorf("queryRunInfo(ctx, fs, %v) got %d runs; want %d", query, got, numRuns)
	}
	for _, run := range runs {
		tags := run.GetTags()
		if got := len(tags); got != 1 {
			t.Errorf("Run %v has %d tags; want 1", run, got)
			continue
		}
		if got := tags[0]; got != subargTag {
			t.Errorf("Run %v has tag %q; want %q", run, got, subargTag)
		}
	}
}

func TestListSampleBatches(t *testing.T) {
	defer fs.FakeClear()
	ctx := context.Background()
	benchmarkKey := "12345"
	var nonEmptyRunKey string
	var emptyRunKey string

	for _, runParams := range []struct {
		runKey      *string
		addBatchKey bool
	}{
		{
			runKey:      &nonEmptyRunKey,
			addBatchKey: true,
		},
		{
			runKey: &emptyRunKey,
		},
	} {
		runInfo := &pgpb.RunInfo{
			BenchmarkKey: proto.String(benchmarkKey),
			TimestampMs:  proto.Float64(float64(0)),
		}

		if runParams.addBatchKey {
			runInfo.BatchKeyList = []string{"983457"}
		}
		if creation, err := fs.CreateRunInfo(ctx, runInfo); err != nil {
			t.Fatalf("fs.CreateRunInfo(ctx, run) got err %v; want nil", err)
		} else if creation.Status.GetCode() != pgpb.Status_SUCCESS {
			t.Fatalf("fs.CreateRunInfo(ctx, run) was unsuccessful, want success. Error info: %v", creation.Status.GetFailMessage())
		} else {
			*runParams.runKey = creation.GetKey()
		}
	}

	for _, test := range []struct {
		name                 string
		runKey               string
		expectedExitStatus   subcommands.ExitStatus
		expectedErrSubstring string
	}{
		{
			name:                 "Running listSampleBatches with an empty run key",
			expectedExitStatus:   subcommands.ExitUsageError,
			expectedErrSubstring: "run_key",
		},
		{
			name:                 "Running listSampleBatches for an invalid run",
			runKey:               "483024",
			expectedExitStatus:   subcommands.ExitFailure,
			expectedErrSubstring: "want 1",
		},
		{
			name:                 "Running listSampleBatches for a run with no sample batches",
			runKey:               emptyRunKey,
			expectedExitStatus:   subcommands.ExitFailure,
			expectedErrSubstring: "no sample batches",
		},
		{
			name:               "Running listSampleBatches for a run with sample batches",
			runKey:             nonEmptyRunKey,
			expectedExitStatus: subcommands.ExitSuccess,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			subargRunKey = test.runKey
			if status, err := listSampleBatches(ctx, fs); status != test.expectedExitStatus {
				t.Errorf("listSampleBatches(ctx, fs) got status %v, want %v", status, test.expectedExitStatus)
			} else if err == nil && len(test.expectedErrSubstring) > 0 {
				t.Errorf("listSampleBatches(ctx, fs) got nil error, want error with substring %v", test.expectedErrSubstring)
			} else if err != nil && len(test.expectedErrSubstring) == 0 {
				t.Errorf("listSampleBatches(ctx, fs) got error %v, want nil", err)
			} else if err != nil && len(test.expectedErrSubstring) > 0 && !strings.Contains(err.Error(), test.expectedErrSubstring) {
				t.Errorf("listSampleBatches(ctx, fs) got error %v, want error with substring %v", err, test.expectedErrSubstring)
			}
		})
	}
}

func TestDisplaySampleBatch(t *testing.T) {
	defer fs.FakeClear()
	ctx := context.Background()
	var batchKey string

	if creation, err := fs.CreateSampleBatch(ctx, &pgpb.SampleBatch{
		BenchmarkKey: proto.String("12345"),
		RunKey:       proto.String("6789"),
		SampleErrorList: []*pgpb.SampleError{
			{
				InputValue:   proto.Float64(float64(859)),
				SamplerName:  proto.String("sampler"),
				ErrorMessage: proto.String("err"),
			}},
	}); err != nil {
		t.Fatalf("fs.CreateSampleBatch(ctx, sampleBatch) got err %v, want nil", err)
	} else if creation.Status.GetCode() != pgpb.Status_SUCCESS {
		t.Fatalf("fs.CreateSampleBatch(ctx, sampleBatch) was unsuccessful, want success. Error info: %v", creation.Status.GetFailMessage())
	} else {
		batchKey = creation.GetKey()
	}

	for _, test := range []struct {
		name                 string
		batchKey             string
		expectedExitStatus   subcommands.ExitStatus
		expectedErrSubstring string
	}{
		{
			name:                 "running displaySampleBatch for an empty batch_key",
			expectedExitStatus:   subcommands.ExitUsageError,
			expectedErrSubstring: "batch_key",
		},
		{
			name:                 "Running displaySampleBatch for an invalid batch_key",
			batchKey:             "483024",
			expectedExitStatus:   subcommands.ExitFailure,
			expectedErrSubstring: "no results",
		},
		{
			name:               "Running displaySampleBatch for a valid batch_key",
			batchKey:           batchKey,
			expectedExitStatus: subcommands.ExitSuccess,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			subargBatchKey = test.batchKey
			if status, err := displaySampleBatch(ctx, fs); status != test.expectedExitStatus {
				t.Errorf("displaySampleBatch(ctx, fs) got status %v, want %v", status, test.expectedExitStatus)
			} else if err == nil && len(test.expectedErrSubstring) > 0 {
				t.Errorf("displaySampleBatch(ctx, fs) got nil error, want error with substring %v", test.expectedErrSubstring)
			} else if err != nil && len(test.expectedErrSubstring) == 0 {
				t.Errorf("displaySampleBatch(ctx, fs) got error %v, want nil", err)
			} else if err != nil && len(test.expectedErrSubstring) > 0 && !strings.Contains(err.Error(), test.expectedErrSubstring) {
				t.Errorf("displaySampleBatch(ctx, fs) got error %v, want error with substring %v", err, test.expectedErrSubstring)
			}
		})
	}
}

func readRunFilesFile(name string) ([]byte, error) {
	path, err := bazel.Runfile(name)
	if err != nil {
	  return nil, err
	}
	return ioutil.ReadFile(path)
}
