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

/*
The mako-g3 command gives google3 users of the mako framework
(go/mako for more information) the ability to accomplish tasks via the
command line.

Execute with no params to see usage message. Providing the incorrect arguments
to a command will result in the command usage being displayed.

All queries are done by primary key which ensures strong consistency.

Due to retry logic both inside mako clients and on the server
invalid queries (eg. for a non-existent key) can take a while. If you'd like to
see the output from the query library add these flags to your query before the
command. Example:
  $ mako -vmodule=google3storage=2 --alsologtostderr <command> ...


*/
package lib

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"

	log "github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	pgpb "github.com/google/mako/spec/proto/mako_go_proto"
	"github.com/google/subcommands"

	"github.com/google/mako/spec/go/mako"
)

var (
	flagNoInteractive = flag.Bool("nointeractive", false, "Skip all user prompts, assuming user entered \"yes\"")

	// All the possible subcommand args
	subargBenchmarkKey    string
	subargBenchmarkName   string
	subargOwner           string
	subargProjectName     string
	subargRunKey          string
	subargBatchKey        string
	subargTimestampMinMs  float64
	subargTimestampMaxMs  float64
	subargMinBuildID      int64
	subargMaxBuildID      int64
	subargTestPassID      string
	subargTag             string
	subargTagList         string
	parsedTags            []string
	subargAnnotationIndex int
	subargValueKey        string
	subargLabel           string
	subargDesc            string
	subargDisplayRuns     bool

	// places where all output will go - are vars so that we can switch it out for testing purposes
	outWriter io.Writer = os.Stdout
	errWriter io.Writer = os.Stderr

	// storage client instance to use for all commands
	storageClient mako.Storage
)

const (
	listBenchmarkUsage = `
  list_benchmarks [-project_name=<projectName>] [-benchmark_name=<benchmarkName>] [-owner=<owner>]

  Lists all benchmarks matching the given parameters.

	<owner> must be a full and exact string match for an owner in the Benchmark's BenchmarkInfo.owner_list.
	Most owners look like user@google.com or mdb_group@prod.google.com.
	Cloud service accounts might look something like service-account@project-name.iam.gserviceaccount.com.

  NOTE: At least one of -project_name, -benchmark_name, or -owner is required.

`

	displayBenchmarkUsage = `
  display_benchmark -benchmark_key=<key>

  Displays all data for the provided benchmark.

  NOTE: -benchmark_key is required.

`

	createBenchmarkUsage = `
  create_benchmark [<path>]

  This command creates a new benchmark with the information specified in the
  benchmark text-proto file.

  Leaving the path blank will load a template BenchmarkInfo text proto
  (see createBenchmarkTemplate below). Writing to that proto and exiting
  the editor will push the new BenchmarkInfo to be created on the Mako server.

  The text-proto file at the specified path should be in the same form as
  https://github.com/google/mako/blob/master/examples/benchmark_info/create_benchmark.config.

  NOTE: Because you are creating a new benchmark the benchmark_key must be
  empty.

  Also note that the owner_list must be populated. Probably a good idea to
  set as you or "*" to allow world access. See config file above for more information.

  NOTE: More information about representing protobuffs in text format can be found
  here: sites/protocol-buffers/user-docs/miscellaneous-howtos/text-format-examples

`

	updateBenchmarkUsage = `
  update_benchmark (<path> | -benchmark_key=<key>)

  This command updates the specified benchmark with any changes made to the
  benchmark text-proto file.

  If -benchmark_key is set then the corresponding BenchmarkInfo will be queried
  and presented in an editor for edits to be made directly through the CLI.

  The text-proto file at the specified path should be in the same form as
  https://github.com/google/mako/blob/master/examples/benchmark_info/update_benchmark.config.

  NOTE: Because you are updating an existing benchmark the benchmark_key must
  be valid as it is used to identify which benchmark record to make changes
  to.

  Also note that the caller must be in the owner_list of the benchmark.

  NOTE: More information about representing protobuffs in text format can be found
	here: go/proto-text-format

`

	deleteBenchmarkUsage = `
  delete_benchmark -benchmark_key=<key>

  Deletes the benchmark specified by the benchmark_key passed on command line.

  NOTE: -benchmark_key is required.

  NOTE: All child data (eg. run_info and sample batches) need to be removed
  before deleting a benchmark. Use the delete_runs subcommand to do this.

`

	listAnnotationsUsage = `
  list_annotations -run_key=<key>

  Lists all annotations for the run specified with run_key.

  The index printed for each annotation can be used as input to delete_annotation.

  NOTE: -run_key is required.

`

	addAnnotationUsage = `
  add_annotation -run_key=<key> -label=<label> -value_key=<key> -description=<description>

  Add an annotation to the run specified with run_key.

  NOTE: All flag fields are required. See RunAnnotation proto for documentation on the flags/fields
  https://github.com/google/mako/blob/master/spec/proto/mako.proto

`

	deleteAnnotationUsage = `
  delete_annotation -run_key=<key> -annotation_index=<index>

  Delete an annotation from the run specified with run_key.

  The annotation_index is the index that is output from list_annotations.

`

	listRunsUsage = `
  list_runs -benchmark_key=<key> [-run_timestamp_min_ms=<minTimestampMs>] [-run_timestamp_max_ms=<maxTimestampMs>] [-run_build_id_min=<minBuildID>] [-run_build_id_max=<maxBuildID>] [-test_pass_id=<testPassID>] [-tag_list=<tags>]

  Lists all runs matching the input parameters.

  NOTE: -benchmark_key is required.

`

	displayRunUsage = `
  display_run -run_key=<key>

  Displays all data for the provided run.

  NOTE: -run_key is required.

`

	deleteRunsUsage = `
  delete_runs -benchmark_key=<key> [-run_key=<key>] [-run_timestamp_min_ms=<minTimestampMs>] [-run_timestamp_max_ms=<maxTimestampMs>] [-run_build_id_min=<minBuildID>] [-run_build_id_max=<maxBuildID>] [-test_pass_id=<testPassID>] [-tag_list=<tags>]

  Delete one or more runs and associated sample batch data with supplied filters.

  NOTE: While the deletion is running, affected run charts may fail to load
  if the run still exists yet the batch data has been removed. Once processing
  completes, this is not an issue.

  Required flags:
    -benchmark_key

  Flags for filtering runs to be deleted:
    -run_key
    -run_timestamp_min_ms
    -run_timestamp_max_ms
    -tag_list

  The timestamp for a run can be easily acquired in one of two ways:
   Use the display_run subcommand.
   The dashboard run chart displays the timestamp value.

  Example deleting a single run under benchmark with key 123:
    mako delete_runs -benchmark_key=123 -run_key=456
  Example deleting runs under benchmark with key 123 after a certain time:
    mako delete_runs -benchmark_key=123 -run_timestamp_min_ms=946684800000
  Example deleting runs under benchmark with key 123 and with tags "a=1","b=5" applied:
    mako delete_runs -benchmark_key=123 -tag_list="a=1,b=5"
  Example deleting all runs for a benchmark with key 123:
    mako delete_runs -benchmark_key=123

`

	updateRunUsage = `
  update_run (<path> | -run_key=<key>)

  Update a single run.

  If a run key is provided, query for the run with the provided run key,
  present the user with an editor with the run info, and write back any edits
  that have been made to the run info.

  If a path is provided, update RunInfo with the RunInfo text proto provided at the path.

  The user must be on the owner's list of the benchmark for the run that is being updated.

  NOTE: Typically providing a run key is the preferred method of updating a run.
  The path option is available to allow the user to correct any potential errors
  from editing the RunInfo directly in the command line and use the saved text
  proto to update the run in a subsequent execution of the command.
`

	listTagsUsage = `
  list_tags -run_key=<key>

  List all tags from the specified run.

  NOTE: -run_key is required.

  NOTE: The index can be used in delete_tag command.

`

	addTagUsage = `
  add_tag -tag=<tag> [-benchmark_key=<key>] [-run_key=<key>] [-run_timestamp_min_ms=<minTimestampMs>] [-run_timestamp_max_ms=<maxTimestampMs>] [-run_build_id_min=<minBuildID>] [-run_build_id_max=<maxBuildID>] [-test_pass_id=<testPassID>] [-tag_list=<tags>]

  Add a tag to the specified runs.

  NOTE: -tag is required.

`

	deleteTagUsage = `
  delete_tag -tag=<tag> [-benchmark_key=<key>] [-run_key=<key>] [-run_timestamp_min_ms=<minTimestampMs>] [-run_timestamp_max_ms=<maxTimestampMs>] [-run_build_id_min=<minBuildID>] [-run_build_id_max=<maxBuildID>] [-test_pass_id=<testPassID>] [-tag_list=<tags>]

  Delete a tag from the specified run.

  NOTE: -tag is required.

`

	listSampleBatchesUsage = `
  list_sample_batches -run_key=<key>

  List all sample batches from the specified run.

  NOTE: -run_key is required.
`

	displaySampleBatchUsage = `
  display_sample_batch -batch_key=<key>

  Display all data for the provided sample batch.

  NOTE: -batch_key is required.
`

	maxDisplayedErrors = 100

	createBenchmarkTemplate = `#
# PLEASE COMPLETE THIS BenchmarkInfo TEMPLATE.
# FIND FULL DOCUMENTATION OF BenchmarkInfo AT https://github.com/google/mako/blob/master/spec/proto/mako.proto
#
# NOTE: Comments will automatically be removed.
#

# REQUIRED: specify a benchmark name.
benchmark_name: ""

# REQUIRED: specify a project name.
project_name: ""

# REQUIRED: specify one or more owners.
owner_list: "user@google.com"
owner_list: "group@prod.google.com"

# REQUIRED: declare information for the x-axis of your run charts.
# value_key: should be short and should not change
# type: TIMESTAMP/NUMERIC
# label: human-readable label to show on charts. Can be changed.
input_value_info: <
  value_key: "t"
  label: "time"
  type: TIMESTAMP
>

# OPTIONAL: (but recommended) declare one or more metrics
# value_key: should be short and should not change. Tests will write points with this key.
# label: human-readable label to show on charts. Can can changed.
metric_info_list: <
  value_key: "m1"
  label: "MetricName_ms"
>
metric_info_list: <
  value_key: "m2"
  label: "OtherMetric"
>
`

	makoTempDirName = "mako-cli"

	makoTempProtoFileExtension = "textpb"
)

