// Copyright 2022 Cisco Systems, Inc.
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

package version

import (
	"reflect"
	"testing"
)

func TestIsDev(t *testing.T) {
	t.Skip("disabled test until test case is fixed") // TODO

	tests := []struct {
		name string
		dev  string
		want bool
	}{
		{"valid true", "true", true},
		{"valid false", "false", false},
		{"invalid empty", "", false},
		{"invalid any string", "test", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defIsDev = tt.dev
			if got := IsDev(); got != tt.want {
				t.Errorf("IsDev() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetVersion(t *testing.T) {
	t.Skip("disabled test until test case is fixed") // TODO

	tests := []struct {
		name    string
		version string
		gitHash string
		want    string
	}{
		{
			name:    "valid",
			version: "0.0.0-123",
			gitHash: "51657a4",
			want:    "0.0.0-123",
		},
		{
			name:    "valid default",
			version: "",
			gitHash: "51657a4",
			want:    "51657a4",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defVersion = tt.version
			defGitHash = tt.gitHash
			if got := GetVersionShort(); got != tt.want {
				t.Errorf("GetVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_localTime(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "valid",
			args: args{
				s: "1632254040",
			},
			want: "2021-09-21 12:54:00 -0700 PDT",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := localTime(tt.args.s); got.String() != tt.want {
				t.Errorf("localTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildInfo(t *testing.T) {
	t.Skip("disabled test until test case is fixed") // TODO

	type args struct {
		gitBranch      string
		gitHash        string
		gitDirty       string
		buildTimestamp string
		buildHost      string
		gitTimestamp   string
	}
	tests := []struct {
		name string
		args args
		want [][]string
	}{
		{
			name: "",
			args: args{
				gitBranch:      "main",
				gitHash:        "51657a4",
				gitDirty:       "Clean",
				buildTimestamp: "1632254040",
				buildHost:      "TEST-M-LCEN",
				gitTimestamp:   "1632254040",
			},
			want: [][]string{
				{"abc", "xyz"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defGitBranch = tt.args.gitBranch
			defGitHash = tt.args.gitHash
			defGitDirty = tt.args.gitDirty
			defBuildTimestamp = tt.args.buildTimestamp
			defBuildHost = tt.args.buildHost
			defGitTimestamp = tt.args.gitTimestamp

			//out := GetVersionDetailsHuman()

			if got := GetVersionDetailsHuman(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetVersionDetailsHuman() = %v, want %v", got, tt.want)
			}
		})
	}
}
