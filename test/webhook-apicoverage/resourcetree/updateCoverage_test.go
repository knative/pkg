/*
Copyright 2018 The Knative Authors

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

package resourcetree

import (
	"reflect"
	"testing"
)

func TestSimpleStructValue(t *testing.T) {
	tree := getTestTree(basicTypeName, reflect.TypeOf(baseType{}))
	tree.UpdateCoverage(reflect.ValueOf(getBaseTypeValue()))
	if err := verifyBaseTypeValue("", tree.Root); err != nil {
		t.Fatal(err)
	}
}

func TestPtrValueAllCovered(t *testing.T) {
	tree := getTestTree(ptrTypeName, reflect.TypeOf(ptrType{}))
	tree.UpdateCoverage(reflect.ValueOf(getPtrTypeValueAllCovered()))
	if err := verifyPtrValueAllCovered(tree.Root); err != nil {
		t.Fatal(err)
	}
}

func TestPtrValueSomeCovered(t *testing.T) {
	tree := getTestTree(ptrTypeName, reflect.TypeOf(ptrType{}))
	tree.UpdateCoverage(reflect.ValueOf(getPtrTypeValueSomeCovered()))
	if err := verifyPtrValueSomeCovered(tree.Root); err != nil {
		t.Fatal(err)
	}
}

func TestArrValueAllCovered(t *testing.T) {
	tree := getTestTree(arrayTypeName, reflect.TypeOf(arrayType{}))
	tree.UpdateCoverage(reflect.ValueOf(getArrValueAllCovered()))
	if err := verifyArryValueAllCovered(tree.Root); err != nil {
		t.Fatal(err)
	}
}

func TestArrValueSomeCovered(t *testing.T) {
	tree := getTestTree(arrayTypeName, reflect.TypeOf(arrayType{}))
	tree.UpdateCoverage(reflect.ValueOf(getArrValueSomeCovered()))
	if err := verifyArrValueSomeCovered(tree.Root); err != nil {
		t.Fatal(err)
	}
}

func TestOtherValue(t *testing.T) {
	tree := getTestTree(otherTypeName, reflect.TypeOf(otherType{}))
	tree.UpdateCoverage(reflect.ValueOf(getOtherTypeValue()))
	if err := verifyOtherTypeValue(tree.Root); err != nil {
		t.Fatal(err)
	}
}
