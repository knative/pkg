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

// Package wrappedstorage contains boilerplate code to allow easy SWIG-wrapping of C++ storage
// clients.
//
// It also contains wrappedstorage.Interface, which should be implemented by any SWIG library
// providing a Go API to a SWIG storage wrapper. Implementing this interface allows SWIGged
// storage consumers to "reach in" and pull out the wrapper, in order to pass it into their C++
// storage-consuming components.
package wrappedstorage

import (
	"context"
	"errors"
	"fmt"
	"runtime"

	pgpb "github.com/google/mako/spec/proto/mako_go_proto"
)

// Storage interface for a storage lib that provides a Go-native API to a SWIGged C++ storage
// implementation. A SWIGged Storage consumer can use GetSwigWrap() to get the lib's
// underlying SWIG wrapper, which can be passed to the C++ Storage consumer. Storage
// calls from this Storage consumer will then completely avoid the SWIG boundary and
// associated marshalling/unmarshalling.
//
// Take care when making use of this behavior to early cleanup of the C++ resources
// while they're still being used. See go/mako-swig-to-swig.
//
// It might be expected that this interface should embed the storage.Storage interface
// (in otherwords, that methods like CreateBenchmarkInfo should be defined), but this
// is not necessarily the case. The Go-side API could be completely different -- the
// important part is that there is an underlying C++ mako::Storage pointer.
type Storage interface {
	GetSwigWrap() SwigWrap
}

// SwigWrap is an interface for a SWIG wrapper. There is no way to statically
// determine that a struct's Swigcptr() returns a C Storage pointer (versus the
// pointer to some different type). Instead we rely on a runtime panic if the returned
// C pointer points to something else.
type SwigWrap interface {
	Swigcptr() uintptr
}

// Simple is the most basic and straight-forward WrappedStorage implementation. Given
// an object satisfying simpleSwigWrap (in other words, given a simple wrapping of a
// mako::Storage*), it forwards all storage calls to it.
//
// Simple wrappers like fake_storage.go and google3_storage.go embed this struct in
// order to not have to replicate this code.
//
// Zero value is not useful. Use NewSimple().
type Simple struct {
	wrapper       SimpleSwigWrap
	releaseMemory func()
}

// NewSimple returns a wrappedstorage.Simple that's ready to use. The given finalizer
// (which should clean up any SWIG-related memory) will be called when the object is
// garbage-collected.
//
// Returns nil if `finalizer` is nil.
func NewSimple(w SimpleSwigWrap, finalizer func()) *Simple {
	if finalizer == nil {
		return nil
	}

	s := &Simple{wrapper: w, releaseMemory: finalizer}
	runtime.SetFinalizer(s, func(s *Simple) {
		s.releaseMemory()
	})
	return s
}

// CreateBenchmarkInfo creates a benchmark info record. See interface description for
// more information.
func (s *Simple) CreateBenchmarkInfo(_ context.Context, bi *pgpb.BenchmarkInfo) (*pgpb.CreationResponse, error) {
	cr := &pgpb.CreationResponse{}
	if !s.wrapper.CreateBenchmarkInfo(bi, cr) {
		return cr, errors.New(cr.GetStatus().GetFailMessage())
	}
	// go/mako-go-swig-finalizer-problem
	runtime.KeepAlive(s)
	return cr, nil
}

// UpdateBenchmarkInfo updates a benchmark info record. See interface description for
// more information.
func (s *Simple) UpdateBenchmarkInfo(_ context.Context, bi *pgpb.BenchmarkInfo) (*pgpb.ModificationResponse, error) {
	mr := &pgpb.ModificationResponse{}
	if !s.wrapper.UpdateBenchmarkInfo(bi, mr) {
		return mr, errors.New(mr.GetStatus().GetFailMessage())
	}
	// go/mako-go-swig-finalizer-problem
	runtime.KeepAlive(s)
	return mr, nil
}

// QueryBenchmarkInfo queries a benchmark info record. See interface description for
// more information.
func (s *Simple) QueryBenchmarkInfo(_ context.Context, biq *pgpb.BenchmarkInfoQuery) (*pgpb.BenchmarkInfoQueryResponse, error) {
	biqr := &pgpb.BenchmarkInfoQueryResponse{}
	if !s.wrapper.QueryBenchmarkInfo(biq, biqr) {
		return biqr, errors.New(biqr.GetStatus().GetFailMessage())
	}
	// go/mako-go-swig-finalizer-problem
	runtime.KeepAlive(s)
	return biqr, nil
}

