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
package fakestorage

import (
	"context"
	"sync"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/google/mako/spec/go/mako"
	pgpb "github.com/google/mako/spec/proto/mako_go_proto"
)

func TestCreateThenLookup(t *testing.T) {
	s := New()

	ctx := context.Background()
	description := "cubbies win!"
	ri := &pgpb.RunInfo{BenchmarkKey: proto.String("123"), TimestampMs: proto.Float64(1), Description: proto.String(description)}
	resp, err := s.CreateRunInfo(ctx, ri)
	if err != nil {
		t.Fatalf("CreateRunInfo() err: %v", err)
	}
	key := resp.GetKey()
	riq := &pgpb.RunInfoQuery{RunKey: proto.String(key)}
	result, err := s.QueryRunInfo(ctx, riq)
	if err != nil {
		t.Fatalf("QueryRunInfo() err: %v", err)
	}
	if l := len(result.GetRunInfoList()); l != 1 {
		t.Fatalf("QueryRunInfo() results len %d; want %d", l, 1)
	}
	actualRi := result.GetRunInfoList()[0]
	if k := actualRi.GetRunKey(); k != key {
		t.Errorf("RunInfo.run_key %q; want %q", k, key)
	}
	if d := actualRi.GetDescription(); d != description {
		t.Errorf("RunInfo.run_key %q; want %q", d, description)
	}
}

func TestCount(t *testing.T) {
	s := New()
	s.FakeClear()

	ctx := context.Background()
	createb := func(proj string) {
		ivi := &pgpb.ValueInfo{ValueKey: proto.String("t"), Label: proto.String("time")}
		bi := &pgpb.BenchmarkInfo{ProjectName: proto.String(proj), BenchmarkName: proto.String("Leia"), OwnerList: []string{"leia@alderaan.gov"}, InputValueInfo: ivi}
		_, err := s.CreateBenchmarkInfo(ctx, bi)
		if err != nil {
			t.Fatalf("CreateBenchmarkInfo failure. Benchmark was %v. Err was %v", bi, err)
		}
	}
	creater := func(tag string) {
		ri := &pgpb.RunInfo{Tags: []string{tag}, BenchmarkKey: proto.String("key"), TimestampMs: proto.Float64(0)}
		_, err := s.CreateRunInfo(ctx, ri)
		if err != nil {
			t.Fatalf("CreateRunInfo failure. Run was %v. Err was %v", ri, err)
		}
	}
	assertb := func(proj string, want int64) {
		biq := &pgpb.BenchmarkInfoQuery{ProjectName: proto.String(proj)}
		cr, err := s.CountBenchmarkInfo(ctx, biq)
		if err != nil {
			t.Fatalf("CountBenchmarkInfo failure. Query was %v", biq)
		}
		if cr.GetCount() != want {
			t.Errorf("Counted %v benchmarks with project=%v, expected %v", cr.GetCount(), proj, want)
		}
	}
	assertr := func(tag string, want int64) {
		riq := &pgpb.RunInfoQuery{Tags: []string{tag}}
		cr, err := s.CountRunInfo(ctx, riq)
		if err != nil {
			t.Fatalf("CountRunInfo failure. Query was %v", riq)
		}
		if cr.GetCount() != want {
			t.Errorf("Counted %v runs with tag=%v, expected %v", cr.GetCount(), tag, want)
		}
	}

	assertb("p1", 0)

	createb("p1")
	assertb("p1", 1)
	assertb("p2", 0)

	createb("p2")
	createb("p2")
	assertb("p1", 1)
	assertb("p2", 2)

	assertr("r1", 0)

	creater("r1")
	assertr("r1", 1)
	assertr("r2", 0)

	creater("r2")
	creater("r2")
	assertr("r1", 1)
	assertr("r2", 2)
}

func TestMultipleRoutinesSeeSameSWIGState(t *testing.T) {
	var wg sync.WaitGroup
	runKey := ""

	// Create a RunInfo
	wg.Add(1)
	go func() {
		defer wg.Done()
		s := New()
		ctx := context.Background()
		ri := &pgpb.RunInfo{BenchmarkKey: proto.String("123"), TimestampMs: proto.Float64(1)}
		resp, err := s.CreateRunInfo(ctx, ri)
		if err != nil {
			t.Fatalf("CreateRunInfo() err: %v", err)
		}
		runKey = resp.GetKey()
	}()
	wg.Wait()

	// Verify key has been set
	if runKey == "" {
		t.Fatal("expected run_key to be set")
	}

	// Launch another go routine that reads this RunInfo
	wg.Add(1)
	go func() {
		defer wg.Done()
		s := New()
		ctx := context.Background()
		riq := &pgpb.RunInfoQuery{RunKey: proto.String(runKey)}
		result, err := s.QueryRunInfo(ctx, riq)
		if err != nil {
			t.Fatalf("QueryRunInfo() err: %v", err)
		}
		if l := len(result.GetRunInfoList()); l != 1 {
			t.Fatalf("QueryRunInfo() results len %d; want %d", l, 1)
		}
		actualRi := result.GetRunInfoList()[0]
		if k := actualRi.GetRunKey(); k != runKey {
			t.Errorf("RunInfo.run_key %q; want %q", k, runKey)
		}
	}()
	wg.Wait()
}

