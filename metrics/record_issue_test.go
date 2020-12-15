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

// An attempt to reproduce knative/eventing#4645

package metrics

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"go.opencensus.io/resource"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"

	logtesting "knative.dev/pkg/logging/testing"
	"knative.dev/pkg/metrics/metricskey"
)

const concurrentTestRecorders = 8

func TestRecordConcurrentWithResource(t *testing.T) {
	registerTestView(t, []tag.Key{
		keyEventType,
		keyRespCode,
		keyRespCodeClass,
		keyContName,
		keyUniqName,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 9*time.Minute)
	defer cancel()

	MemStatsOrDie(ctx)

	ctx = metricskey.WithResource(ctx, resource.Resource{
		Type: "knative_broker",
		Labels: map[string]string{
			"namespace_name": "my-ns",
			"broker_name":    "default",
		},
	})

	ctx, err := tag.New(
		ctx,
		tag.Insert(keyEventType, "dev.knative.sources.ping"),
		tag.Insert(keyRespCode, "202"),
		tag.Insert(keyRespCodeClass, "2xx"),
		tag.Insert(keyContName, "ingress"),
		tag.Insert(keyUniqName, "mt-broker-ingress-67ff9c869b-n88c8e5a848d2c97c277c43d81512e6cdf"),
	)
	if err != nil {
		t.Fatal("Error creating OpenCensus tags:", err)
	}

	initTestExporter(t)

	var wg sync.WaitGroup

	// scrape metrics regularly to invoke the gatherer and trigger a metric
	// consistency check, which we expect to fail eventually
	tick := time.Tick(15 * time.Second)
	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			select {
			case <-ctx.Done():
				return

			case <-tick:
				fmt.Println("\nScraping metrics")
				resp, err := http.Get("http://127.0.0.1:9090/metrics")
				if err != nil {
					t.Error("Error scraping metrics:", err)
				}
				resp.Body.Close()
			}
		}
	}()

	for i := 0; i < concurrentTestRecorders; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return

				default:
					Record(ctx, eventCountM.M(1))
					Record(ctx, eventCountM.M(1))
					Record(ctx, eventCountM.M(1))
				}
			}
		}()
	}

	wg.Wait()
}

var (
	keyEventType     = tag.MustNewKey("event_type")
	keyRespCode      = tag.MustNewKey("response_code")
	keyRespCodeClass = tag.MustNewKey("response_code_class")
	keyContName      = tag.MustNewKey("container_name")
	keyUniqName      = tag.MustNewKey("unique_name")
)

var eventCountM = stats.Int64(
	"event_count",
	"Fake number of events dispatched by the test component",
	stats.UnitDimensionless,
)

func registerTestView(t *testing.T, tk []tag.Key) {
	err := RegisterResourceView(
		&view.View{
			Measure:     eventCountM,
			Description: eventCountM.Description(),
			Aggregation: view.Count(),
			TagKeys:     tk,
		},
	)
	if err != nil {
		t.Fatal("Error registering OpenCensus stats view:", err)
	}
}

func initTestExporter(t *testing.T) {
	eo := ExporterOptions{
		Domain:    "metrics-test.example.com",
		Component: "metrics_test",
		ConfigMap: map[string]string{
			BackendDestinationKey: string(prometheus),
		},
	}

	logger := logtesting.TestLogger(t)

	if err := UpdateExporter(context.Background(), eo, logger); err != nil {
		t.Fatal("Error updating metrics exporter:", err)
	}
}
