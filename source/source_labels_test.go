/*
 * Copyright 2019 The Knative Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package source

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestLabels(t *testing.T) {

	sourceLabeles := Labels("foo", "foo-source-controller")

	wantTags := map[string]string{
		sourceControllerName: "foo-source-controller",
		sourceName:           "foo",
	}

	eq := cmp.Equal(sourceLabeles, wantTags)
	if !eq {
		t.Fatalf("%v is not equal to %v", sourceLabeles, wantTags)
	}
}
