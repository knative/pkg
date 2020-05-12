package mock

import (
	"path/filepath"
)

// more friendly type casts for better readability of what some strings are
type project string
type bucketName string
type Method string

// mockpath contains the bucket path to an object and the object name
type mockpath struct {
	dir string
	obj string
}

func newMockPath(dir, obj string) mockpath {
	return mockpath{
		dir: dir,
		obj: obj,
	}
}

// toString stringify mockpath
func (m mockpath) toString() string {
	return filepath.Join(m.dir, m.obj)
}

// Fake GCS objects
type object struct {
	name mockpath
	//NOTE: current ObjectAttrs supported:
	//	Size
	//	Bucket
	//	Name
	bkt     string
	content []byte
}

// bucket of objects - structure is flat
type objects struct {
	o map[mockpath]*object
}

// project with buckets
type buckets struct {
	bucket map[bucketName]*objects
}

// Error map to return custom errors for specific methods
type ReturnError struct {
	NumCall uint8
	Err     error
}
