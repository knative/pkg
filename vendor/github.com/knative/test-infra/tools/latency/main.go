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

package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"golang.org/x/net/context"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

var ctx = context.Background()
var client *storage.Client
var e2eDuration = regexp.MustCompile("\\((\\d+\\.\\d+)s\\)")
var sourceDir string

type Metric struct {
	values        []int
	_percentile95 int
	_outliers     []int
	_processed    bool
	_name         string
}

type MetricsMap map[string]Metric

const (
	logDir     = "logs/"
	bucketName = "knative-prow"

	// Taken from github.com/bwmarrin/snowflake
	snowflakeEpoch     = int64(1288834974657)
	snowflakeTimeShift = uint64(10 + 12)
	snowflakeNodeShift = uint64(12)
)

func (m *MetricsMap) addValue(name string, value int) {
	(*m)[name] = (*m)[name].addValue(name, value)
}

func (m Metric) addValue(name string, value int) Metric {
	values := append(m.values, value)
	m.values = values
	m._processed = false
	m._name = name
	return m
}

func (m *Metric) process() {
	if (*m)._processed {
		return
	}
	tmp := make([]int, len((*m).values))
	copy(tmp, (*m).values)
	sort.Sort(sort.IntSlice(tmp))
	size := len(tmp)
	// Arrays are 0-based
	index95 := int(math.Round(float64(size)*95/100)) - 1
	(*m)._percentile95 = 0
	(*m)._outliers = make([]int, 0)
	(*m)._processed = true
	if index95 >= size {
		log.Printf("Metric has insuficient samples (%d)", size)
		return
	}
	(*m)._percentile95 = tmp[index95]
	(*m)._outliers = tmp[index95+1:]
}

func (m *Metric) Percentile95() int {
	(*m).process()
	return (*m)._percentile95
}

func (m *Metric) Outliers() []int {
	(*m).process()
	return (*m)._outliers
}

func (m *Metric) WorstOutlier() int {
	(*m).process()
	if len((*m)._outliers) < 1 {
		log.Printf("No outliers for %s (not enough data)", (*m)._name)
		return 0
	}
	return (*m)._outliers[len((*m)._outliers)-1]
}

func sameDate(d1, d2 time.Time) bool {
	return d1.Year() == d2.Year() && d1.YearDay() == d2.YearDay()
}

func listBuilds(dir string, rangeStart, rangeEnd int) []int {
	var builds []int
	it := client.Bucket(bucketName).Objects(ctx, &storage.Query{
		Prefix:    logDir + dir + "/",
		Delimiter: "/",
	})
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatalf("Error iterating: %v", err)
		}
		if attrs.Prefix != "" {
			buildNumber, err := strconv.Atoi(path.Base(attrs.Prefix))
			if err == nil && buildNumber >= rangeStart && buildNumber <= rangeEnd {
				builds = append(builds, buildNumber)
			}
		}
	}
	sort.Sort(sort.IntSlice(builds))
	return builds
}

func atoi(str, name string) int {
	value, err := strconv.Atoi(str)
	if err != nil {
		log.Fatalf("Unexpected string '%s' for %s: %v", str, name, err)
	}
	return value
}

func atof(str, name string) float64 {
	value, err := strconv.ParseFloat(str, 64)
	if err != nil {
		log.Fatalf("Unexpected string '%s' for %s: %v", str, name, err)
	}
	return value
}

func readGcsFile(filename string) ([]byte, error) {
	o := client.Bucket(bucketName).Object(filename)
	if _, err := o.Attrs(ctx); err != nil {
		return []byte(fmt.Sprintf("Cannot get attributes of '%s'", filename)), err
	}
	f, err := o.NewReader(ctx)
	if err != nil {
		return []byte(fmt.Sprintf("Cannot open '%s'", filename)), err
	}
	defer f.Close()
	contents, err := ioutil.ReadAll(f)
	if err != nil {
		return []byte(fmt.Sprintf("Cannot read '%s'", filename)), err
	}
	return contents, nil
}