/*********** HELPERS ***************/
func createTempTextProtoFile() (string, error) {
	tmpdir := path.Join(os.TempDir(), makoTempDirName)
	if err := os.Mkdir(tmpdir, 0700); err != nil && !os.IsExist(err) {
		return "", fmt.Errorf("error when creating temp dir at: %s. details: %s", tmpdir, err.Error())
	}

	tf, err := ioutil.TempFile(tmpdir, "")
	if err != nil {
		return "", fmt.Errorf("error creating temp file in tmp dir: %s. details: %s", tmpdir, err.Error())
	}
	tf.Close()

	oldpath := tf.Name()
	newpath := tf.Name() + "." + makoTempProtoFileExtension

	if err := os.Rename(oldpath, newpath); err != nil {
		return "", fmt.Errorf("error renaming temp file from %s to %s. details: %s", oldpath, newpath, err.Error())
	}

	return newpath, nil
}

func defaultEditor() string {
	editor := os.Getenv("EDITOR")
	editorProgram := ""
	if editor == "" {
		editorProgram = "vi"
	} else if editor == "gvim" || editor == "/usr/bin/gvim" {
		editorProgram = editor + " -f"
	} else {
		editorProgram = editor
	}

	return editorProgram
}

func parseProtoPathFromCommandLineArgs(args []string) (string, error) {
	if len(args) < 2 {
		return "", errors.New("proto path must be supplied")
	}

	return args[1], nil
}

func getProtoAtPathAsString() (string, subcommands.ExitStatus, error) {
	protoPath, err := parseProtoPathFromCommandLineArgs(flag.Args())
	if err != nil {
		return "", subcommands.ExitUsageError, err
	}

	path, err := readPathAsString(protoPath)
	if err != nil {
		return path, subcommands.ExitFailure, err
	}
	return path, subcommands.ExitSuccess, nil
}

func benchmarkInfoFromProvidedPath() (*pgpb.BenchmarkInfo, subcommands.ExitStatus, error) {
	str, status, err := getProtoAtPathAsString()
	if err != nil {
		return nil, status, err
	}

	bi := &pgpb.BenchmarkInfo{}
	if err := unmarshal(bi, str); err != nil {
		return nil, subcommands.ExitFailure, err
	}

	return bi, subcommands.ExitSuccess, nil
}

func runInfoFromProvidedPath() (*pgpb.RunInfo, subcommands.ExitStatus, error) {
	str, status, err := getProtoAtPathAsString()
	if err != nil {
		return nil, status, err
	}

	ri := &pgpb.RunInfo{}
	if err := unmarshal(ri, str); err != nil {
		return nil, subcommands.ExitFailure, err
	}

	return ri, subcommands.ExitSuccess, nil
}

func getUserEditedProto(textToEdit string) (string, string, error) {
	fpath, err := createTempTextProtoFile()
	if err != nil {
		return "", "", err
	}

	f, err := os.OpenFile(fpath, os.O_RDWR, 0700)
	defer f.Close()
	if err != nil {
		return "", "", err
	}

	if _, err := f.Write([]byte(textToEdit)); err != nil {
		return "", "", fmt.Errorf("error writing queried proto to temp file at: %s. details: %s", fpath, err.Error())
	}
	fmt.Fprint(outWriter, "created temp file at "+fpath+". opening for edit...")

	cmd := exec.Command(defaultEditor(), fpath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return "", "", fmt.Errorf("error launching vim to edit proto at: %s. details: %s", fpath, err.Error())
	}

	if err := cmd.Wait(); err != nil {
		return "", "", fmt.Errorf("error while waiting for user to finish editing. details: %s", err.Error())
	}

	editedContents, err := readPathAsString(fpath)
	if err != nil {
		return "", "", err
	}

	if textToEdit == editedContents {
		return "", "", errors.New("error: no changes were made. check " + fpath)
	}

	return fpath, editedContents, nil
}

