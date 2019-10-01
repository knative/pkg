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

// Resultstore converts --build=gs://prefix/JOB/NUMBER from prow's pod-utils to a ResultStore invocation suite, which it optionally will --upload=gcp-project.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/storage"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/yaml"

	"k8s.io/test-infra/prow/flagutil"
	"k8s.io/test-infra/prow/logrusutil"
	"k8s.io/test-infra/testgrid/config"
	"k8s.io/test-infra/testgrid/metadata"
	"k8s.io/test-infra/testgrid/resultstore"
	"k8s.io/test-infra/testgrid/util/gcs"
)

var re = regexp.MustCompile(`( ?|^)\[[^]]+\]( |$)`)

// Converts "[k8s.io] hello world [foo]" into "hello world", []string{"k8s.io", "foo"}
func stripTags(str string) (string, []string) {
	tags := re.FindAllString(str, -1)
	for i, w := range tags {
		w = strings.TrimSpace(w)
		tags[i] = w[1 : len(w)-1]
	}
	var reals []string
	for _, p := range re.Split(str, -1) {
		if p == "" {
			continue
		}
		reals = append(reals, p)
	}
	return strings.Join(reals, " "), tags
}

type options struct {
	path           gcs.Path
	jobs           flagutil.Strings
	deadline       time.Duration
	latest         int
	override       bool
	account        string
	gcsAuth        bool
	pending        bool
	repeat         time.Duration
	project        string
	secret         string
	testgridConfig string
}

func (o *options) parse(flags *flag.FlagSet, args []string) error {
	flags.Var(&o.path, "build", "Download a specific gs://bucket/to/job/build-1234 url (instead of latest builds for each --job)")
	flags.Var(&o.jobs, "job", "Configures specific jobs to update (repeatable, all jobs when --job and --build are both empty)")
	flags.StringVar(&o.testgridConfig, "config", "gs://k8s-testgrid/config", "Path to local/testgrid/config.pb or gs://bucket/testgrid/config.pb")
	flags.IntVar(&o.latest, "latest", 1, "Configures the number of latest builds to migrate")
	flags.BoolVar(&o.override, "override", false, "Replace the existing ResultStore data for each build")
	flags.StringVar(&o.account, "service-account", "", "Authenticate with the service account at specified path")
	flags.BoolVar(&o.gcsAuth, "gcs-auth", false, "Use service account for gcs auth if set (default auth if unset)")
	flags.BoolVar(&o.pending, "pending", false, "Include pending results when set (otherwise ignore them)")
	flags.StringVar(&o.project, "upload", "", "Upload results to specified gcp project instead of stdout")
	flags.StringVar(&o.secret, "secret", "", "Use the specified secret guid instead of randomly generating one.")
	flags.DurationVar(&o.deadline, "deadline", 0, "Timeout after the specified deadling duration (use 0 for no deadline)")
	flags.DurationVar(&o.repeat, "repeat", 0, "Repeatedly transfer after sleeping for this duration (exit after one run when 0)")
	flags.Parse(args)
	return nil
}

func parseOptions() options {
	var o options
	if err := o.parse(flag.CommandLine, os.Args[1:]); err != nil {
		logrus.WithError(err).Fatal("Invalid flags")
	}
	return o
}

func main() {
	logrusutil.ComponentInit("storeship")

	opt := parseOptions()
	for {
		err := run(opt)
		if opt.repeat == 0 {
			if err != nil {
				logrus.WithError(err).Fatal("Failed transfer")
			}
			return
		}
		if err != nil {
			logrus.WithError(err).Error("Failed transfer")
		}
		if opt.repeat > time.Second {
			logrus.Infof("Sleeping for %s...", opt.repeat)
		}
		time.Sleep(opt.repeat)
	}
}

func str(inv interface{}) string {
	buf, err := yaml.Marshal(inv)
	if err != nil {
		panic(err)
	}
	return string(buf)
}

func print(inv ...interface{}) {
	for _, i := range inv {
		fmt.Println(str(i))
	}
}

func trailingSlash(s string) string {
	if strings.HasSuffix(s, "/") {
		return s
	}
	return s + "/"
}

