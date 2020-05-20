package mock

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"cloud.google.com/go/storage"
)

// I don't know if it is easier or not to use go mock, but we really only need two things:
// 1) Ability to mimick creation of buckets and objects
// 2) Ability to mimick returning errors
//
// We don't need arbitrary return values, so generators like go mock or testify might be
// overkill and doesn't give us the flexibility we need (e.g., will have to specify and
// and reason about the state after each call rather than just pretend we have this fake
// storage. The behavior of these commands, at the level of detail we care about, is pretty
// easy to replicate.

var (
	MethodNewStorageBucket    = Method("NewStorageBucket")
	MethodDeleteStorageBucket = Method("NewDeleteStorageBucket")
	MethodListChildrenFiles   = Method("ListChildrenFiles")
	MethodListDirectChildren  = Method("ListDirectChildren")
	MethodAttrObject          = Method("AttrObject")
	MethodCopyObject          = Method("CopyObject")
	MethodReadObject          = Method("ReadObject")
	MethodWriteObject         = Method("WriteObject")
	MethodDeleteObject        = Method("DeleteObject")
	MethodDownload            = Method("Download")
	MethodUpload              = Method("Upload")
)

// mock GCS Client
type clientMocker struct {
	// project with buckets
	gcp map[project]*buckets
	// error map
	// - on each call of the higher level function that calls any number of methods
	//	in this library, you can use SetError(map[Method]*ReturnError) or ClearError()
	//	to create the error return values you want. Default is nil.
	err map[Method]*ReturnError

	// reverse index to lookup which project a bucket is under as GCS has a global
	// bucket namespace.
	revIndex map[bucket]project
}

func NewClientMocker() *clientMocker {
	c := &clientMocker{
		gcp:      make(map[project]*buckets),
		err:      make(map[Method]*ReturnError),
		revIndex: make(map[bucket]project),
	}
	return c
}

// SetError sets the number of calls of an interface function before an error is returned.
// Otherwise it will return the err of the mock function itself (which is usually nil).
func (c *clientMocker) SetError(m map[Method]ReturnError) {
	for k, v := range m {
		c.err[k] = &ReturnError{
			NumCall: v.NumCall,
			Err:     v.Err,
		}
	}
}

// ClearError clears the error map in mock client
func (c *clientMocker) ClearError() {
	for k := range c.err {
		// Apparently Go is okay with deleting keys as you iterate.
		delete(c.err, k)
	}
}

// getError is a helper that returns the error if it is set for this function
func (c *clientMocker) getError(funcName Method) error {
	if val, ok := c.err[funcName]; ok {
		if val.NumCall <= 0 {
			delete(c.err, funcName)
			return val.Err
		}
		val.NumCall = val.NumCall - 1
	}
	return nil
}

// getBucketRoot is a helper that returns the objects bucket if it exists
func (c *clientMocker) getBucketRoot(bkt string) *objects {
	p, ok := c.revIndex[bucket(bkt)]
	if !ok {
		return nil
	}

	bktRoot, ok := c.gcp[p].bkt[bucket(bkt)]
	if !ok {
		return nil
	}
	return bktRoot
}

// NewStorageBucket mock creates a new storage bucket in gcp
func (c *clientMocker) NewStorageBucket(ctx context.Context, bkt, projectName string) error {
	if err := c.getError(MethodNewStorageBucket); err != nil {
		return err
	}

	p := project(projectName)

	if _, ok := c.revIndex[bucket(bkt)]; ok {
		return NewBucketExistError(bkt)
	}

	if _, ok := c.gcp[p]; !ok {
		c.gcp[p] = &buckets{
			bkt: make(map[bucket]*objects),
		}
	}
	c.gcp[p].bkt[bucket(bkt)] = &objects{
		obj: make(map[mockpath]*object),
	}
	c.revIndex[bucket(bkt)] = p
	return nil
}