func benchmarkInfoFromTemplate() (*pgpb.BenchmarkInfo, error) {
	path, contents, err := getUserEditedProto(createBenchmarkTemplate)
	if err != nil {
		return nil, err
	}

	updatedBenchmarkInfo := &pgpb.BenchmarkInfo{}
	if err := unmarshal(updatedBenchmarkInfo, contents); err != nil {
		return nil, fmt.Errorf("error: updated BenchmarkInfo is an invalid text proto.\n fix the BenchmarkInfo at %s and run\n\n mako create_benchmark %s\n\n error details: %s", path, path, err.Error())
	}

	return updatedBenchmarkInfo, nil
}

func userUpdatedBenchmarkInfo(bi *pgpb.BenchmarkInfo) (*pgpb.BenchmarkInfo, error) {
	path, contents, err := getUserEditedProto(proto.MarshalTextString(bi))
	if err != nil {
		return nil, err
	}

	updatedBenchmarkInfo := &pgpb.BenchmarkInfo{}
	if err := unmarshal(updatedBenchmarkInfo, contents); err != nil {
		return nil, fmt.Errorf("error: updated BenchmarkInfo is an invalid text proto.\n fix the BenchmarkInfo at %s and run\n\n mako update_benchmark %s\n\n error details: %s", path, path, err.Error())
	}

	return updatedBenchmarkInfo, nil
}

func userUpdatedRunInfo(ri *pgpb.RunInfo) (*pgpb.RunInfo, subcommands.ExitStatus, error) {
	path, contents, err := getUserEditedProto(proto.MarshalTextString(ri))
	if err != nil {
		return nil, subcommands.ExitFailure, err
	}

	updatedRunInfo := &pgpb.RunInfo{}
	if err := unmarshal(updatedRunInfo, contents); err != nil {
		return nil, subcommands.ExitFailure, fmt.Errorf("error: updated RunInfo is an invalid text proto.\n fix the RunInfo at %s and run\n\n mako update_run %s\n\n error details: %s", path, path, err.Error())
	}

	return updatedRunInfo, subcommands.ExitSuccess, nil
}

func readPathAsString(protoPath string) (string, error) {
	log.Infof("Loading text-proto file from path: %s\n", protoPath)
	data, err := ioutil.ReadFile(protoPath)
	if err != nil {
		return "", fmt.Errorf("error opening file: %s. details: %s", protoPath, err.Error())
	}
	return string(data), nil
}

func unmarshal(msg proto.Message, str string) error {
	// Expect data to hold benchmarkInfo instance.
	if err := proto.UnmarshalText(str, msg); err != nil {
		return fmt.Errorf("error attempting to marshal data from file. Error: %s", err.Error())
	}
	return nil
}

func queryRunInfo(ctx context.Context, s mako.Storage, q pgpb.RunInfoQuery) ([]*pgpb.RunInfo, error) {
	var runs []*pgpb.RunInfo
	first := true
	for first || q.GetCursor() != "" {
		first = false
		riqr, err := s.QueryRunInfo(ctx, &q)
		if err != nil {
			return nil, err
		}
		q.Cursor = riqr.Cursor
		runs = append(runs, riqr.GetRunInfoList()...)
	}
	return runs, nil
}

func queryOneRunInfo(ctx context.Context, s mako.Storage, runKey string) (*pgpb.RunInfo, error) {
	log.Infof("Querying for single run with key " + runKey)
	riqr, err := s.QueryRunInfo(ctx, &pgpb.RunInfoQuery{RunKey: proto.String(runKey)})
	if err != nil {
		return nil, err
	}

	ris := riqr.GetRunInfoList()

	if sz := len(ris); sz != 1 {
		return nil, fmt.Errorf("got %d results from query (want 1)", sz)
	}
	return ris[0], nil
}

func runInfo(ctx context.Context, s mako.Storage) (*pgpb.RunInfo, subcommands.ExitStatus, error) {
	if subargRunKey == "" {
		return runInfoFromProvidedPath()
	}
	original, err := queryOneRunInfo(ctx, s, subargRunKey)
	if err != nil {
		return nil, subcommands.ExitFailure, err
	}
	return userUpdatedRunInfo(original)
}

func updateRun(ctx context.Context, s mako.Storage) (subcommands.ExitStatus, error) {
	log.Info("Updating run")

	ri, status, err := runInfo(ctx, s)
	if err != nil {
		return status, err
	}

	if subargRunKey != "" {
		fmt.Fprint(outWriter, "Updating run "+subargRunKey+" with edited proto")
	}

	if err := updateRunInfo(ctx, s, ri); err != nil {
		return subcommands.ExitFailure, err
	}

	if subargRunKey != "" {
		fmt.Fprint(outWriter, "Updated run "+subargRunKey+" successfully")
	} else {
		fmt.Fprint(outWriter, "Run update successful")
	}

	return subcommands.ExitSuccess, nil
}

func updateRunInfo(ctx context.Context, s mako.Storage, ri *pgpb.RunInfo) error {
	if mr, err := s.UpdateRunInfo(ctx, ri); err != nil {
		return err
	} else if c := mr.GetCount(); c != 1 {
		return fmt.Errorf("got %d modified results (want 1)", c)
	}
	return nil
}

func userConfirm(msg string) (bool, error) {
	if *flagNoInteractive {
		return true, nil
	}

	fmt.Fprintf(outWriter, "%s (yes/no): ", msg)
	var response string
	if _, err := fmt.Scanln(&response); err != nil {
		return false, err
	}
	response = strings.TrimSpace(response)
	for _, v := range []string{"y", "Y", "yes", "Yes", "YES"} {
		if response == v {
			fmt.Fprint(outWriter, "Proceeding...")
			return true, nil
		}
	}
	fmt.Fprint(outWriter, "Aborted")
	return false, nil
}

func validateRunQueryArgs() (subcommands.ExitStatus, error) {
	if subargRunKey == "" && subargBenchmarkKey == "" {
		return subcommands.ExitUsageError, fmt.Errorf("One of -run_key or -benchmark_key must be specified")
	}
	if subargBenchmarkKey == "*" {
		return subcommands.ExitFailure, fmt.Errorf("-benchmark_key of '*' is not valid for per-run operations")
	}
	if (subargTimestampMinMs >= 0.0 || subargTimestampMaxMs >= 0.0) &&
		(subargMinBuildID >= 0.0 || subargMaxBuildID >= 0.0) {
		return subcommands.ExitFailure, fmt.Errorf("cannot filter runs based on both timestamp and build id currently")
	}
	if subargTimestampMaxMs >= 0.0 && subargTimestampMinMs > subargTimestampMaxMs {
		return subcommands.ExitFailure, fmt.Errorf("-run_timestamp_min_ms %f must not be greater than -run_timestamp_max_ms %f",
			subargTimestampMinMs, subargTimestampMaxMs)
	}
	if subargMaxBuildID >= 0 && subargMinBuildID > subargMaxBuildID {
		return subcommands.ExitFailure, fmt.Errorf("-run_build_id_min %d must be less than -run_build_id_max %d",
			subargMinBuildID, subargMaxBuildID)
	}
	parsedTags = nil
	if subargTagList != "" {
		parsedTags = strings.Split(subargTagList, ",")
	}
	for _, tag := range parsedTags {
		if tag == "" {
			return subcommands.ExitFailure, fmt.Errorf("-tag_list can not have empty tag string")
		}
	}
	return subcommands.ExitSuccess, nil
}

