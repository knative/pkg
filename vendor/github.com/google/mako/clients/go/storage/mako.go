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

// Package mako provides a Mako storage client that communicates with the
// externalized Mako (Mako) service at the default hostname,
// 'makoperf.appspot.com', or a user provided hostname.
//
// See https://github.com/google/mako/blob/master/spec/go/storage.go for more
// information about interface.
//
// Calling this storage client's methods isn't needed when using the normal flow
// of Mako. To create a Mako run from performance metrics use Quickstore
// instead (go/mako-quickstore). To use Mako to generate load and
// measure performance, start with Mako Small Load (go/mako-load). This
// storage instance can be passed as an argument to all Mako entry points.
//
// Call this class's methods only when you know what you are doing and want to
// directly access Mako storage programmatically.
//
// The authentication method is chosen based on environment and flags:
//
// - Use Application Default Credentials
// (https://developers.google.com/identity/protocols/application-default-credentials)
// for authentication and communicate with the Mako server over HTTP instead.
// If necessary, use --mako_auth_ca_cert=<path> to specify a path to
// a CA certs bundle file to use for SSL.
//
// The returned client is thread/goroutine-safe.
package mako

import (
	"fmt"

	wrap "github.com/google/mako/clients/cxx/storage/go/mako_client_wrap"
	"github.com/google/mako/clients/go/storage/g3storage"
)

const makoAppHostname = "mako.dev"

// New returns a new Mako client for makoperf.appspot.com.
func New() (*g3storage.Storage, error) {
	return NewWithHostname(makoAppHostname)
}

// NewWithHostname returns a new Mako client connected to an arbitrary server,
// typically used for testing.
func NewWithHostname(hostname string) (*g3storage.Storage, error) {
	error := []string{""}
	w := wrap.NewMakoClient(hostname, error)
	if error[0] != "" {
		return nil, fmt.Errorf("failure calling into C++ NewMakoClient: %v", error[0])
	}
	return g3storage.NewFromWrapper(w), nil
}
