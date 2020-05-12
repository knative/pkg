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

package gcs

import (
	"context"

	"cloud.google.com/go/storage"
)

type Client interface {
	NewStorageBucket(ctx context.Context, bkt, project string) error
	DeleteStorageBucket(ctx context.Context, bkt string) error
	Exists(ctx context.Context, bkt, objPath string) bool
	ListChildrenFiles(ctx context.Context, bkt, dirPath string) ([]string, error)
	ListDirectChildren(ctx context.Context, bkt, dirPath string) ([]string, error)
	AttrObject(ctx context.Context, bkt, objPath string) (*storage.ObjectAttrs, error)
	CopyObject(ctx context.Context, srcBkt, srcObjPath, dstBkt, dstObjPath string) error
	ReadObject(ctx context.Context, bkt, objPath string) ([]byte, error)
	WriteObject(ctx context.Context, bkt, objPath string, content []byte) (int, error)
	DeleteObject(ctx context.Context, bkt, objPath string) error
	Download(ctx context.Context, bktName, objPath, filePath string) error
	Upload(ctx context.Context, bktName, objPath, filePath string) error
}