// buildRunQuery prepares and returns a RunInfoQuery with the given parameters.
// Parameters are expected to have been validated (with validateRunQueryArgs) before calling
// Parameters 'timestampMin', 'timestampMax', 'buildIDMin', and 'buildIDMax' are ignored if < 0.0
func buildRunQuery(benchmark, run string, timestampMin, timestampMax float64, buildIDMin,
	buildIDMax int64, testPassID string, tags []string) *pgpb.RunInfoQuery {

	// Build query
	runQuery := &pgpb.RunInfoQuery{}
	if run != "" {
		runQuery.RunKey = proto.String(run)
	}
	if benchmark != "" {
		runQuery.BenchmarkKey = proto.String(benchmark)
	}
	if timestampMin >= 0.0 {
		runQuery.MinTimestampMs = proto.Float64(timestampMin)
		runQuery.RunOrder = pgpb.RunOrder_TIMESTAMP.Enum()
	}
	if timestampMax >= 0.0 {
		runQuery.MaxTimestampMs = proto.Float64(timestampMax)
		runQuery.RunOrder = pgpb.RunOrder_TIMESTAMP.Enum()
	}
	if buildIDMin >= 0 {
		runQuery.MinBuildId = proto.Int64(buildIDMin)
		runQuery.RunOrder = pgpb.RunOrder_BUILD_ID.Enum()
	}
	if buildIDMax >= 0 {
		runQuery.MaxBuildId = proto.Int64(buildIDMax)
		runQuery.RunOrder = pgpb.RunOrder_BUILD_ID.Enum()
	}
	if testPassID != "" {
		runQuery.TestPassId = proto.String(testPassID)
	}
	if len(tags) > 0 {
		runQuery.Tags = tags
	}

	return runQuery
}

// represents a function that modifies a single run, returning
// true if the run was actually modified
type perRunOp func(runInfo pgpb.RunInfo) (bool, error)

// given a query and perRunOp, issue the query and edit each run with perRunOp
func updateEachRun(ctx context.Context, runQuery pgpb.RunInfoQuery, s mako.Storage, op perRunOp) error {
	runInfos, err := queryRunInfo(ctx, s, runQuery)
	if err != nil {
		return err
	}

	log.Infof("RunInfoQuery returned %d runs:", len(runInfos))
	for _, runInfo := range runInfos {
		log.Infof(" %s", runInfo.GetRunKey())
	}

	runCount := len(runInfos)

	// we don't consider runCount==0 an error
	if runCount == 0 {
		fmt.Fprint(outWriter, "Found 0 runs matching the input parameters. Nothing to do.")
		return nil
	}

	// User confirmation
	ok, err := userConfirm(fmt.Sprintf("Are you sure you want to update %d runs?", runCount))
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	numUpdated := 0
	var errs []error
	maxErrors := maxDisplayedErrors
	//TODO(b/138267664): Test with many runs, to evaluate whether we should parallelize this operation
	for _, runInfo := range runInfos {
		updated, err := op(*runInfo)
		if err != nil && len(errs) < maxErrors {
			errs = append(errs, err)
		} else if updated {
			numUpdated++
		}
	}

	fmt.Fprintf(outWriter, "Successfully edited %d runs\n", numUpdated)

	var errstrs []string
	for _, err := range errs {
		errstrs = append(errstrs, err.Error())
	}

	if len(errs) == maxErrors {
		return fmt.Errorf("encountered the following errors. Only first 100 errors shown:\n%s", strings.Join(errstrs, "\n"))
	} else if len(errs) > 0 {
		return fmt.Errorf("encountered the following errors:\n%s", strings.Join(errstrs, "\n"))
	}

	return nil
}

// -- benchmark commands --

func queryBenchmarkInfoByKey(ctx context.Context, s mako.Storage, benchmarkKey string) (*pgpb.BenchmarkInfo, error) {
	log.Infof("Querying for benchmark " + benchmarkKey)

	bqr, err := s.QueryBenchmarkInfo(ctx, &pgpb.BenchmarkInfoQuery{BenchmarkKey: proto.String(benchmarkKey)})
	if err != nil {
		return nil, err
	}

	bil := bqr.GetBenchmarkInfoList()

	if sz := len(bil); sz != 1 {
		return nil, fmt.Errorf("got %d results (want 1)", sz)
	}

	return bil[0], nil
}

func listBenchmarks(ctx context.Context, s mako.Storage) (subcommands.ExitStatus, error) {
	if subargProjectName == "" && subargBenchmarkName == "" && subargOwner == "" {
		return subcommands.ExitUsageError, fmt.Errorf("at least one of -benchmark_name, -project_name, or -owner must be specified")
	}

	bqr, err := s.QueryBenchmarkInfo(ctx, &pgpb.BenchmarkInfoQuery{
		BenchmarkName: proto.String(subargBenchmarkName),
		ProjectName:   proto.String(subargProjectName),
		Owner:         proto.String(subargOwner),
	})
	if err != nil {
		return subcommands.ExitFailure, err
	}

	if len(bqr.GetBenchmarkInfoList()) == 0 {
		fmt.Fprintln(errWriter, "No results! Check your arguments.")
	}

	for _, benchInfo := range bqr.GetBenchmarkInfoList() {
		fmt.Fprintln(outWriter, benchInfo.GetBenchmarkKey())
	}

	return subcommands.ExitSuccess, nil
}

func displayBenchmark(ctx context.Context, s mako.Storage) (subcommands.ExitStatus, error) {
	if subargBenchmarkKey == "" {
		return subcommands.ExitUsageError, errors.New("missing -benchmark_key")
	}

	bi, err := queryBenchmarkInfoByKey(ctx, s, subargBenchmarkKey)
	if err != nil {
		return subcommands.ExitFailure, err
	}

	fmt.Fprint(outWriter, proto.MarshalTextString(bi))

	if !subargDisplayRuns {
		return subcommands.ExitSuccess, nil
	}

	// Also list all the runs that are part of the benchmark.
	fmt.Fprint(outWriter, "Run keys associated with this benchmark:")
	if rqr, err := s.QueryRunInfo(ctx, &pgpb.RunInfoQuery{BenchmarkKey: proto.String(subargBenchmarkKey)}); err != nil {
		return subcommands.ExitFailure, err
	} else if len(rqr.GetRunInfoList()) == 0 {
		fmt.Fprintln(outWriter, "None")
	} else {
		for _, ri := range rqr.GetRunInfoList() {
			fmt.Fprintln(outWriter, ri.GetRunKey())
		}
	}

	return subcommands.ExitSuccess, nil
}

