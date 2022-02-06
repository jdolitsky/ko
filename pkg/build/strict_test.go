// Copyright 2020 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package build

import "testing"

func TestStrictReference(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		strict bool
		path   string
	}{{
		name:   "loose",
		input:  "github.com/foo/bar",
		strict: false,
		path:   "github.com/foo/bar",
	}, {
		name:   "strict",
		input:  "ko://github.com/foo/bar",
		strict: true,
		path:   "github.com/foo/bar",
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ref := newRef(test.input)
			if got, want := ref.IsStrict(), test.strict; got != want {
				t.Errorf("got: %v, want: %v", got, want)
			}
			if got, want := ref.Path(), test.path; got != want {
				t.Errorf("got: %v, want: %v", got, want)
			}
			if got, want := ref.String(), test.input; got != want {
				t.Errorf("got: %v, want: %v", got, want)
			}
		})
	}
}

func TestStrictOverrideReference(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		strict bool
		path   string
	}{{
		name:   "loose",
		input:  "bloop",
		strict: false,
		path:   "bloop",
	}, {
		name:   "strict",
		input:  "koverride://blip",
		strict: true,
		path:   "blip",
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ref := newOverrideRef(test.input)
			if got, want := ref.IsStrict(), test.strict; got != want {
				t.Errorf("got: %v, want: %v", got, want)
			}
			if got, want := ref.Path(), test.path; got != want {
				t.Errorf("got: %v, want: %v", got, want)
			}
			if got, want := ref.String(), test.input; got != want {
				t.Errorf("got: %v, want: %v", got, want)
			}
		})
	}
}