func TestStorageLimits(t *testing.T) {
	metricValueCountMax := 1
	batchSizeMax := 2
	s := NewWithLimits(metricValueCountMax, 100, batchSizeMax, 100, 100, 100)

	ctx := context.Background()
	a, err := s.GetMetricValueCountMax(ctx)
	if err != nil {
		t.Errorf("GetMetricValueCountMax() err: %v", err)
	}
	if a != metricValueCountMax {
		t.Errorf("GetMetricValueCountMax() got %d; want %d", a, metricValueCountMax)
	}
	a, err = s.GetBatchSizeMax(ctx)
	if err != nil {
		t.Errorf("GetBatchSizeMax() err: %v", err)
	}
	if a != batchSizeMax {
		t.Errorf("GetBatchSizeMax() got %d; want %d", a, batchSizeMax)
	}
}

func TestLotsOfRoutines(t *testing.T) {
	var wg sync.WaitGroup
	numberOfRoutines := 100
	wg.Add(numberOfRoutines)
	s := New()
	for i := 0; i < numberOfRoutines; i++ {
		go func() {
			defer wg.Done()
			ctx := context.Background()
			ri := &pgpb.RunInfo{BenchmarkKey: proto.String("123"), TimestampMs: proto.Float64(1)}
			_, err := s.CreateRunInfo(ctx, ri)
			if err != nil {
				t.Fatalf("CreateRunInfo() err: %v", err)
			}
		}()
	}
	wg.Wait()
}

func TestFakeClear(t *testing.T) {
	s := New()

	ctx := context.Background()
	description := "cubbies win!"
	ri := &pgpb.RunInfo{BenchmarkKey: proto.String("123"), TimestampMs: proto.Float64(1), Description: proto.String(description)}
	resp, err := s.CreateRunInfo(ctx, ri)
	if err != nil {
		t.Fatalf("CreateRunInfo() err: %v", err)
	}
	key := resp.GetKey()

	// Clear data, should fail to lookup.
	s.FakeClear()

	riq := &pgpb.RunInfoQuery{RunKey: proto.String(key)}
	result, err := s.QueryRunInfo(ctx, riq)
	if err != nil {
		t.Fatalf("QueryRunInfo() err: %v", err)
	}
	if l := len(result.GetRunInfoList()); l != 0 {
		t.Fatalf("QueryRunInfo() results len %d; want %d", l, 0)
	}
}

func TestCastAsStorage(t *testing.T) {
	var _ mako.Storage = (*FakeStorage)(nil)
}

func TestQueriesWithLimitZero(t *testing.T) {
	s := New()

	want := 1

	ctx := context.Background()

	ivi := &pgpb.ValueInfo{ValueKey: proto.String("t"), Label: proto.String("time")}
	bi := &pgpb.BenchmarkInfo{ProjectName: proto.String("test"), BenchmarkName: proto.String("Leia"), OwnerList: []string{"leia@alderaan.gov"}, InputValueInfo: ivi}

	resp, err := s.CreateBenchmarkInfo(ctx, bi)
	if err != nil {
		t.Fatalf("CreateBenchmarkInfo() err: %v", err)
	}
	bk := resp.GetKey()
	biq := &pgpb.BenchmarkInfoQuery{BenchmarkKey: proto.String(bk), Limit: proto.Int32(0)}
	biqr, err := s.QueryBenchmarkInfo(ctx, biq)
	if err != nil {
		t.Fatalf("QueryBenchmarkInfo failure: %v. Query was %v", err, biq)
	}
	if len(biqr.BenchmarkInfoList) != want {
		t.Errorf("Query returned %v benchmarks, expected %v", len(biqr.BenchmarkInfoList), want)
	}

	ri := &pgpb.RunInfo{BenchmarkKey: proto.String(bk), TimestampMs: proto.Float64(1)}
	_, err = s.CreateRunInfo(ctx, ri)
	if err != nil {
		t.Fatalf("CreateRunInfo() err: %v", err)
	}
	riq := &pgpb.RunInfoQuery{BenchmarkKey: proto.String(bk), Limit: proto.Int32(0)}
	riqr, err := s.QueryRunInfo(ctx, riq)
	if err != nil {
		t.Fatalf("QueryRunInfo failure: %v. Query was %v", err, riq)
	}
	if len(riqr.RunInfoList) != want {
		t.Errorf("Query returned %v runs, expected %v.", len(riqr.RunInfoList), want)
	}

	kv := &pgpb.KeyedValue{ValueKey: proto.String("v"), Value: proto.Float64(1.0)}
	p := &pgpb.SamplePoint{InputValue: proto.Float64(1.0), MetricValueList: []*pgpb.KeyedValue{kv}}
	sb := &pgpb.SampleBatch{BenchmarkKey: proto.String(bk), RunKey: proto.String("rk"), SamplePointList: []*pgpb.SamplePoint{p}}
	_, err = s.CreateSampleBatch(ctx, sb)
	if err != nil {
		t.Fatalf("CreateSampleBatch() err: %v", err)
	}
	sbq := &pgpb.SampleBatchQuery{BenchmarkKey: proto.String(bk), Limit: proto.Int32(0)}
	sbqr, err := s.QuerySampleBatch(ctx, sbq)
	if err != nil {
		t.Fatalf("QuerySampleBatch failure: %v. Query was %v", err, sbq)
	}
	if len(sbqr.SampleBatchList) != want {
		t.Errorf("Query returned %v batches, expected %v.", len(sbqr.SampleBatchList), want)
	}

}