func createBenchmark(ctx context.Context, s mako.Storage) (subcommands.ExitStatus, error) {
	log.Info("Creating benchmark")

	var bi *pgpb.BenchmarkInfo
	var err error
	var status subcommands.ExitStatus

	if sz := len(flag.Args()); sz < 2 {
		bi, err = benchmarkInfoFromTemplate()
		if err != nil {
			status = subcommands.ExitFailure
		} else {
			status = subcommands.ExitSuccess
		}
	} else {
		bi, status, err = benchmarkInfoFromProvidedPath()
	}
	if err != nil {
		return status, err
	}

	cr, err := s.CreateBenchmarkInfo(ctx, bi)
	for _, warning := range cr.GetStatus().GetWarningMessages() {
		fmt.Fprintf(outWriter, "Warning during Benchmark creation: %s\n", warning)
	}
	if err != nil {
		return subcommands.ExitFailure, err
	}
	if len(cr.GetKey()) == 0 {
		return subcommands.ExitUsageError, fmt.Errorf("got empty creation key")
	}

	fmt.Fprint(outWriter, "Benchmark creation successful. ")
	fmt.Fprintf(outWriter, "Please add the benchmark_key: '%s' to your benchmark text-proto file.\n", cr.GetKey())
	return subcommands.ExitSuccess, nil
}

func benchmarkInfo(ctx context.Context, s mako.Storage) (*pgpb.BenchmarkInfo, subcommands.ExitStatus, error) {
	if subargBenchmarkKey == "" {
		return benchmarkInfoFromProvidedPath()
	}
	original, err := queryBenchmarkInfoByKey(ctx, s, subargBenchmarkKey)
	if err != nil {
		return nil, subcommands.ExitFailure, err
	}
	bi, err := userUpdatedBenchmarkInfo(original)
	if err != nil {
		return bi, subcommands.ExitFailure, err
	}
	return bi, subcommands.ExitSuccess, nil
}

func updateBenchmark(ctx context.Context, s mako.Storage) (subcommands.ExitStatus, error) {
	log.Info("Updating benchmark...")

	bi, status, err := benchmarkInfo(ctx, s)
	if err != nil {
		return status, err
	}

	if subargBenchmarkKey != "" {
		fmt.Fprint(outWriter, "Updating benchmark "+subargBenchmarkKey+" with edited proto.")
	}

	if err := updateBenchmarkInfo(ctx, s, bi); err != nil {
		return subcommands.ExitFailure, err
	}

	if subargBenchmarkKey != "" {
		fmt.Fprint(outWriter, "Updated benchmark "+subargBenchmarkKey+" successfully.")
	} else {
		fmt.Fprint(outWriter, "Benchmark update successful.\n")
	}

	return subcommands.ExitSuccess, nil
}

func updateBenchmarkInfo(ctx context.Context, s mako.Storage, bi *pgpb.BenchmarkInfo) error {
	mr, err := s.UpdateBenchmarkInfo(ctx, bi)

	for _, warning := range mr.GetStatus().GetWarningMessages() {
		fmt.Fprintf(outWriter, "Warning during Benchmark update: %s\n", warning)
	}

	if err != nil {
		return err
	} else if c := mr.GetCount(); c != 1 {
		return fmt.Errorf("got %d modified results (want 1)", c)
	}

	return nil
}

func deleteBenchmark(ctx context.Context, s mako.Storage) (subcommands.ExitStatus, error) {
	if subargBenchmarkKey == "" {
		return subcommands.ExitUsageError, fmt.Errorf("-benchmark_key missing")
	}

	log.Infof("Deleting benchmark with key: %q\n", subargBenchmarkKey)

	if modResp, err := s.DeleteBenchmarkInfo(ctx, &pgpb.BenchmarkInfoQuery{BenchmarkKey: proto.String(subargBenchmarkKey)}); err != nil {
		return subcommands.ExitFailure, err
	} else if c := modResp.GetCount(); c != 1 {
		return subcommands.ExitFailure, fmt.Errorf("got %d modified results (want 1)", c)
	}

	fmt.Fprintln(outWriter, "Deletion of benchmark successful.")
	return subcommands.ExitSuccess, nil
}

// -- annotation commands --

func listAnnotations(ctx context.Context, s mako.Storage) (subcommands.ExitStatus, error) {
	if subargRunKey == "" {
		return subcommands.ExitUsageError, fmt.Errorf("-run_key missing")
	}

	log.Infof("Listing annotations for run: %q\n", subargRunKey)

	ri, err := queryOneRunInfo(ctx, s, subargRunKey)
	if err != nil {
		return subcommands.ExitFailure, err
	}

	if len(ri.GetAnnotationList()) == 0 {
		fmt.Fprintln(outWriter, "Run has no annotations. Add some with the 'add_annotation' command")
		return subcommands.ExitSuccess, nil
	}

	fmt.Fprint(outWriter, "\nAnnotations List:")

	for i, annotation := range ri.GetAnnotationList() {
		fmt.Fprintf(outWriter, "#%d  %s\n", i, proto.MarshalTextString(annotation))
	}

	fmt.Fprint(outWriter, "\nThe index listed before each annotation can be used as input to the 'delete_annotation' command.")

	return subcommands.ExitSuccess, nil
}

func addAnnotation(ctx context.Context, s mako.Storage) (subcommands.ExitStatus, error) {
	if subargRunKey == "" {
		return subcommands.ExitUsageError, fmt.Errorf("-run_key missing")
	} else if subargValueKey == "" {
		return subcommands.ExitUsageError, fmt.Errorf("-value_key missing")
	} else if subargLabel == "" {
		return subcommands.ExitUsageError, fmt.Errorf("-label missing")
	} else if subargDesc == "" {
		return subcommands.ExitUsageError, fmt.Errorf("-description missing")
	}

	ra := pgpb.RunAnnotation{
		ValueKey:    proto.String(subargValueKey),
		Label:       proto.String(subargLabel),
		Description: proto.String(subargDesc)}

	log.Infof("Adding annotation: %s\n", proto.MarshalTextString(&ra))
	log.Infof("To run key: %q\n", subargRunKey)

	ri, err := queryOneRunInfo(ctx, s, subargRunKey)
	if err != nil {
		return subcommands.ExitFailure, err
	}

	ri.AnnotationList = append(ri.AnnotationList, &ra)

	if err := updateRunInfo(ctx, s, ri); err != nil {
		return subcommands.ExitFailure, err
	}

	fmt.Fprintln(outWriter, "Successfully added a new annotation.")

	return subcommands.ExitSuccess, nil
}

func deleteAnnotation(ctx context.Context, s mako.Storage) (subcommands.ExitStatus, error) {
	if subargRunKey == "" {
		return subcommands.ExitUsageError, fmt.Errorf("-run_key missing")
	} else if subargAnnotationIndex == -1 {
		return subcommands.ExitUsageError, fmt.Errorf("-annotation_index missing")
	}

	log.Infof("Deleting index %d of run key: %q\n", subargAnnotationIndex, subargRunKey)

	ri, err := queryOneRunInfo(ctx, s, subargRunKey)
	if err != nil {
		return subcommands.ExitFailure, err
	}

	if subargAnnotationIndex < 0 || subargAnnotationIndex >= len(ri.AnnotationList) {
		return subcommands.ExitFailure, fmt.Errorf("annotation index %d out of range [0, %d]", subargAnnotationIndex, len(ri.AnnotationList)-1)
	}

	// Make a copy because the element will be overwritten.
	ann := *ri.AnnotationList[subargAnnotationIndex]

	ri.AnnotationList = append(ri.AnnotationList[:subargAnnotationIndex],
		ri.AnnotationList[subargAnnotationIndex+1:]...)

	if err := updateRunInfo(ctx, s, ri); err != nil {
		return subcommands.ExitFailure, err
	}

	fmt.Fprintf(outWriter, "Successfully deleted annotation: %s\n", proto.MarshalTextString(&ann))

	return subcommands.ExitSuccess, nil
}

