/*
Copyright 2019 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/golang/protobuf/jsonpb"
	mako "github.com/google/mako/spec/proto/mako_go_proto"

	"log"
	"net"
	"os"
	"sync"

	"google.golang.org/grpc"

	"knative.dev/pkg/test/mako/config"
	qspb "knative.dev/pkg/third_party/mako/proto/quickstore_go_proto"
)

const (
	port = ":9813"
	// A 10 minutes run at 1000 rps of eventing perf tests is usually ~= 70 MBi, so 100MBi is reasonable
	defaultServerMaxReceiveMessageSize = 1024 * 1024 * 100
)

type server struct {
	info     *mako.BenchmarkInfo
	stopOnce sync.Once
	stopCh   chan struct{}
}

func (s *server) Store(ctx context.Context, in *qspb.StoreInput) (*qspb.StoreOutput, error) {
	m := jsonpb.Marshaler{}
	qi, _ := m.MarshalToString(in.GetQuickstoreInput())
	fmt.Printf("# %s\n", qi)
	writer := csv.NewWriter(os.Stdout)

	kv := calculateKeyIndexColumnsMap(s.info)
	cols := make([]string, len(kv))
	for k, i := range kv {
		cols[i] = k
	}
	fmt.Printf("# %s\n", strings.Join(cols, ","))

	for _, sp := range in.GetSamplePoints() {
		for _, mv := range sp.GetMetricValueList() {
			vals := map[string]string{"inputValue": fmt.Sprintf("%f", sp.GetInputValue())}
			vals[mv.GetValueKey()] = fmt.Sprintf("%f", mv.GetValue())
			writer.Write(makeRow(vals, kv))
		}
	}

	for _, ra := range in.GetRunAggregates() {
		vals := map[string]string{ra.GetValueKey(): fmt.Sprintf("%f", ra.GetValue())}
		writer.Write(makeRow(vals, kv))
	}

	for _, sa := range in.GetSampleErrors() {
		vals := map[string]string{"inputValue": fmt.Sprintf("%f", sa.GetInputValue()), "errorMessage": sa.GetErrorMessage()}
		writer.Write(makeRow(vals, kv))
	}

	writer.Flush()

	fmt.Printf("# CSV end\n")

	return &qspb.StoreOutput{}, nil
}

func makeRow(points map[string]string, kv map[string]int) []string {
	row := make([]string, len(kv))
	for k, v := range points {
		row[kv[k]] = v
	}
	return row
}

func calculateKeyIndexColumnsMap(info *mako.BenchmarkInfo) map[string]int {
	kv := make(map[string]int)
	kv["inputValue"] = 0
	kv["errorMessage"] = 1
	for i, m := range info.MetricInfoList {
		kv[*m.ValueKey] = i + 2
	}
	return kv
}

func (s *server) ShutdownMicroservice(ctx context.Context, in *qspb.ShutdownInput) (*qspb.ShutdownOutput, error) {
	s.stopOnce.Do(func() { close(s.stopCh) })
	return &qspb.ShutdownOutput{}, nil
}

var waitTimeBeforeEnd int

func init() {
	flag.IntVar(&waitTimeBeforeEnd, "w", 0, "Wait time in seconds before tear down")
}

func main() {
	flag.Parse()

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer(grpc.MaxRecvMsgSize(defaultServerMaxReceiveMessageSize))
	stopCh := make(chan struct{})
	info := config.MustGetBenchmark()
	fmt.Printf("# Benchmark %s - %s/n", *info.BenchmarkKey, *info.BenchmarkName)
	go func() {
		qspb.RegisterQuickstoreServer(s, &server{info: info, stopCh: stopCh})
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()
	<-stopCh
	s.GracefulStop()

	time.Sleep(time.Second * time.Duration(waitTimeBeforeEnd))
}