func run(opt options) error {
	var ctx context.Context
	var cancel context.CancelFunc
	if opt.deadline > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), opt.deadline)
	} else {
		ctx, cancel = context.WithCancel(context.Background())
	}
	defer cancel()

	var gcsAccount string
	if opt.gcsAuth {
		gcsAccount = opt.account
	}
	storageClient, err := storageClient(ctx, gcsAccount)
	if err != nil {
		return fmt.Errorf("storage client: %v", err)
	}

	logrus.WithFields(logrus.Fields{"testgrid": opt.testgridConfig}).Info("Reading testgrid config...")
	cfg, err := config.Read(opt.testgridConfig, ctx, storageClient)
	if err != nil {
		return fmt.Errorf("read testgrid config: %v", err)
	}

	var rsClient *resultstore.Client
	if opt.project != "" {
		rsClient, err = resultstoreClient(ctx, opt.account, resultstore.Secret(opt.secret))
		if err != nil {
			return fmt.Errorf("resultstore client: %v", err)
		}
	}

	// Should we just transfer a specific build?
	if opt.path.Bucket() != "" { // All valid --build=gs://whatever values have a non-empty bucket.
		return transferBuild(ctx, storageClient, rsClient, opt.project, opt.path, opt.override, true)
	}

	groups, err := findGroups(cfg, opt.jobs.Strings()...)
	if err != nil {
		return fmt.Errorf("find groups: %v", err)
	}

	logrus.Infof("Finding latest builds for %d groups...\n", len(groups))
	buildsChan, buildsErrChan := findBuilds(ctx, storageClient, groups)
	transferErrChan := transfer(ctx, storageClient, rsClient, opt, buildsChan)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-buildsErrChan:
		if err != nil {
			return fmt.Errorf("find builds: %v", err)
		}
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-transferErrChan:
		if err != nil {
			return fmt.Errorf("transfer: %v", err)
		}
	}
	return nil
}

func joinErrs(errs []error, sep string) string {
	var out []string
	for _, e := range errs {
		out = append(out, e.Error())
	}
	return strings.Join(out, sep)
}

func transfer(ctx context.Context, storageClient *storage.Client, rsClient *resultstore.Client, opt options, buildsChan <-chan buildsInfo) <-chan error {
	retChan := make(chan error)
	go func() {
		transferErrChan := make(chan error)
		var wg sync.WaitGroup
		var total int
		for info := range buildsChan {
			total++
			wg.Add(1)
			go func(info buildsInfo) {
				defer wg.Done()
				if err := transferLatest(ctx, storageClient, rsClient, opt.project, info.builds, opt.latest, opt.override, opt.pending); err != nil {
					logrus.WithError(err).Error("Transfer failed")
					select {
					case <-ctx.Done():
					case transferErrChan <- fmt.Errorf("transfer %s in %s: %v", info.name, info.prefix, err):
					}
				}
			}(info)
		}
		go func() {
			defer close(transferErrChan)
			wg.Wait()
		}()
		var errs []error
		for err := range transferErrChan {
			errs = append(errs, err)
		}
		var err error
		if n := len(errs); n > 0 {
			err = fmt.Errorf("%d errors transferring %d groups: %v", n, total, joinErrs(errs, ", "))
		}
		select {
		case <-ctx.Done():
		case retChan <- err:
		}

	}()
	return retChan
}

type bucketChecker struct {
	buckets map[string]bool
	client  *storage.Client
	lock    sync.RWMutex
}

func (bc *bucketChecker) writable(ctx context.Context, path gcs.Path) bool {
	name := path.Bucket()
	bc.lock.RLock()
	writable, present := bc.buckets[name]
	bc.lock.RUnlock()
	if present {
		return writable
	}
	bc.lock.Lock()
	defer bc.lock.Unlock()
	writable, present = bc.buckets[name]
	if present {
		return writable
	}
	const want = "storage.objects.create"
	have, err := bc.client.Bucket(name).IAM().TestPermissions(ctx, []string{want})
	if err != nil || len(have) != 1 || have[0] != want {
		bc.buckets[name] = false
		logrus.WithError(err).WithFields(logrus.Fields{"bucket": name, "want": want, "have": have}).Error("No write access")
	} else {
		bc.buckets[name] = true
	}
	return bc.buckets[name]
}