// -- run commands --

func listRuns(ctx context.Context, s mako.Storage) (subcommands.ExitStatus, error) {
	if subargBenchmarkKey == "" {
		return subcommands.ExitUsageError, fmt.Errorf("-benchmark_key missing")
	}
	if status, err := validateRunQueryArgs(); err != nil {
		return status, err
	}

	runQuery := buildRunQuery(subargBenchmarkKey, subargRunKey, subargTimestampMinMs,
		subargTimestampMaxMs, subargMinBuildID, subargMaxBuildID, subargTestPassID, parsedTags)
	runInfos, err := queryRunInfo(ctx, s, *runQuery)
	if err != nil {
		return subcommands.ExitFailure, err
	}

	if len(runInfos) == 0 {
		fmt.Fprintln(errWriter, "No results! Check your arguments.")
	}

	for _, runInfo := range runInfos {
		fmt.Fprintln(outWriter, *runInfo.RunKey)
	}

	return subcommands.ExitSuccess, nil
}

func displayRun(ctx context.Context, s mako.Storage) (subcommands.ExitStatus, error) {
	if subargRunKey == "" {
		return subcommands.ExitUsageError, fmt.Errorf("-run_key missing")
	}

	ri, err := queryOneRunInfo(ctx, s, subargRunKey)
	if err != nil {
		return subcommands.ExitFailure, err
	}

	fmt.Fprint(outWriter, proto.MarshalTextString(ri))

	return subcommands.ExitSuccess, nil
}

func deleteRuns(ctx context.Context, s mako.Storage) (subcommands.ExitStatus, error) {
	if status, err := validateRunQueryArgs(); err != nil {
		return status, err
	}
	runQuery := buildRunQuery(subargBenchmarkKey, subargRunKey, subargTimestampMinMs, subargTimestampMaxMs,
		subargMinBuildID, subargMaxBuildID, subargTestPassID, parsedTags)

	// Count the runs. This adds processing time, but it's important to let the user
	// know how much data will be deleted and get confirmation.
	runCount := 0
	for {
		response, err := s.QueryRunInfo(ctx, runQuery)
		if err != nil {
			return subcommands.ExitFailure, err
		}
		runCount += len(response.GetRunInfoList())
		runQuery.Cursor = proto.String(response.GetCursor())
		if response.GetCursor() == "" {
			break
		}
	}
	if runCount == 0 {
		return subcommands.ExitFailure, fmt.Errorf("no results for query: %+v", runQuery)
	}

	// User confirmation
	ok, err := userConfirm(fmt.Sprintf("Are you sure you want to delete %d runs?", runCount))
	if err != nil {
		return subcommands.ExitFailure, err
	}
	if !ok {
		return subcommands.ExitSuccess, nil
	}

	// Delete runs and child batches.
	// NOTE: The count here may be different than above if runs were created/deleted elsewhere.
	response, err := s.DeleteRunInfo(ctx, runQuery)
	if err != nil {
		return subcommands.ExitFailure, err
	}
	fmt.Fprintf(outWriter, "Done. Deleted %d runs.\n", response.GetCount())
	return subcommands.ExitSuccess, nil
}

// -- sample batch commands --

func listSampleBatches(ctx context.Context, s mako.Storage) (subcommands.ExitStatus, error) {
	if subargRunKey == "" {
		return subcommands.ExitUsageError, errors.New("-run_key is missing")
	}

	runInfo, err := queryOneRunInfo(ctx, s, subargRunKey)
	if err != nil {
		return subcommands.ExitFailure, err
	}

	if len(runInfo.GetBatchKeyList()) == 0 {
		return subcommands.ExitFailure, fmt.Errorf("no sample batches for run key %s", subargRunKey)
	}

	for _, batchKey := range runInfo.GetBatchKeyList() {
		fmt.Fprintln(outWriter, batchKey)
	}

	return subcommands.ExitSuccess, nil
}

func displaySampleBatch(ctx context.Context, s mako.Storage) (subcommands.ExitStatus, error) {
	if subargBatchKey == "" {
		return subcommands.ExitUsageError, errors.New("-batch_key missing")
	}

	response, err := s.QuerySampleBatch(ctx, &pgpb.SampleBatchQuery{
		BatchKey: proto.String(subargBatchKey),
	})
	if err != nil {
		return subcommands.ExitFailure, err
	}
	if len(response.GetSampleBatchList()) == 0 {
		return subcommands.ExitFailure, errors.New("no results! Check your arguments")
	}
	if len(response.GetSampleBatchList()) > 1 {
		return subcommands.ExitFailure, fmt.Errorf("got %d results, wanted 1", len(response.GetSampleBatchList()))
	}
	fmt.Fprint(outWriter, proto.MarshalTextString(response.GetSampleBatchList()[0]))

	return subcommands.ExitSuccess, nil
}

// -- tag commands --

func deleteTag(ctx context.Context, s mako.Storage) (subcommands.ExitStatus, error) {
	if subargTag == "" {
		return subcommands.ExitUsageError, fmt.Errorf("-tag missing")
	}
	if status, err := validateRunQueryArgs(); err != nil {
		return status, err
	}

	// doesn't make sense to delete tags from runs that don't have said tag, so let's add it to the query
	tagList := parsedTags
	exists := false
	for _, v := range tagList {
		if v == subargTag {
			exists = true
			break
		}
	}
	if !exists {
		tagList = append(tagList, subargTag)
	}

	runQuery := buildRunQuery(subargBenchmarkKey, subargRunKey, subargTimestampMinMs, subargTimestampMaxMs,
		subargMinBuildID, subargMaxBuildID, subargTestPassID, tagList)

	err := updateEachRun(ctx, *runQuery, s, func(ri pgpb.RunInfo) (bool, error) {
		removeTagIndex := -1
		for i, tag := range ri.Tags {
			if tag == subargTag {
				removeTagIndex = i
				break
			}
		}

		if removeTagIndex == -1 {
			return false, fmt.Errorf("Didn't find tag %s in run %s", subargTag, ri.RunKey)
		}
		ri.Tags = append(ri.Tags[:removeTagIndex], ri.Tags[removeTagIndex+1:]...)
		log.Infof("Deleting tag %s (index %d) from run %s", subargTag, removeTagIndex, ri.RunKey)
		if err := updateRunInfo(ctx, s, &ri); err != nil {
			return false, err
		}

		return true, nil
	})

	if err != nil {
		return subcommands.ExitFailure, err
	}
	return subcommands.ExitSuccess, nil
}

