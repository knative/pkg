package mock

import (
	"context"
	"fmt"

	"knative.dev/pkg/test/gcs"
)

// Example on how to override errors
const (
	bkt  = "NewBkt"
	proj = "NewProject"
)

func topFunction(c gcs.Client) error {
	ctx := context.Background()
	if err := c.NewStorageBucket(ctx, bkt, proj); err != nil {
		return err
	}

	// Should have returned error, but SetError override to nil
	if _, err := c.ReadObject(ctx, bkt, "non-existent-file"); err != nil {
		return err
	}

	if _, err := c.ListChildrenFiles(ctx, bkt, ""); err != nil {
		return err
	}

	if _, err := c.ListChildrenFiles(ctx, bkt, ""); err != nil {
		return err
	}

	// Should not have returned error, but SetError override to NewNoBucketError(bkt)
	if _, err := c.ListChildrenFiles(ctx, bkt, ""); err != nil {
		return err
	}

	return nil
}

func ExampleSetError() {
	mockClient := NewClientMocker()

	// Call to ReadObject, first call should return error, but returns nil
	// because it is overridden.
	mockClient.SetError(
		map[Method]*ReturnError{
			MethodReadObject: {
				NumCall: uint8(0),
				Err:     nil,
			},
			MethodListChildrenFiles: {
				NumCall: uint8(2),
				Err:     NewNoBucketError(bkt),
			},
		})

	fmt.Printf("%v", topFunction(mockClient))
	// Output:
	// no bucket NewBkt
}