func findGroups(cfg *config.Configuration, jobs ...string) ([]config.TestGroup, error) {
	var groups []config.TestGroup
	for _, job := range jobs {
		tg := cfg.FindTestGroup(job)
		if tg == nil {
			return nil, fmt.Errorf("job %s not found in test groups", job)
		}
		groups = append(groups, *tg)
	}
	if len(jobs) == 0 {
		for _, tg := range cfg.TestGroups {
			groups = append(groups, *tg)
		}
	}
	return groups, nil
}

type buildsInfo struct {
	name   string
	prefix gcs.Path
	builds []gcs.Build
}

func findGroupBuilds(ctx context.Context, storageClient *storage.Client, bc *bucketChecker, group config.TestGroup, buildsChan chan<- buildsInfo, errChan chan<- error) {
	log := logrus.WithFields(logrus.Fields{
		"testgroup":  group.Name,
		"gcs_prefix": "gs://" + group.GcsPrefix,
	})
	log.Debug("Get latest builds...")
	tgPath, err := gcs.NewPath("gs://" + group.GcsPrefix)
	if err != nil {
		log.WithError(err).Error("Bad build URL")
		err = fmt.Errorf("test group %s: gs://%s prefix invalid: %v", group.Name, group.GcsPrefix, err)
		select {
		case <-ctx.Done():
		case errChan <- err:
		}
		return
	}
	if !bc.writable(ctx, *tgPath) {
		log.Debug("Skip unwritable group")
		return
	}

	builds, err := gcs.ListBuilds(ctx, storageClient, *tgPath)
	if err != nil {
		log.WithError(err).Error("Failed to list builds")
		err := fmt.Errorf("test group %s: list %s: %v", group.Name, *tgPath, err)
		select {
		case <-ctx.Done():
		case errChan <- err:
		}
		return
	}
	info := buildsInfo{
		name:   group.Name,
		prefix: *tgPath,
		builds: builds,
	}
	select {
	case <-ctx.Done():
	case buildsChan <- info:
	}
}

func findBuilds(ctx context.Context, storageClient *storage.Client, groups []config.TestGroup) (<-chan buildsInfo, <-chan error) {
	buildsChan := make(chan buildsInfo)
	errChan := make(chan error)
	bc := bucketChecker{
		buckets: map[string]bool{},
		client:  storageClient,
	}
	go func() {
		innerErrChan := make(chan error)
		defer close(buildsChan)
		defer close(errChan)
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		var wg sync.WaitGroup
		for _, testGroup := range groups {
			wg.Add(1)
			go func(testGroup config.TestGroup) {
				defer wg.Done()
				findGroupBuilds(ctx, storageClient, &bc, testGroup, buildsChan, innerErrChan)
			}(testGroup)
		}
		go func() {
			defer close(innerErrChan)
			wg.Wait()
		}()
		var errs []error
		for err := range innerErrChan {
			errs = append(errs, err)
		}

		var err error
		if n := len(errs); n > 0 {
			err = fmt.Errorf("%d errors finding builds from %d groups: %v", n, len(groups), joinErrs(errs, ", "))
		}

		select {
		case <-ctx.Done():
		case errChan <- err:
		}
	}()
	return buildsChan, errChan
}

func transferLatest(ctx context.Context, storageClient *storage.Client, rsClient *resultstore.Client, project string, builds gcs.Builds, max int, override bool, includePending bool) error {

	for i, build := range builds {
		if i >= max {
			break
		}
		path, err := gcs.NewPath(fmt.Sprintf("gs://%s/%s", build.BucketPath, build.Prefix))
		if err != nil {
			return fmt.Errorf("bad %s path: %v", build, err)
		}
		if err := transferBuild(ctx, storageClient, rsClient, project, *path, override, includePending); err != nil {
			return fmt.Errorf("%s: %v", build, err)
		}
	}
	return nil
}