func listTags(ctx context.Context, s mako.Storage) (subcommands.ExitStatus, error) {
	if subargRunKey == "" {
		return subcommands.ExitUsageError, fmt.Errorf("-run_key missing")
	}

	log.Infof("Listing tags for run: %s\n", subargRunKey)

	ri, err := queryOneRunInfo(ctx, s, subargRunKey)
	if err != nil {
		return subcommands.ExitFailure, err
	}

	if len(ri.GetTags()) == 0 {
		fmt.Fprint(outWriter, "Run has no tags. Add some with the 'add_tag' command")
		return subcommands.ExitSuccess, nil
	}

	fmt.Fprint(outWriter, "Tags:")

	for i, tag := range ri.GetTags() {
		fmt.Fprintf(outWriter, "#%d %s\n", i, tag)
	}

	return subcommands.ExitSuccess, nil
}

func addTag(ctx context.Context, s mako.Storage) (subcommands.ExitStatus, error) {
	if subargTag == "" {
		return subcommands.ExitUsageError, fmt.Errorf("-tag missing")
	}
	if status, err := validateRunQueryArgs(); err != nil {
		return status, err
	}

	log.Infof("Adding tag: %q to run: %q\n", subargTag, subargRunKey)

	runQuery := buildRunQuery(subargBenchmarkKey, subargRunKey, subargTimestampMinMs, subargTimestampMaxMs,
		subargMinBuildID, subargMaxBuildID, subargTestPassID, parsedTags)
	err := updateEachRun(ctx, *runQuery, s, func(ri pgpb.RunInfo) (bool, error) {
		// don't add tag if it already exists
		for _, t := range ri.Tags {
			if t == subargTag {
				log.Infof("Run %s already has tag %s", ri.RunKey, subargTag)
				return false, nil
			}
		}

		ri.Tags = append(ri.Tags, subargTag)

		log.Infof("Adding tag %s to run %s", subargTag, ri.GetRunKey())

		// Update the run with new tag.
		if err := updateRunInfo(ctx, s, &ri); err != nil {
			return false, err
		}

		return true, nil
	})

	if err != nil {
		return subcommands.ExitFailure, err
	}
	return subcommands.ExitSuccess, nil
}

// cmd struct is used to wrap the subcommands.Command interface into
// an easy to construct struct.
type cmd struct {
	name     string
	synopsis string
	usage    string
	setFlags func(*flag.FlagSet)
	execute  func(ctx context.Context, s mako.Storage) (subcommands.ExitStatus, error)
}

func (c *cmd) Name() string     { return c.name }
func (c *cmd) Synopsis() string { return c.synopsis }
func (c *cmd) Usage() string {
	// first, print a one line example using all commands
	// TODO(b/123900836) finish this so that we don't need to be repetitive with all the usage consts at the top of the file
	return c.usage
}
func (c *cmd) SetFlags(f *flag.FlagSet) { c.setFlags(f) }
func (c *cmd) Execute(ctx context.Context, f *flag.FlagSet, args ...interface{}) subcommands.ExitStatus {
	status, err := c.execute(ctx, storageClient)
	if err != nil {
		fmt.Fprintf(errWriter, "***\n %v\n***\n", err)
		// Commands should fail with a status of ExitUsageError any time a user
		// should be shown usage information after the error. This is typically when
		// an argument or flag is missing. It is not typically done for invalid
		// values as the error in these cases is typically self explanatory.
		if status == subcommands.ExitUsageError {
			fmt.Fprintf(errWriter, "%v", c.Usage())
		}
	}
	return status
}