// DeleteBenchmarkInfo deletes a benchmark info record. See interface description for
// more information.
func (s *Simple) DeleteBenchmarkInfo(_ context.Context, biq *pgpb.BenchmarkInfoQuery) (*pgpb.ModificationResponse, error) {
	mr := &pgpb.ModificationResponse{}
	if !s.wrapper.DeleteBenchmarkInfo(biq, mr) {
		return mr, errors.New(mr.GetStatus().GetFailMessage())
	}
	// go/mako-go-swig-finalizer-problem
	runtime.KeepAlive(s)
	return mr, nil
}

// CountBenchmarkInfo counts matching benchmark info records. See interface description for
// more information.
func (s *Simple) CountBenchmarkInfo(_ context.Context, biq *pgpb.BenchmarkInfoQuery) (*pgpb.CountResponse, error) {
	cr := &pgpb.CountResponse{}
	if !s.wrapper.CountBenchmarkInfo(biq, cr) {
		return cr, errors.New(cr.GetStatus().GetFailMessage())
	}
	// go/mako-go-swig-finalizer-problem
	runtime.KeepAlive(s)
	return cr, nil
}

// CreateRunInfo creates a benchmark info record. See interface description for
// more information.
func (s *Simple) CreateRunInfo(_ context.Context, ri *pgpb.RunInfo) (*pgpb.CreationResponse, error) {
	cr := &pgpb.CreationResponse{}
	if !s.wrapper.CreateRunInfo(ri, cr) {
		return cr, errors.New(cr.GetStatus().GetFailMessage())
	}
	// go/mako-go-swig-finalizer-problem
	runtime.KeepAlive(s)
	return cr, nil
}

// UpdateRunInfo updates a benchmark info record. See interface description for
// more information.
func (s *Simple) UpdateRunInfo(_ context.Context, ri *pgpb.RunInfo) (*pgpb.ModificationResponse, error) {
	mr := &pgpb.ModificationResponse{}
	if !s.wrapper.UpdateRunInfo(ri, mr) {
		return mr, errors.New(mr.GetStatus().GetFailMessage())
	}
	// go/mako-go-swig-finalizer-problem
	runtime.KeepAlive(s)
	return mr, nil
}

// QueryRunInfo queries a benchmark info record. See interface description for
// more information.
func (s *Simple) QueryRunInfo(_ context.Context, riq *pgpb.RunInfoQuery) (*pgpb.RunInfoQueryResponse, error) {
	riqr := &pgpb.RunInfoQueryResponse{}
	if !s.wrapper.QueryRunInfo(riq, riqr) {
		return riqr, errors.New(riqr.GetStatus().GetFailMessage())
	}
	// go/mako-go-swig-finalizer-problem
	runtime.KeepAlive(s)
	return riqr, nil
}

// DeleteRunInfo deletes a benchmark info record. See interface description for
// more information.
func (s *Simple) DeleteRunInfo(_ context.Context, riq *pgpb.RunInfoQuery) (*pgpb.ModificationResponse, error) {
	mr := &pgpb.ModificationResponse{}
	if !s.wrapper.DeleteRunInfo(riq, mr) {
		return mr, errors.New(mr.GetStatus().GetFailMessage())
	}
	// go/mako-go-swig-finalizer-problem
	runtime.KeepAlive(s)
	return mr, nil
}

// CountRunInfo counts matching benchmark info records. See interface description for
// more information.
func (s *Simple) CountRunInfo(_ context.Context, riq *pgpb.RunInfoQuery) (*pgpb.CountResponse, error) {
	cr := &pgpb.CountResponse{}
	if !s.wrapper.CountRunInfo(riq, cr) {
		return cr, errors.New(cr.GetStatus().GetFailMessage())
	}
	// go/mako-go-swig-finalizer-problem
	runtime.KeepAlive(s)
	return cr, nil
}

// CreateSampleBatch creates a benchmark info record. See interface description for
// more information.
func (s *Simple) CreateSampleBatch(_ context.Context, sb *pgpb.SampleBatch) (*pgpb.CreationResponse, error) {
	cr := &pgpb.CreationResponse{}
	if !s.wrapper.CreateSampleBatch(sb, cr) {
		return cr, errors.New(cr.GetStatus().GetFailMessage())
	}
	// go/mako-go-swig-finalizer-problem
	runtime.KeepAlive(s)
	return cr, nil
}

// QuerySampleBatch queries a benchmark info record. See interface description for
// more information.
func (s *Simple) QuerySampleBatch(_ context.Context, sbq *pgpb.SampleBatchQuery) (*pgpb.SampleBatchQueryResponse, error) {
	sbqr := &pgpb.SampleBatchQueryResponse{}
	if !s.wrapper.QuerySampleBatch(sbq, sbqr) {
		return sbqr, errors.New(sbqr.GetStatus().GetFailMessage())
	}
	// go/mako-go-swig-finalizer-problem
	runtime.KeepAlive(s)
	return sbqr, nil
}