func TestGetHostname(t *testing.T) {
	s := New()
	want := "example.com"
	if got := s.GetHostname(context.Background()); got != want {
		t.Errorf("storage.GetHostname() got %v; want %v", got, want)
	}
}

// testing for go/mako-go-swig-finalizer-problem
func TestCreateBenchWrapperLifetime(t *testing.T) {
	s := New()
	ivi := &pgpb.ValueInfo{ValueKey: proto.String("t"), Label: proto.String("time")}
	bi := &pgpb.BenchmarkInfo{ProjectName: proto.String("p"), BenchmarkName: proto.String("Leia"), OwnerList: []string{"leia@alderaan.gov"}, InputValueInfo: ivi}
	_, err := s.CreateBenchmarkInfo(context.Background(), bi)
	if err != nil {
		t.Fatal("CreateBenchmarkInfo failed")
	}
}

// testing for go/mako-go-swig-finalizer-problem
func TestUpdateBenchWrapperLifetime(t *testing.T) {
	s := New()
	ivi := &pgpb.ValueInfo{ValueKey: proto.String("t"), Label: proto.String("time")}
	bi := &pgpb.BenchmarkInfo{ProjectName: proto.String("p"), BenchmarkName: proto.String("Leia"), OwnerList: []string{"leia@alderaan.gov"}, InputValueInfo: ivi}
	cr, err := s.CreateBenchmarkInfo(context.Background(), bi)
	if err != nil {
		t.Fatal("CreateBenchmarkInfo failed")
	}
	bi.BenchmarkKey = cr.Key
	_, err = s.UpdateBenchmarkInfo(context.Background(), bi)
	if err != nil {
		t.Fatal("UpdateBenchmarkInfo failed")
	}
}

// testing for go/mako-go-swig-finalizer-problem
func TestQueryBenchWrapperLifetime(t *testing.T) {
	s := New()
	biq := &pgpb.BenchmarkInfoQuery{ProjectName: proto.String("p")}
	_, err := s.QueryBenchmarkInfo(context.Background(), biq)
	if err != nil {
		t.Fatal("QueryBenchmarkInfo failed")
	}
}

// testing for go/mako-go-swig-finalizer-problem
func TestDeleteBenchWrapperLifetime(t *testing.T) {
	s := New()
	biq := &pgpb.BenchmarkInfoQuery{ProjectName: proto.String("p")}
	_, err := s.DeleteBenchmarkInfo(context.Background(), biq)
	if err != nil {
		t.Fatal("DeleteBenchmarkInfo failed")
	}
}

// testing for go/mako-go-swig-finalizer-problem
func TestCountBenchWrapperLifetime(t *testing.T) {
	s := New()
	biq := &pgpb.BenchmarkInfoQuery{ProjectName: proto.String("p")}
	_, err := s.CountBenchmarkInfo(context.Background(), biq)
	if err != nil {
		t.Fatal("CountBenchmarkInfo failed")
	}
}

// testing for go/mako-go-swig-finalizer-problem
func TestCreateRunWrapperLifetime(t *testing.T) {
	s := New()
	ri := &pgpb.RunInfo{BenchmarkKey: proto.String("xxxx"), TimestampMs: proto.Float64(0)}
	_, err := s.CreateRunInfo(context.Background(), ri)
	if err != nil {
		t.Fatal("CreateRunInfo failed")
	}
}