func registerMakoCommands(commander *subcommands.Commander) {
	// -- benchmark commands --

	commander.Register(
		&cmd{
			name:     "list_benchmarks",
			synopsis: "Lists matching benchmark keys.",
			usage:    listBenchmarkUsage,
			setFlags: func(f *flag.FlagSet) {
				f.StringVar(&subargBenchmarkName, "benchmark_name", "", "Name of the benchmark.")
				f.StringVar(&subargProjectName, "project_name", "", "Name of the project.")
				f.StringVar(&subargOwner, "owner", "", "An owner of the benchmark.")
			},
			execute: listBenchmarks}, "benchmarks")

	commander.Register(
		&cmd{
			name:     "display_benchmark",
			synopsis: "Displays a benchmark.",
			usage:    displayBenchmarkUsage,
			setFlags: func(f *flag.FlagSet) {
				f.StringVar(&subargBenchmarkKey, "benchmark_key", "", "Benchmark key to display")
				f.BoolVar(&subargDisplayRuns, "display_runs", false, "Display all associated run keys")
			},
			execute: displayBenchmark}, "benchmarks")

	commander.Register(
		&cmd{
			name:     "create_benchmark",
			synopsis: "Create a new benchmark.",
			usage:    createBenchmarkUsage,
			setFlags: func(f *flag.FlagSet) { return },
			execute:  createBenchmark}, "benchmarks")

	commander.Register(
		&cmd{
			name:     "update_benchmark",
			synopsis: "Update an existing benchmark.",
			usage:    updateBenchmarkUsage,
			setFlags: func(f *flag.FlagSet) {
				f.StringVar(&subargBenchmarkKey, "benchmark_key", "", "Benchmark key to update")
			},
			execute: updateBenchmark}, "benchmarks")

	commander.Register(
		&cmd{
			name:     "delete_benchmark",
			synopsis: "Deletes a new benchmark.",
			usage:    deleteBenchmarkUsage,
			setFlags: func(f *flag.FlagSet) {
				f.StringVar(&subargBenchmarkKey, "benchmark_key", "", "Benchmark key to remove")
			},
			execute: deleteBenchmark}, "benchmarks")

	// -- annotation commands --

	commander.Register(
		&cmd{
			name:     "list_annotations",
			synopsis: "Lists the annotations for a run.",
			usage:    listAnnotationsUsage,
			setFlags: func(f *flag.FlagSet) { f.StringVar(&subargRunKey, "run_key", "", "Run key to list annotations for") },
			execute:  listAnnotations}, "annotations")

	commander.Register(
		&cmd{
			name:     "add_annotation",
			synopsis: "Add an annotation to a run.",
			usage:    addAnnotationUsage,
			setFlags: func(f *flag.FlagSet) {
				f.StringVar(&subargRunKey, "run_key", "", "Run key to add an annotation to")
				f.StringVar(&subargValueKey, "value_key", "", "Value key for annotation")
				f.StringVar(&subargLabel, "label", "", "Label for annotation")
				f.StringVar(&subargDesc, "description", "", "Description for annotation")
			},
			execute: addAnnotation}, "annotations")

	commander.Register(
		&cmd{
			name:     "delete_annotation",
			synopsis: "Delete an annotation.",
			usage:    deleteAnnotationUsage,
			setFlags: func(f *flag.FlagSet) {
				f.StringVar(&subargRunKey, "run_key", "", "Run key to add an annotation to")
				f.IntVar(&subargAnnotationIndex, "annotation_index", -1, "Index of the annotation to delete according to list_annotations output.")
			},
			execute: deleteAnnotation}, "annotations")

	// -- run commands --

	commander.Register(
		&cmd{
			name:     "list_runs",
			synopsis: "Lists matching run keys.",
			usage:    listRunsUsage,
			setFlags: func(f *flag.FlagSet) {
				f.StringVar(&subargBenchmarkKey, "benchmark_key", "", "Benchmark associated with run.")
				f.Float64Var(&subargTimestampMinMs, "run_timestamp_min_ms", -1.0,
					"Min timestamp of runs to list in milliseconds.")
				f.Float64Var(&subargTimestampMaxMs, "run_timestamp_max_ms", -1.0,
					"Max timestamp of runs to list in milliseconds.")
				f.Int64Var(&subargMinBuildID, "run_build_id_min", -1, "Min build ID of runs to list")
				f.Int64Var(&subargMaxBuildID, "run_build_id_max", -1, "Max build ID of runs to list")
				f.StringVar(&subargTestPassID, "test_pass_id", "",
					"Test Pass ID associated with run - multiple runs can be grouped as a single (ID'ed) test pass")
				f.StringVar(&subargTagList, "tag_list", "",
					"Comma delimited list of tags. Only runs that contain all of these tags will be returned.")
			},
			execute: listRuns}, "runs")

	commander.Register(
		&cmd{
			name:     "display_run",
			synopsis: "Displays a run.",
			usage:    displayRunUsage,
			setFlags: func(f *flag.FlagSet) { f.StringVar(&subargRunKey, "run_key", "", "Run key to display.") },
			execute:  displayRun}, "runs")

	commander.Register(
		&cmd{
			name:     "delete_runs",
			synopsis: "Delete one or more runs and associated sample batch data.",
			usage:    deleteRunsUsage,
			setFlags: func(f *flag.FlagSet) {
				f.StringVar(&subargBenchmarkKey, "benchmark_key", "", "Benchmark associated with runs.")
				f.StringVar(&subargRunKey, "run_key", "", "Specific run key to delete.")
				f.Float64Var(&subargTimestampMinMs, "run_timestamp_min_ms", -1.0,
					"Min timestamp of runs to delete in milliseconds.")
				f.Float64Var(&subargTimestampMaxMs, "run_timestamp_max_ms", -1.0,
					"Max timestamp of runs to delete in milliseconds.")
				f.Int64Var(&subargMinBuildID, "run_build_id_min", -1, "Min build ID of runs to delete.")
				f.Int64Var(&subargMaxBuildID, "run_build_id_max", -1, "Max build ID of runs to delete.")
				f.StringVar(&subargTestPassID, "test_pass_id", "",
					"Test Pass ID associated with run - multiple runs can be grouped as a single (ID'ed) test pass.")
				f.StringVar(&subargTagList, "tag_list", "",
					"Comma delimited list of tags. Only runs that contain all of these tags will be deleted.")
			},
			execute: deleteRuns}, "runs")

	commander.Register(
		&cmd{
			name:     "update_run",
			synopsis: "Update single run",
			usage:    updateRunUsage,
			setFlags: func(f *flag.FlagSet) {
				f.StringVar(&subargRunKey, "run_key", "", "Specific run key to update")
			},
			execute: updateRun}, "runs")

	// -- sample batch commands --

	commander.Register(
		&cmd{
			name:     "list_sample_batches",
			synopsis: "List all sample batches from a run.",
			usage:    listSampleBatchesUsage,
			setFlags: func(f *flag.FlagSet) {
				f.StringVar(&subargRunKey, "run_key", "", "Lists sample batches from this run")
			},
			execute: listSampleBatches}, "sample batches")

	commander.Register(
		&cmd{
			name:     "display_sample_batch",
			synopsis: "Display a sample batch.",
			usage:    displaySampleBatchUsage,
			setFlags: func(f *flag.FlagSet) {
				f.StringVar(&subargBatchKey, "batch_key", "", "Displays all data from this sample batch")
			},
			execute: displaySampleBatch}, "sample batches")

	// -- tag commands --

	commander.Register(
		&cmd{
			name:     "list_tags",
			synopsis: "List all tags from the specified run.",
			usage:    listTagsUsage,
			setFlags: func(f *flag.FlagSet) {
				f.StringVar(&subargRunKey, "run_key", "", "Lists tags from this run.")
			},
			execute: listTags}, "tags")

	commander.Register(
		&cmd{
			name:     "add_tag",
			synopsis: "Add a tag to the specified runs.",
			usage:    addTagUsage,
			setFlags: func(f *flag.FlagSet) {
				f.StringVar(&subargBenchmarkKey, "benchmark_key", "", "Benchmark the runs are part of.")
				f.StringVar(&subargTag, "tag", "", "The tag to add.")
				f.StringVar(&subargRunKey, "run_key", "", "Specific run key to edit")
				f.Float64Var(&subargTimestampMinMs, "run_timestamp_min_ms", -1.0,
					"Min timestamp of runs to edit in milliseconds.")
				f.Float64Var(&subargTimestampMaxMs, "run_timestamp_max_ms", -1.0,
					"Max timestamp of runs to edit in milliseconds.")
				f.Int64Var(&subargMinBuildID, "run_build_id_min", -1, "Min build ID of runs to edit.")
				f.Int64Var(&subargMaxBuildID, "run_build_id_max", -1, "Max build ID of runs to edit.")
				f.StringVar(&subargTestPassID, "test_pass_id", "",
					"Test Pass ID associated with run - multiple runs can be grouped as a single (ID'ed) test pass.")
				f.StringVar(&subargTagList, "tag_list", "",
					"Comma delimited list of tags. Only runs that contain all of these tags will have additional tag addded.")
			},
			execute: addTag}, "tags")

	commander.Register(
		&cmd{
			name:     "delete_tag",
			synopsis: "Delete the specified tag.",
			usage:    deleteTagUsage,
			setFlags: func(f *flag.FlagSet) {
				f.StringVar(&subargBenchmarkKey, "benchmark_key", "", "Benchmark the runs are part of.")
				f.StringVar(&subargTag, "tag", "", "The tag to remove.")
				f.StringVar(&subargRunKey, "run_key", "", "Specific run key to edit.")
				f.Float64Var(&subargTimestampMinMs, "run_timestamp_min_ms", -1.0,
					"Min timestamp of runs to edit in milliseconds.")
				f.Float64Var(&subargTimestampMaxMs, "run_timestamp_max_ms", -1.0,
					"Max timestamp of runs to edit in milliseconds.")
				f.Int64Var(&subargMinBuildID, "run_build_id_min", -1, "Min build ID of runs to edit.")
				f.Int64Var(&subargMaxBuildID, "run_build_id_max", -1, "Max build ID of runs to edit.")
				f.StringVar(&subargTestPassID, "test_pass_id", "",
					"Test Pass ID associated with run - multiple runs can be grouped as a single (ID'ed) test pass.")
				f.StringVar(&subargTagList, "tag_list", "",
					"Comma delimited list of tags. Only runs that contain all of these tags will have tag removed.")
			},
			execute: deleteTag}, "tags")

	// These flags are pulled out into main help menu.
	commander.ImportantFlag("nointeractive")

	commander.Register(commander.HelpCommand(), "")
	commander.Register(commander.FlagsCommand(), "")

}

// Run is the entrypoint to the CLI logic.
func Run(ctx context.Context, ts mako.Storage, stdout io.Writer, stderr io.Writer, commander *subcommands.Commander) subcommands.ExitStatus {
	// we redirect output to our custom writer, which will be checked at the end of the test
	outWriter = stdout
	errWriter = stderr

	registerMakoCommands(commander)
	commander.Output = outWriter
	commander.Error = errWriter

	storageClient = ts
	return commander.Execute(ctx)
}