// DeleteStorageBucket mock deletes a storage bucket from gcp
func (c *clientMocker) DeleteStorageBucket(ctx context.Context, bkt string) error {
	if err := c.getError(MethodDeleteStorageBucket); err != nil {
		return err
	}

	bktName := bucket(bkt)

	p, ok := c.revIndex[bktName]
	if !ok {
		return NewNoBucketError(bkt)
	}

	delete(c.gcp[p].bkt, bktName)
	delete(c.revIndex, bktName)
	return nil
}

// Exists mock check if an object exists
func (c *clientMocker) Exists(ctx context.Context, bkt, objPath string) bool {
	bktRoot := c.getBucketRoot(bkt)
	if bktRoot == nil {
		return false
	}

	// just the bucket
	if objPath == "" {
		return true
	}

	dir, obj := filepath.Split(objPath)
	if _, ok := bktRoot.obj[newMockPath(dir, obj)]; ok {
		return true
	}

	// could be asking for if a directory exists. Since our structure is flat, at
	// path of an object containing the searched for directory as its subpath means
	// the directory "exists"
	// NOTE: this is inefficient....but we are not scale testing with mock anyway.
	for k := range bktRoot.obj {
		if strings.HasPrefix(k.dir, objPath) {
			return true
		}
	}
	return false
}

// ListChildrenFiles mock lists all children recursively
func (c *clientMocker) ListChildrenFiles(ctx context.Context, bkt, dirPath string) ([]string, error) {
	if err := c.getError(MethodListChildrenFiles); err != nil {
		return nil, err
	}

	bktRoot := c.getBucketRoot(bkt)
	if bktRoot == nil {
		return nil, NewNoBucketError(bkt)
	}

	dir := strings.TrimRight(dirPath, " /") + "/"
	var children []string
	for k := range bktRoot.obj {
		if strings.HasPrefix(k.dir, dir) {
			children = append(children, k.toString())
		}
	}

	return children, nil
}

// mock lists all direct children recursively
func (c *clientMocker) ListDirectChildren(ctx context.Context, bkt, dirPath string) ([]string, error) {
	if err := c.getError(MethodListDirectChildren); err != nil {
		return nil, err
	}

	bktRoot := c.getBucketRoot(bkt)
	if bktRoot == nil {
		return nil, NewNoBucketError(bkt)
	}

	dir := strings.TrimRight(dirPath, " /") + "/"
	var children []string
	for k, _ := range bktRoot.obj {
		if k.dir == dir {
			children = append(children, k.toString())
		}
	}

	return children, nil
}

// AttrObject mock returns the attribute of an object
func (c *clientMocker) AttrObject(ctx context.Context, bkt, objPath string) (*storage.ObjectAttrs, error) {
	if err := c.getError(MethodAttrObject); err != nil {
		return nil, err
	}

	bktRoot := c.getBucketRoot(bkt)
	if bktRoot == nil {
		return nil, NewNoBucketError(bkt)
	}

	dir, obj := filepath.Split(objPath)
	if obj == "" {
		return nil, NewNoObjectError(bkt, obj, dir)
	}
	o, ok := bktRoot.obj[newMockPath(dir, obj)]
	if !ok {
		return nil, NewNoObjectError(bkt, obj, dir)
	}

	return &storage.ObjectAttrs{
		Bucket: bkt,
		Name:   objPath,
		Size:   int64(len(o.content)),
	}, nil
}

// CopyObject mocks the copying of one object to another
func (c *clientMocker) CopyObject(ctx context.Context, srcBkt, srcObjPath, dstBkt, dstObjPath string) error {
	if err := c.getError(MethodCopyObject); err != nil {
		return err
	}

	srcBktRoot := c.getBucketRoot(srcBkt)
	if srcBktRoot == nil {
		return NewNoBucketError(srcBkt)
	}

	dstBktRoot := c.getBucketRoot(dstBkt)
	if dstBktRoot == nil {
		return NewNoBucketError(dstBkt)
	}

	srcDir, srcObjName := filepath.Split(srcObjPath)
	if srcObjName == "" {
		return NewNoObjectError(srcBkt, srcObjName, srcDir)
	}

	dstDir, dstObjName := filepath.Split(dstObjPath)
	if dstObjName == "" {
		return NewNoObjectError(dstBkt, dstObjName, dstDir)
	}

	srcMockPath := newMockPath(srcDir, srcObjName)
	dstMockPath := newMockPath(dstDir, dstObjName)

	srcObj, ok := srcBktRoot.obj[srcMockPath]
	if !ok {
		return NewNoObjectError(srcBkt, srcObjName, srcDir)
	}

	dstBktRoot.obj[dstMockPath] = &object{
		name:    srcObj.name,
		bkt:     dstBkt,
		content: make([]byte, len(srcBktRoot.obj[srcMockPath].content)),
	}
	copy(dstBktRoot.obj[dstMockPath].content, srcBktRoot.obj[srcMockPath].content)
	return nil
}