func parseMetricEntry(fields []string, dataStartIndex int) (string, int) {
	// Use first slice only
	nameFields := strings.Split(fields[dataStartIndex], "/")
	name := fields[dataStartIndex]
	if len(nameFields) == 3 {
		name = nameFields[0] + "/" + nameFields[2]
	} else if len(nameFields) > 3 {
		log.Printf("Unexpected metric name '%s' (too many parts)", fields[dataStartIndex])
		return "", 0
	}
	startTime := atoi(fields[dataStartIndex+1], "start time")
	endTime := atoi(fields[dataStartIndex+2], "end time")
	duration := endTime - startTime
	if duration < 1 {
		log.Printf("Unexpected duration %d for %s", duration, name)
		return "", 0
	}
	return name, duration
}

func parseLog(dir string, dateRestriction time.Time, metrics MetricsMap) {
	buildDir := logDir + dir
	log.Printf("Parsing '%s'", buildDir)
	contents, err := readGcsFile(buildDir + "/started.json")
	if err != nil {
		log.Printf("%s, skipping: %v", contents, err)
		return
	}
	if !dateRestriction.IsZero() {
		jsonStruct := make(map[string]interface{})
		if err = json.Unmarshal(contents, &jsonStruct); err != nil {
			log.Printf("Error parsing JSON '%s', skipping: %v", contents, err)
			return
		}
		jobStarted := time.Unix(int64(jsonStruct["timestamp"].(float64)), 0)
		log.Printf("Job started on %s", jobStarted)
		if !sameDate(dateRestriction, jobStarted) {
			log.Printf("Job start date is not %s, skipping", dateRestriction)
			return
		}
	}
	logFile := buildDir + "/build-log.txt"
	log.Printf("Parsing '%s'", logFile)
	o := client.Bucket(bucketName).Object(logFile)
	if _, err := o.Attrs(ctx); err != nil {
		log.Printf("Cannot get attributes of '%s', assuming not ready yet: %v", logFile, err)
		return
	}
	f, err := o.NewReader(ctx)
	if err != nil {
		log.Fatalf("Error opening '%s': %v", logFile, err)
	}
	defer f.Close()
	startedE2ETests := false
	scanner := bufio.NewScanner(f)
	sampleSize := make(map[string]int, 1)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		// TODO(adrcunha): This is ugly, use a better marker.
		// I0711 15:08:31.815] CREATING TEST CLUSTER
		if len(fields) == 5 && fields[2] == "CREATING" && fields[3] == "TEST" && fields[4] == "CLUSTER" {
			startedE2ETests = true
		}
		// I0911 16:05:43.464] ==== CREATING TEST CLUSTER ====
		// I0424 14:10:56.870] ==== CREATING TEST CLUSTER IN US-CENTRAL1 ====
		if len(fields) >= 7 && fields[3] == "CREATING" && fields[4] == "TEST" && fields[5] == "CLUSTER" {
			startedE2ETests = true
		}
		// I0711 22:23:20.729] --- PASS: TestHelloWorldFromShell (36.85s)
		if startedE2ETests && len(fields) == 6 && fields[2] == "---" && fields[3] == "PASS:" {
			name := "E2E:" + fields[4]
			if !e2eDuration.MatchString(fields[5]) {
				log.Printf("Unrecognized test duration '%s'", fields[5])
				continue
			}
			duration := int(atof(e2eDuration.FindStringSubmatch(fields[5])[1], "E2E test duration") * 1000000000.0)
			if duration < 1 {
				log.Printf("Unexpected duration '%s' for %s", fields[5], name)
			}
			metrics.addValue(name, duration)
			sampleSize[name] += 1
		}
		name := ""
		duration := 0
		// I0629 06:12:45.919] info	test/logging.go:64	metric WaitForEndpointState/HelloWorldServesText 1530235518250057368 1530235518337078608 87.02124ms
		if len(fields) == 9 && fields[2] == "info" && fields[4] == "metric" {
			name, duration = parseMetricEntry(fields, 5)
		}
		// After https://github.com/knative/serving/pull/1366
		// I0726 23:25:49.262] info	TestBlueGreenRoute	test/logging.go:64	metric WaitForRevision/prodmmueefvc-00002/RevisionIsReady 1532647086352596120 1532647386394195502 5m0.041599382s
		if len(fields) == 10 && fields[2] == "info" && fields[5] == "metric" {
			name, duration = parseMetricEntry(fields, 6)
		}
		// Afer https://github.com/knative/pkg/pull/122
		// I0726 23:25:49.262] 2018-10-12T18:18:06.835-0700    info    TestBlueGreenRoute      test/logging.go:64      metric WaitForRevision/prodmmueefvc-00002/RevisionIsReady 1532647086352596120 1532647386394195502 5m0.041599382s
		if len(fields) == 11 && fields[3] == "info" && fields[6] == "metric" {
			name, duration = parseMetricEntry(fields, 7)
		}
		if name == "" {
			continue
		}
		metrics.addValue(name, duration)
		sampleSize[name] += 1
	}
	log.Printf("Finished parsing '%s'", logFile)
	totalMetrics := 0
	for name, count := range sampleSize {
		log.Printf("* Collected %d samples for metric '%s'", count, name)
		totalMetrics += count
	}
	log.Printf("* Collected a total of %d samples", totalMetrics)
}