func transferBuild(ctx context.Context, storageClient *storage.Client, rsClient *resultstore.Client, project string, path gcs.Path, override bool, includePending bool) error {
	build := gcs.Build{
		Bucket:     storageClient.Bucket(path.Bucket()),
		Context:    ctx,
		Prefix:     trailingSlash(path.Object()),
		BucketPath: path.Bucket(),
	}

	log := logrus.WithFields(logrus.Fields{"build": build})

	log.Debug("Downloading...")
	result, err := download(ctx, storageClient, build)
	if err != nil {
		return fmt.Errorf("download: %v", err)
	}

	switch val, _ := result.started.Metadata.String(resultstoreKey); {
	case val != nil && override:
		log = log.WithFields(logrus.Fields{"previously": *val})
		log.Warn("Replacing result...")
	case val != nil:
		log.WithFields(logrus.Fields{
			"resultstore": *val,
		}).Debug("Already transferred")
		return nil
	}

	if (result.started.Pending || result.finished.Running) && !includePending {
		log.Debug("Skip pending result")
		return nil
	}

	desc := "Results of " + path.String()
	log.Debug("Converting...")
	inv, target, test := convert(project, desc, path, *result)

	if project == "" {
		print(inv.To(), test.To())
		return nil
	}

	log.Debug("Uploading...")
	viewURL, err := upload(rsClient, inv, target, test)
	if err != nil {
		return fmt.Errorf("upload %s: %v", viewURL, err)
	}
	log = log.WithFields(logrus.Fields{"resultstore": viewURL})
	log.Info("Transferred result")
	changed, err := insertLink(&result.started, viewURL)
	if err != nil {
		return fmt.Errorf("insert resultstore link into metadata: %v", err)
	}
	if !changed { // already has the link
		return nil
	}
	log.Debug("Inserting link...")
	if err := updateStarted(ctx, storageClient, path, result.started); err != nil {
		return fmt.Errorf("update started.json: %v", err)
	}
	return nil
}

const (
	linksKey       = "links"
	resultstoreKey = "resultstore"
	urlKey         = "url"
)

// insertLink attempts to set metadata.links.resultstore.url to viewURL.
//
// returns true if started metadata was updated.
func insertLink(started *gcs.Started, viewURL string) (bool, error) {
	if started.Metadata == nil {
		started.Metadata = metadata.Metadata{}
	}
	meta := started.Metadata
	var changed bool
	top, present := meta.String(resultstoreKey)
	if !present || top == nil || *top != viewURL {
		changed = true
		meta[resultstoreKey] = viewURL
	}
	links, present := meta.Meta(linksKey)
	if present && links == nil {
		return false, fmt.Errorf("metadata.links is not a Metadata value: %v", meta[linksKey])
	}
	if links == nil {
		links = &metadata.Metadata{}
		changed = true
	}
	resultstoreMeta, present := links.Meta(resultstoreKey)
	if present && resultstoreMeta == nil {
		return false, fmt.Errorf("metadata.links.resultstore is not a Metadata value: %v", (*links)[resultstoreKey])
	}
	if resultstoreMeta == nil {
		resultstoreMeta = &metadata.Metadata{}
		changed = true
	}
	val, present := resultstoreMeta.String(urlKey)
	if present && val == nil {
		return false, fmt.Errorf("metadata.links.resultstore.url is not a string value: %v", (*resultstoreMeta)[urlKey])
	}
	if !changed && val != nil && *val == viewURL {
		return false, nil
	}

	(*resultstoreMeta)[urlKey] = viewURL
	(*links)[resultstoreKey] = *resultstoreMeta
	meta[linksKey] = *links
	return true, nil
}

func updateStarted(ctx context.Context, storageClient *storage.Client, path gcs.Path, started gcs.Started) error {
	startedPath, err := path.ResolveReference(&url.URL{Path: "started.json"})
	if err != nil {
		return fmt.Errorf("resolve started.json: %v", err)
	}
	buf, err := json.Marshal(started)
	if err != nil {
		return fmt.Errorf("encode started.json: %v", err)
	}
	// TODO(fejta): compare and swap
	if err := gcs.Upload(ctx, storageClient, *startedPath, buf, gcs.Default); err != nil {
		return fmt.Errorf("upload started.json: %v", err)
	}
	return nil
}