// ReadObject mocks reading from an object
func (c *clientMocker) ReadObject(ctx context.Context, bkt, objPath string) ([]byte, error) {
	if err := c.getError(MethodReadObject); err != nil {
		return nil, err
	}

	bktRoot := c.getBucketRoot(bkt)
	if bktRoot == nil {
		return nil, NewNoBucketError(bkt)
	}

	dir, objName := filepath.Split(objPath)
	if objName == "" {
		return nil, NewNoObjectError(bkt, objName, dir)
	}

	obj, ok := bktRoot.obj[newMockPath(dir, objName)]
	if !ok {
		return nil, NewNoObjectError(bkt, objName, dir)
	}

	return obj.content, nil
}

// WriteObject mocks writing to an object
func (c *clientMocker) WriteObject(ctx context.Context, bkt, objPath string, content []byte) (int, error) {
	if err := c.getError(MethodWriteObject); err != nil {
		return -1, err
	}

	bktRoot := c.getBucketRoot(bkt)
	if bktRoot == nil {
		return -1, NewNoBucketError(bkt)
	}

	dir, objName := filepath.Split(objPath)
	if objName == "" {
		return -1, NewNoObjectError(bkt, objName, dir)
	}

	mockPath := newMockPath(dir, objName)
	bktRoot.obj[mockPath] = &object{
		name:    mockPath,
		bkt:     bkt,
		content: make([]byte, len(content)),
	}
	copy(bktRoot.obj[mockPath].content, content)
	return len(content), nil
}

// DeleteObject mocks deleting an object
func (c *clientMocker) DeleteObject(ctx context.Context, bkt, objPath string) error {
	if err := c.getError(MethodDeleteObject); err != nil {
		return err
	}

	bktRoot := c.getBucketRoot(bkt)
	if bktRoot == nil {
		return nil
	}

	dir, objName := filepath.Split(objPath)
	if objName == "" {
		return nil
	}

	delete(bktRoot.obj, newMockPath(dir, objName))
	return nil
}

// Download mocks downloading an object to a local file
func (c *clientMocker) Download(ctx context.Context, bkt, objPath, filePath string) error {
	if err := c.getError(MethodDownload); err != nil {
		return err
	}

	bktRoot := c.getBucketRoot(bkt)
	if bktRoot == nil {
		return NewNoBucketError(bkt)
	}

	dir, objName := filepath.Split(objPath)
	if objName == "" {
		return NewNoObjectError(bkt, objName, dir)
	}

	obj, ok := bktRoot.obj[newMockPath(dir, objName)]
	if !ok {
		return NewNoObjectError(bkt, objName, dir)
	}

	f, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(obj.content)
	return err
}

// Upload mocks uploading a local file to an object
func (c *clientMocker) Upload(ctx context.Context, bkt, objPath, filePath string) error {
	if err := c.getError(MethodUpload); err != nil {
		return err
	}

	bktRoot := c.getBucketRoot(bkt)
	if bktRoot == nil {
		return NewNoBucketError(bkt)
	}

	dir, objName := filepath.Split(objPath)
	if objName == "" {
		return NewNoObjectError(bkt, objName, dir)
	}

	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}

	mockPath := newMockPath(dir, objName)
	bktRoot.obj[mockPath] = &object{
		name:    mockPath,
		bkt:     bkt,
		content: make([]byte, len(content)),
	}
	copy(bktRoot.obj[mockPath].content, content)
	return nil
}