func writeXml(f *os.File, s string) {
	_, err := f.WriteString(s + "\n")
	if err != nil {
		log.Fatalf("Cannot write to '%s': %v", f.Name(), err)
	}
}

func writeMetricXmlProperty(f *os.File, name string, value int) {
	writeXml(f, fmt.Sprintf(" <testcase class_name=\"latency_metrics\" name=\"%s\" time=\"0\">", name))
	writeXml(f, "  <properties>")
	writeXml(f, fmt.Sprintf("   <property name=\"latency\" value=\"%.2f\"></property>", float64(value)/1000000.0))
	writeXml(f, "  </properties>")
	writeXml(f, " </testcase>")
}

func createXml(dir string, metrics MetricsMap) {
	outputFile := dir + "/junit_bazel.xml"
	f, err := os.Create(outputFile)
	if err != nil {
		log.Fatalf("Cannot create '%s': %v", outputFile, err)
	}
	defer f.Close()
	// TODO: use encoding/xml?
	writeXml(f, "<testsuite>")
	for metric, data := range metrics {
		writeMetricXmlProperty(f, metric, data.Percentile95())
		writeMetricXmlProperty(f, metric+"/outlier:worst", data.WorstOutlier())
	}
	writeXml(f, "</testsuite>")
}

// Regenerate creates and returns an approximate snowflake ID based on a UNIX timestamp
func regenerateSnowflakeID(unixTimestamp int64) int64 {
	nodeTime := int64(unixTimestamp * 1000)
	nodeNode := int64(0)
	nodeStep := int64(0)

	r := int64((nodeTime-snowflakeEpoch)<<snowflakeTimeShift |
		(nodeNode << snowflakeNodeShift) |
		(nodeStep),
	)
	return r
}

func getSnowflakeUnixTimestamp(n int64) int64 {
	return ((int64(n) >> snowflakeTimeShift) + snowflakeEpoch) / 1000
}

func getBuildRangeFromSnowflakeID(latestBuild int64, hoursBack int64, dateRestriction time.Time) (int64, int64) {
	latestBuildTimestamp := getSnowflakeUnixTimestamp(latestBuild)
	log.Printf("Latest build timestamp is %s", time.Unix(int64(latestBuildTimestamp), 0))
	// Build 731 started 7/1/2018 12:32AM
	// No metrics exist before build 680
	buildRangeStart := int64(0)
	buildRangeEnd := int64(999999999999)
	if !dateRestriction.IsZero() {
		buildRangeStart = regenerateSnowflakeID(latestBuildTimestamp - (hoursBack+24)*60*60)
		buildRangeStartTimestamp := getSnowflakeUnixTimestamp(buildRangeStart)
		log.Printf("Build range start is %d, timestamp is %s", buildRangeStart, time.Unix(buildRangeStartTimestamp, 0))
		buildRangeEnd = regenerateSnowflakeID(buildRangeStartTimestamp + (24+24)*60*60)
		buildRangeEndTimestamp := getSnowflakeUnixTimestamp(buildRangeEnd)
		log.Printf("Build range end is %d, timestamp is %s", buildRangeEnd, time.Unix(buildRangeEndTimestamp, 0))
	}
	return buildRangeStart, buildRangeEnd
}

