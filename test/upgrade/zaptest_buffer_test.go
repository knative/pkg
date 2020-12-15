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

package upgrade_test

import (
	"bytes"
	"sync"

	"go.uber.org/zap/zaptest"
)

// To avoid race condition on zaptest.Buffer, see: https://stackoverflow.com/a/36226525/844449
type buffer struct {
	bytes.Buffer
	sync.Mutex
	zaptest.Syncer
}

func (b *buffer) Read(p []byte) (n int, err error) {
	b.Mutex.Lock()
	defer b.Mutex.Unlock()
	return b.Buffer.Read(p)
}

func (b *buffer) Write(p []byte) (n int, err error) {
	b.Mutex.Lock()
	defer b.Mutex.Unlock()
	return b.Buffer.Write(p)
}

func (b *buffer) String() string {
	b.Mutex.Lock()
	defer b.Mutex.Unlock()
	return b.Buffer.String()
}