// testing for go/mako-go-swig-finalizer-problem
func TestUpdateRunWrapperLifetime(t *testing.T) {
	s := New()
	ri := &pgpb.RunInfo{BenchmarkKey: proto.String("xxxx"), TimestampMs: proto.Float64(0)}
	cr, err := s.CreateRunInfo(context.Background(), ri)
	if err != nil {
		t.Fatal("CreateRunInfo failed")
	}
	ri.RunKey = cr.Key
	_, err = s.UpdateRunInfo(context.Background(), ri)
	if err != nil {
		t.Fatal("UpdateRunInfo failed")
	}
}

// testing for go/mako-go-swig-finalizer-problem
func TestQueryRunWrapperLifetime(t *testing.T) {
	s := New()
	riq := &pgpb.RunInfoQuery{RunKey: proto.String("k")}
	_, err := s.QueryRunInfo(context.Background(), riq)
	if err != nil {
		t.Fatal("QueryRunInfo failed")
	}
}

// testing for go/mako-go-swig-finalizer-problem
func TestDeleteRunWrapperLifetime(t *testing.T) {
	s := New()
	riq := &pgpb.RunInfoQuery{RunKey: proto.String("k")}
	_, err := s.DeleteRunInfo(context.Background(), riq)
	if err != nil {
		t.Fatal("DeleteRunInfo failed")
	}
}

// testing for go/mako-go-swig-finalizer-problem
func TestCountRunWrapperLifetime(t *testing.T) {
	s := New()
	riq := &pgpb.RunInfoQuery{RunKey: proto.String("k")}
	_, err := s.CountRunInfo(context.Background(), riq)
	if err != nil {
		t.Fatal("CountRunInfo failed")
	}
}

// testing for go/mako-go-swig-finalizer-problem
func TestCreateBatchWrapperLifetime(t *testing.T) {
	s := New()
	kv := &pgpb.KeyedValue{ValueKey: proto.String("v"), Value: proto.Float64(1.0)}
	p := &pgpb.SamplePoint{InputValue: proto.Float64(1.0), MetricValueList: []*pgpb.KeyedValue{kv}}
	sb := &pgpb.SampleBatch{BenchmarkKey: proto.String("xxxx"), RunKey: proto.String("xxxx"), SamplePointList: []*pgpb.SamplePoint{p}}
	_, err := s.CreateSampleBatch(context.Background(), sb)
	if err != nil {
		t.Fatal("CreateSampleBatch failed")
	}
}

// testing for go/mako-go-swig-finalizer-problem
func TestQueryBatchWrapperLifetime(t *testing.T) {
	s := New()
	sbq := &pgpb.SampleBatchQuery{BatchKey: proto.String("k")}
	_, err := s.QuerySampleBatch(context.Background(), sbq)
	if err != nil {
		t.Fatal("QuerySampleBatch failed")
	}
}

// testing for go/mako-go-swig-finalizer-problem
func TestDeleteBatchWrapperLifetime(t *testing.T) {
	s := New()
	sbq := &pgpb.SampleBatchQuery{BatchKey: proto.String("k")}
	_, err := s.DeleteSampleBatch(context.Background(), sbq)
	if err != nil {
		t.Fatal("DeleteSampleBatch failed")
	}
}

// testing for go/mako-go-swig-finalizer-problem
func TestGetMetricCountWrapperLifetime(t *testing.T) {
	s := New()
	_, err := s.GetMetricValueCountMax(context.Background())
	if err != nil {
		t.Fatal("GetMetricValueCountMax failed")
	}
}

// testing for go/mako-go-swig-finalizer-problem
func TestGetErrorCountWrapperLifetime(t *testing.T) {
	s := New()
	_, err := s.GetSampleErrorCountMax(context.Background())
	if err != nil {
		t.Fatal("GetSampleErrorCountMax failed")
	}
}

// testing for go/mako-go-swig-finalizer-problem
func TestBatchSizeMaxWrapperLifetime(t *testing.T) {
	s := New()
	_, err := s.GetBatchSizeMax(context.Background())
	if err != nil {
		t.Fatal("GetBatchSizeMax failed")
	}
}

// testing for go/mako-go-swig-finalizer-problem
func TestGetSwigWrapWrapperLifetime(t *testing.T) {
	s := New()
	_ = s.GetSwigWrap()
}

// testing for go/mako-go-swig-finalizer-problem
func TestFakeClearWrapWrapperLifetime(t *testing.T) {
	s := New()
	s.FakeClear()
}

// testing for go/mako-go-swig-finalizer-problem
func TestGetHostnameWrapWrapperLifetime(t *testing.T) {
	s := New()
	_ = s.GetHostname(context.Background())
}