// DeleteSampleBatch deletes a benchmark info record. See interface description for
// more information.
func (s *Simple) DeleteSampleBatch(_ context.Context, sbq *pgpb.SampleBatchQuery) (*pgpb.ModificationResponse, error) {
	mr := &pgpb.ModificationResponse{}
	if !s.wrapper.DeleteSampleBatch(sbq, mr) {
		return mr, errors.New(mr.GetStatus().GetFailMessage())
	}
	// go/mako-go-swig-finalizer-problem
	runtime.KeepAlive(s)
	return mr, nil
}

// GetMetricValueCountMax returns the max number of metric values that can be
// saved per run.
// See interface description for more information.
func (s *Simple) GetMetricValueCountMax(_ context.Context) (int, error) {
	var d int
	if errStr := s.wrapper.GetMetricValueCountMax(&d); errStr != "" {
		return 0, fmt.Errorf("GetMetricValueCountMax() err: %s", errStr)
	}
	// go/mako-go-swig-finalizer-problem
	runtime.KeepAlive(s)
	return d, nil
}

// GetSampleErrorCountMax returns the max number of errors that can be saved per run.
// See interface description for more information.
func (s *Simple) GetSampleErrorCountMax(_ context.Context) (int, error) {
	var d int
	if errStr := s.wrapper.GetSampleErrorCountMax(&d); errStr != "" {
		return 0, fmt.Errorf("GetSampleErrorCountMax() err: %s", errStr)
	}
	// go/mako-go-swig-finalizer-problem
	runtime.KeepAlive(s)
	return d, nil
}

// GetBatchSizeMax returns the max binary size (in base 10 bytes).
// See interface description for more information.
func (s *Simple) GetBatchSizeMax(_ context.Context) (int, error) {
	var d int
	if errStr := s.wrapper.GetBatchSizeMax(&d); errStr != "" {
		return 0, fmt.Errorf("GetBatchSizeMax() err: %s", errStr)
	}
	// go/mako-go-swig-finalizer-problem
	runtime.KeepAlive(s)
	return d, nil
}

// GetHostname returns the hostname backing this Storage implementation.
// See interface description for more information.
func (s *Simple) GetHostname(_ context.Context) string {
	hostname := s.wrapper.GetHostname()
	// go/mako-go-swig-finalizer-problem
	runtime.KeepAlive(s)
	return hostname
}

// GetSwigWrap returns the contained SWIG wrapper. Implements wrappedstorage.Interface.
func (s *Simple) GetSwigWrap() SwigWrap {
	w := s.wrapper
	// go/mako-go-swig-finalizer-problem
	runtime.KeepAlive(s)
	return w
}

// SimpleSwigWrap is the interface of a direct, straight-forward wrapping of mako::Storage
type SimpleSwigWrap interface {
	SwigWrap
	CreateBenchmarkInfo(*pgpb.BenchmarkInfo, *pgpb.CreationResponse) bool
	UpdateBenchmarkInfo(*pgpb.BenchmarkInfo, *pgpb.ModificationResponse) bool
	QueryBenchmarkInfo(*pgpb.BenchmarkInfoQuery, *pgpb.BenchmarkInfoQueryResponse) bool
	DeleteBenchmarkInfo(*pgpb.BenchmarkInfoQuery, *pgpb.ModificationResponse) bool
	CountBenchmarkInfo(*pgpb.BenchmarkInfoQuery, *pgpb.CountResponse) bool

	CreateRunInfo(*pgpb.RunInfo, *pgpb.CreationResponse) bool
	UpdateRunInfo(*pgpb.RunInfo, *pgpb.ModificationResponse) bool
	QueryRunInfo(*pgpb.RunInfoQuery, *pgpb.RunInfoQueryResponse) bool
	DeleteRunInfo(*pgpb.RunInfoQuery, *pgpb.ModificationResponse) bool
	CountRunInfo(*pgpb.RunInfoQuery, *pgpb.CountResponse) bool

	CreateSampleBatch(*pgpb.SampleBatch, *pgpb.CreationResponse) bool
	QuerySampleBatch(*pgpb.SampleBatchQuery, *pgpb.SampleBatchQueryResponse) bool
	DeleteSampleBatch(*pgpb.SampleBatchQuery, *pgpb.ModificationResponse) bool

	GetMetricValueCountMax(*int) string
	GetSampleErrorCountMax(*int) string
	GetBatchSizeMax(*int) string
	GetHostname() string
}
