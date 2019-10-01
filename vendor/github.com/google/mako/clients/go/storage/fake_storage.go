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
Package fakestorage is a fake (or in-memory) version of Mako storage.

This is useful for unit/integration testing code that depends on storage.

This fake does not support queries involving cursors concurrent with
updates to any data of the same type.

This fake does not implement all errors that may occur
when calling a real server. It attempts to succeed on all calls
and only fails when it cannot satisfy the request.

All functions that are not part of the spec are prefixed with 'Fake',
and these functions can be used as test helper functions.

For more information about mako see go/mako.

For more information about the interface see https://github.com/google/mako/blob/master/spec/go/storage.go

This package defers work to SWIG wrapping of the fake C++ Mako client.

NOTE: Because of SWIG, although a Context is passed into each of the functions
below it is ignored. For this implementation of Mako Storage, calls cannot
be cancelled via Context.

Functions and methods in this package are safe to call from multiple goroutines
concurrently.
*/
package fakestorage

import (
	"fmt"
	"runtime"

	"github.com/golang/protobuf/proto"
	wrap "github.com/google/mako/clients/cxx/storage/go/fakestorage_wrap"
	"github.com/google/mako/internal/go/wrappedstorage"
	pgpb "github.com/google/mako/spec/proto/mako_go_proto"
)

/*
FakeStorage provides access mako storage via the mako Storage
interface.

More information about the interface can be found at go/mako and
https://github.com/google/mako/blob/master/spec/go/storage.go

The zero value of this struct is not usable, please use New.*
functions below.

NOTE: This struct is public to allow 'Fake' calls which are not part of Storage
interface.
*/
type FakeStorage struct {
	*wrappedstorage.Simple
	wrapper wrap.Storage
}

// New returns a ready to use instance.
func New() *FakeStorage {
	w := wrap.NewStorage()
	return &FakeStorage{wrappedstorage.NewSimple(w, func() { wrap.DeleteStorage(w) }), w}
}

// NewWithLimits returns a ready to use instance with specified limits.
func NewWithLimits(metricValueCountMax, errorCountMax, batchSizeMax, benchLimitMax, runLimitMax, batchLimitMax int) *FakeStorage {
	w := wrap.NewStorage(metricValueCountMax, errorCountMax, batchSizeMax, benchLimitMax, runLimitMax, batchLimitMax)
	return &FakeStorage{wrappedstorage.NewSimple(w, func() { wrap.DeleteStorage(w) }), w}
}

// FakeClear clears the storage system of all known data.
func (s *FakeStorage) FakeClear() {
	s.wrapper.FakeClear()
	// go/mako-go-swig-finalizer-problem
	runtime.KeepAlive(s)
}

// FakeStageBenchmarks inserts the benchmarks in storage.
func (s *FakeStorage) FakeStageBenchmarks(benchmarks []*pgpb.BenchmarkInfo) error {
	benchmarksBytes := make([][]byte, len(benchmarks))
	for i, benchmark := range benchmarks {
		t, err := proto.Marshal(benchmark)
		if err != nil {
			return fmt.Errorf("could not marshal BenchmarkInfo (%v): %v", benchmark, err)
		}
		benchmarksBytes[i] = t
	}

	s.wrapper.FakeStageBenchmarks(benchmarksBytes)
	// go/mako-go-swig-finalizer-problem
	runtime.KeepAlive(s)
	return nil
}

// FakeStageRuns inserts the runs in storage.
func (s *FakeStorage) FakeStageRuns(runs []*pgpb.RunInfo) error {
	runsBytes := make([][]byte, len(runs))
	for i, run := range runs {
		t, err := proto.Marshal(run)
		if err != nil {
			return fmt.Errorf("could not marshal RunInfo (%v): %v", run, err)
		}
		runsBytes[i] = t
	}

	s.wrapper.FakeStageRuns(runsBytes)
	// go/mako-go-swig-finalizer-problem
	runtime.KeepAlive(s)
	return nil
}

// FakeStageBatches inserts the batches in storage.
func (s *FakeStorage) FakeStageBatches(batches []*pgpb.SampleBatch) error {
	batchesBytes := make([][]byte, len(batches))
	for i, batch := range batches {
		t, err := proto.Marshal(batch)
		if err != nil {
			return fmt.Errorf("could not marshal SampleBatch (%v): %v", batch, err)
		}
		batchesBytes[i] = t
	}

	s.wrapper.FakeStageBatches(batchesBytes)
	// go/mako-go-swig-finalizer-problem
	runtime.KeepAlive(s)
	return nil
}
