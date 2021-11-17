/*
Copyright 2020 The Knative Authors.

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
package generators

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParseComments(t *testing.T) {
	comment := []string{
		"This is an example comment to parse",
		"",
		" +foo",
		"+bar",
		"+with:option",
		"+pair:key=value",
		"+manypairs:key1=value1,key2=value2",
		"+duplicate:key1=value1,key1=value2",
	}

	extracted := ExtractCommentTags("+", comment)

	expected := CommentTags{
		"foo": {},
		"bar": {},
		"with": {
			"option": {},
		},
		"pair": {
			"key": []string{"value"},
		},
		"manypairs": {
			"key1": []string{"value1"},
			"key2": []string{"value2"},
		},
		"duplicate": {
			"key1": []string{"value1", "value2"},
		},
	}

	if diff := cmp.Diff(expected, extracted); diff != "" {
		t.Error("diff (-want, +got): ", diff)
	}
}

func TestMergeDuplicates(t *testing.T) {
	comment := []string{
		"This is an example comment to parse",
		"",
		"+foo",
		" +foo",
		"+bar:key=value",
		"+bar",
		"+manypairs:key1=value1",
		"+manypairs:key2=value2",
		"+manypairs:key1=value3",
		"+oops:,,,",
	}

	extracted := ExtractCommentTags("+", comment)

	expected := CommentTags{
		"foo": {},
		"bar": {
			"key": []string{"value"},
		},
		"manypairs": {
			"key1": []string{"value1", "value3"},
			"key2": []string{"value2"},
		},
		"oops": {},
	}

	if diff := cmp.Diff(expected, extracted); diff != "" {
		t.Error("diff (-want, +got): ", diff)
	}
}
