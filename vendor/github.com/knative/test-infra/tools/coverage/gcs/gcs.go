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

// Package gcs stores functions that relates to GCS operations,
// without dependency on the package calc
package gcs

import (
	"context"
	"log"
	"path"
	"strconv"

	"cloud.google.com/go/storage"
	"github.com/knative/test-infra/tools/coverage/artifacts"
	"github.com/knative/test-infra/tools/coverage/logUtil"
	"google.golang.org/api/iterator"
)

type StorageClientIntf interface {
	Bucket(bucketName string) *storage.BucketHandle
	ListGcsObjects(ctx context.Context, bucketName, prefix, delim string) (
		objects []string)
	ProfileReader(ctx context.Context, bucket, object string) *artifacts.ProfileReader
	////CovList(Ctx context.Context, bucket, object string, concernedFiles *map[string]bool) (g *CoverageList)
	DoesObjectExist(ctx context.Context, bucket, object string) bool
}

type StorageClient struct {
	storage.Client
}

func NewStorageClient(ctx context.Context) *StorageClient {
	client, err := storage.NewClient(ctx)

	if err != nil {
		logUtil.LogFatalf("Failed to create client: %v", err)
	}
	return &StorageClient{*client}
}

func (client *StorageClient) ListGcsObjects(ctx context.Context, bucketName,
	prefix, delim string) (objects []string) {
	it := client.Bucket(bucketName).Objects(ctx, &storage.Query{
		Prefix:    prefix,
		Delimiter: delim,
	})

	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatalf("Error iterating: %v", err)
		}

		if attrs.Prefix != "" {
			objects = append(objects, path.Base(attrs.Prefix))
		}
	}
	log.Println("end of ListGcsObjects(...)")
	return
}

func (client StorageClient) ProfileReader(ctx context.Context, bucket,
	object string) *artifacts.ProfileReader {
	log.Printf("Running ProfileReader on bucket '%s', object='%s'\n",
		bucket, object)

	o := client.Bucket(bucket).Object(object)
	reader, err := o.NewReader(ctx)
	if err != nil {
		logUtil.LogFatalf("o.NewReader(Ctx) error: %v", err)
	}
	return artifacts.NewProfileReader(reader)
}

// DoesObjectExist checks whether an object exists in GCS bucket
func (client StorageClient) DoesObjectExist(ctx context.Context, bucket, object string) bool {
	_, err := client.Bucket(bucket).Object(object).Attrs(ctx)
	if err != nil {
		log.Printf("Error getting attrs from object '%s': %v", object, err)
		return false
	}
	return true
}

type GcsBuild struct {
	StorageClient StorageClientIntf
	Bucket        string
	Job           string
	Build         int
	CovThreshold  int
}

func (b *GcsBuild) BuildStr() string {
	return strconv.Itoa(b.Build)
}

type GcsArtifacts struct {
	artifacts.Artifacts
	Ctx    context.Context
	Client StorageClientIntf
	Bucket string
}

func NewGcsArtifacts(ctx context.Context, client StorageClientIntf,
	bucket string, baseArtifacts artifacts.Artifacts) *GcsArtifacts {
	return &GcsArtifacts{baseArtifacts, ctx, client, bucket}
}

func (arts *GcsArtifacts) ProfileReader() *artifacts.ProfileReader {
	return arts.Client.ProfileReader(arts.Ctx, arts.Bucket, arts.ProfilePath())
}
