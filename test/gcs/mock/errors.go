package mock

import (
	"fmt"
)

type notEmptyBucketError struct {
	bkt string
}

func (e *notEmptyBucketError) Error() string {
	return fmt.Sprintf("bucket %s not empty, use force=true", e.bkt)
}

func NewNotEmptyBucketError(bkt string) *notEmptyBucketError {
	return &notEmptyBucketError{bkt}
}

type noBucketError struct {
	bkt string
}

func NewNoBucketError(bkt string) *noBucketError {
	return &noBucketError{bkt}
}

func (e *noBucketError) Error() string {
	return fmt.Sprintf("no bucket %s", e.bkt)
}

type bucketExistError struct {
	bkt string
}

func NewBucketExistError(bkt string) *bucketExistError {
	return &bucketExistError{bkt}
}

func (e *bucketExistError) Error() string {
	return fmt.Sprintf("bucket %s already exists", e.bkt)
}

type noObjectError struct {
	bkt  string
	obj  string
	path string
}

func NewNoObjectError(bkt, obj, path string) *noObjectError {
	return &noObjectError{
		bkt:  bkt,
		obj:  obj,
		path: path,
	}
}

func (e *noObjectError) Error() string {
	return fmt.Sprintf("bucket %s does not contain object \"%s\" under path \"%s\"",
		e.bkt, e.obj, e.path)
}
