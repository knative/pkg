// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/* This file was copied from encoding/json/tags_test.go, part of the built in json lib. */

package validators

import (
	"testing"
)

func TestTagParsing(t *testing.T) {
	name, opts := parseTag("field,foobar,foo")
	if name != "field" {
		t.Fatalf("name = %q, want field", name)
	}
	for _, tt := range []struct {
		opt  string
		want bool
	}{
		{"foobar", true},
		{"foo", true},
		{"bar", false},
	} {
		if opts.Contains(tt.opt) != tt.want {
			t.Errorf("Contains(%q) = %v", tt.opt, !tt.want)
		}
	}
	// Test the empty case.
	opts = ""
	if opts.Contains("foo") != false {
		t.Errorf("\"\".Contains(%q) = %v", "foo", true)
	}
}
