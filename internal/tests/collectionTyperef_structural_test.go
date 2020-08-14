package tests

import (
	"context"

	conflictresolution "github.com/PapaCharlie/go-restli/internal/tests/generated/conflictResolution"
	"github.com/PapaCharlie/go-restli/internal/tests/generated/testsuite"
	collectiontyperef "github.com/PapaCharlie/go-restli/internal/tests/generated/testsuite/typerefs/collectionTyperef"
)

// STRUCTURAL TEST ENSURE INTERFACE DOES NOT DRIFT FOR testsuite.typerefs.collectionTyperef
var _ = collectiontyperef.Client(new(collectionTyperefClient))

type collectionTyperefClient int

func (c *collectionTyperefClient) Get(testsuite.Url) (*conflictresolution.Message, error) {
	panic(nil)
}

func (c *collectionTyperefClient) GetWithContext(context.Context, testsuite.Url) (*conflictresolution.Message, error) {
	panic(nil)
}
