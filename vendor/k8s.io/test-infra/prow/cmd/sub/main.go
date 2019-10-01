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

package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	prowapi "k8s.io/test-infra/prow/apis/prowjobs/v1"
	prowv1 "k8s.io/test-infra/prow/client/clientset/versioned/typed/prowjobs/v1"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/config/secret"
	"k8s.io/test-infra/prow/flagutil"
	"k8s.io/test-infra/prow/logrusutil"
	"k8s.io/test-infra/prow/metrics"
	"k8s.io/test-infra/prow/pubsub/reporter"
	"k8s.io/test-infra/prow/pubsub/subscriber"
)

var (
	flagOptions *options
)

type options struct {
	client         flagutil.ExperimentalKubernetesOptions
	port           int
	pushSecretFile string

	configPath    string
	jobConfigPath string
	pluginConfig  string

	dryRun      bool
	gracePeriod time.Duration
}

type kubeClient struct {
	client prowv1.ProwJobInterface
	dryRun bool
}

func (c *kubeClient) Create(job *prowapi.ProwJob) (*prowapi.ProwJob, error) {
	if c.dryRun {
		return job, nil
	}
	return c.client.Create(job)
}

func init() {
	flagOptions = &options{}
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	fs.IntVar(&flagOptions.port, "port", 80, "HTTP Port.")
	fs.StringVar(&flagOptions.pushSecretFile, "push-secret-file", "", "Path to Pub/Sub Push secret file.")

	fs.StringVar(&flagOptions.configPath, "config-path", "/etc/config/config.yaml", "Path to config.yaml.")
	fs.StringVar(&flagOptions.jobConfigPath, "job-config-path", "", "Path to prow job configs.")

	fs.BoolVar(&flagOptions.dryRun, "dry-run", true, "Dry run for testing. Uses API tokens but does not mutate.")
	fs.DurationVar(&flagOptions.gracePeriod, "grace-period", 180*time.Second, "On shutdown, try to handle remaining events for the specified duration. ")

	flagOptions.client.AddFlags(fs)

	fs.Parse(os.Args[1:])
}

func main() {
	logrusutil.ComponentInit("pubsub-subscriber")

	configAgent := &config.Agent{}
	if err := configAgent.Start(flagOptions.configPath, flagOptions.jobConfigPath); err != nil {
		logrus.WithError(err).Fatal("Error starting config agent.")
	}

	var tokenGenerator func() []byte
	if flagOptions.pushSecretFile != "" {
		var tokens []string
		tokens = append(tokens, flagOptions.pushSecretFile)

		secretAgent := &secret.Agent{}
		if err := secretAgent.Start(tokens); err != nil {
			logrus.WithError(err).Fatal("Error starting secrets agent.")
		}
		tokenGenerator = secretAgent.GetTokenGenerator(flagOptions.pushSecretFile)
	}

	prowjobClient, err := flagOptions.client.ProwJobClient(configAgent.Config().ProwJobNamespace, flagOptions.dryRun)
	if err != nil {
		logrus.WithError(err).Fatal("unable to create prow job client")
	}
	kubeClient := &kubeClient{
		client: prowjobClient,
		dryRun: flagOptions.dryRun,
	}

	promMetrics := subscriber.NewMetrics()

	// Expose prometheus metrics
	metrics.ExposeMetrics("sub", configAgent.Config().PushGateway)

	s := &subscriber.Subscriber{
		ConfigAgent:   configAgent,
		Metrics:       promMetrics,
		ProwJobClient: kubeClient,
		Reporter:      reporter.NewReporter(configAgent.Config),
	}

	// Return 200 on / for health checks.
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {})

	// Will call shutdown which will stop the errGroup
	shutdownCtx, shutdown := context.WithCancel(context.Background())
	errGroup, derivedCtx := errgroup.WithContext(shutdownCtx)
	wg := sync.WaitGroup{}

	// Setting up Push Server
	logrus.Info("Setting up Push Server")
	pushServer := &subscriber.PushServer{
		Subscriber:     s,
		TokenGenerator: tokenGenerator,
	}
	http.Handle("/push", pushServer)

	// Setting up Pull Server
	logrus.Info("Setting up Pull Server")
	pullServer := subscriber.NewPullServer(s)
	errGroup.Go(func() error {
		wg.Add(1)
		defer wg.Done()
		logrus.Info("Starting Pull Server")
		err := pullServer.Run(derivedCtx)
		logrus.WithError(err).Warn("Pull Server exited.")
		return err
	})

	httpServer := &http.Server{Addr: ":" + strconv.Itoa(flagOptions.port)}
	errGroup.Go(func() error {
		wg.Add(1)
		defer wg.Done()
		logrus.Info("Starting HTTP Server")
		err := httpServer.ListenAndServe()
		logrus.WithError(err).Warn("HTTP Server exited.")
		return err
	})

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM, syscall.SIGABRT)

	select {
	case <-shutdownCtx.Done():
		err = shutdownCtx.Err()
		break
	case <-derivedCtx.Done():
		err = derivedCtx.Err()
		break
	case <-sig:
		break
	}

	logrus.WithError(err).Warn("Starting Shutdown")
	shutdown()
	// Shutdown gracefully on SIGTERM or SIGINT
	timeoutCtx, cancel := context.WithTimeout(context.Background(), flagOptions.gracePeriod)
	defer cancel()
	httpServer.Shutdown(timeoutCtx)
	errGroup.Wait()
	wg.Wait()
}