func getBuildRangeFromIncrementalID(latestBuild int64, hoursBack int64, dateRestriction time.Time) (int64, int64) {
	// Build 731 started 7/1/2018 12:32AM
	// No metrics exist before build 680
	buildRangeStart := int64(731)
	buildRangeEnd := int64(999999999999)
	if !dateRestriction.IsZero() {
		buildRangeStart = latestBuild - hoursBack - 24
		log.Printf("Build range start is %d", buildRangeStart)
		if buildRangeStart < 731 {
			buildRangeStart = 731
		}
		buildRangeEnd = buildRangeStart + 24 + 24
		log.Printf("Build range end is %d", buildRangeEnd)
	}
	return buildRangeStart, buildRangeEnd
}

func main() {
	fullParsing := flag.Bool("full-parsing", false, "Whether to parse all logs in the bucket, or just the logs from --days-back before")
	artifactsDir := flag.String("artifacts-dir", "./artifacts", "Directory to store the generated XML file")
	serviceAccount := flag.String("service-account", os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"), "JSON key file for service account to use")
	flag.StringVar(&sourceDir, "source-directory", "", "Directory in Testgrid bucket containing the logs to parse")
	daysBack := flag.Int("days-back", 1, "How many days back to fetch the logs")
	flag.Parse()

	if sourceDir == "" {
		log.Fatal("The --source-directory flag is mandatory")
	}

	hoursBack := *daysBack * 24
	dateRestriction := time.Now().Add(-time.Duration(hoursBack) * time.Hour)
	if *fullParsing {
		if os.Getenv("JOB_TYPE") != "" {
			log.Fatal("Full parsing requested on a Prow job, this will generate bad data")
		}
		dateRestriction = time.Time{}
	}
	log.Printf("Date restriction is %s", dateRestriction)
	var err error
	client, err = storage.NewClient(ctx, option.WithCredentialsFile(*serviceAccount))
	if err != nil {
		log.Fatalf("Failed to create GCS client: %v", err)
		return
	}
	contents, err := readGcsFile(logDir + sourceDir + "/latest-build.txt")
	if err != nil {
		log.Fatalf("Cannot get latest build number. %s: %v", contents, err)
	}
	latestBuild := int64(atoi(string(contents), "latest build"))
	// No metrics exist before build 680
	buildRangeStart := int64(680)
	buildRangeEnd := int64(999999999999)
	if dateRestriction.After(time.Date(2018, 8, 2, 23, 59, 59, 0, time.Local)) {
		buildRangeStart, buildRangeEnd = getBuildRangeFromSnowflakeID(latestBuild, int64(hoursBack), dateRestriction)
	} else {
		buildRangeStart, buildRangeEnd = getBuildRangeFromIncrementalID(latestBuild, int64(hoursBack), dateRestriction)
	}
	metrics := make(MetricsMap)
	for _, buildNumber := range listBuilds(sourceDir, int(buildRangeStart), int(buildRangeEnd)) {
		parseLog(fmt.Sprintf("%s/%d", sourceDir, buildNumber), dateRestriction, metrics)
	}
	if len(metrics) == 0 {
		log.Println("No metrics to aggregate")
	} else {
		createXml(*artifactsDir, metrics)
		log.Println("Metrics aggregation finished successfully")
		totalMetrics := 0
		for metric, data := range metrics {
			count := len(data.values)
			log.Printf("* Collected %d samples for metric '%s'", count, metric)
			totalMetrics += count
		}
		log.Printf("* Collected a total of %d samples", totalMetrics)
	}
	return
}
