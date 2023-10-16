// Copyright 2023 Cisco Systems, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import (
	"testing"
)

func TestAbbreviateString(t *testing.T) {
	tests := []struct {
		name string
		s    string
		n    uint
		want string
	}{
		// ------           1         2
		// ------  12345678901234567890
		{
			name: "ASCII string, no trim required",
			s:    "Hello",
			n:    10,
			want: "Hello",
		},
		{
			name: "ASCII string, trim required",
			s:    "Hello, world!",
			n:    10,
			want: "Hello, wo…",
		},
		{
			name: "Unicode string, no trim required",
			s:    "こんにちは、世界！",
			n:    9,
			want: "こんにちは、世界！",
		},
		{
			name: "Unicode string, trim required",
			s:    "こんにちは、世界！",
			n:    5,
			want: "こんにち…",
		},
		{
			name: "n is 0",
			s:    "Hello, world!",
			n:    0,
			want: "",
		},
		{
			name: "n is 1",
			s:    "Hello, world!",
			n:    1,
			want: "…",
		},
		{
			name: "n is 1 with empty string",
			s:    "",
			n:    1,
			want: "",
		},
		{
			name: "n is 1 with short string",
			s:    "a",
			n:    1,
			want: "a",
		},
		{
			name: "n is 1 with short unicode string",
			s:    "ᇂ",
			n:    1,
			want: "ᇂ",
		},
		{
			name: "Unicode string, trim at the end",
			s:    "こんにちは、世界！",
			n:    8,
			want: "こんにちは、世…",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := abbreviateString(tt.s, tt.n); got != tt.want {
				t.Errorf("%v\n\tabbreviateString() returned %q instead of %q", tt.name, got, tt.want)
			}
		})
	}
}
