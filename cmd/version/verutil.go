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
	"fmt"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/apex/log"
	"github.com/mitchellh/mapstructure"
)

const (
	defaultVersion = "0.0.0-0+local"
)

//nolint:gochecknoglobals //is intentional
var (
	defVersion        string = defaultVersion
	defIsDev          string = "true"
	defGitHash        string
	defGitBranch      string
	defGitDirty       string
	defGitTimestamp   string
	defBuildTimestamp string
	defBuildHost      string
)

type VersionData struct {
	Version           string `yaml:"version" json:"version" human:"Version"`
	VersionMajor      uint   `yaml:"major" json:"major"`
	VersionMinor      uint   `yaml:"minor" json:"minor"`
	VersionPatch      uint   `yaml:"patch" json:"patch"`
	VersionPrerelease string `yaml:"prerelease" json:"prerelease"`
	VersionMeta       string `yaml:"metadata" json:"metadata"`
	IsDev             bool   `yaml:"dev" json:"dev" human:"Dev Build"`
	GitHash           string `yaml:"git_hash" json:"git_hash" human:"Git Hash"`
	GitBranch         string `yaml:"git_branch" json:"git_branch" human:"Git Branch"`
	GitDirty          bool   `yaml:"git_dirty" json:"git_dirty" human:"Git Dirty"`
	GitTimestamp      uint64 `yaml:"git_timestamp" json:"git_timestamp" human:"Git Timestamp"`
	BuildTimestamp    uint64 `yaml:"build_timestamp" json:"build_timestamp" human:"Build Timestamp"`
	BuildHost         string `yaml:"build_host" json:"build_host" human:"Build Host"`
	Platform          string `yaml:"platform" json:"platform" human:"Platform"`
}

var version VersionData

func init() {
	// compute derived values
	version.Version = defVersion
	version.IsDev = defIsDev == "true"
	version.GitHash = defGitHash
	version.GitBranch = defGitBranch
	version.GitDirty = defGitDirty != "false"
	version.GitTimestamp = parseEpoch(defGitTimestamp)
	version.BuildTimestamp = parseEpoch(defBuildTimestamp)
	version.BuildHost = defBuildHost

	// parse version string
	//const semVerPattern = `^([0-9]+)\.([0-9]+)\.([0-9]+)(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?(?:\+[0-9A-Za-z-]+)?$`
	const semVerPattern = `^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+([0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`
	re := regexp.MustCompile(semVerPattern)
	verElems := re.FindStringSubmatch(version.Version)
	nElems := len(verElems)
	if nElems >= 4 {
		version.VersionMajor = parseVerElement(verElems[1])
		version.VersionMinor = parseVerElement(verElems[2])
		version.VersionPatch = parseVerElement(verElems[3])
	}
	if nElems >= 5 {
		version.VersionPrerelease = verElems[4]
	}
	if nElems >= 6 {
		version.VersionMeta = verElems[5]
	}

	// form platform name
	version.Platform = runtime.GOOS + "-" + runtime.GOARCH
}

func IsDev() bool {
	return version.IsDev
}

func parseEpoch(str string) uint64 {
	// return 0 if no timestamp
	if str == "" {
		return 0
	}

	// parse into a 64-bit value
	u, err := strconv.ParseUint(str, 10, 64)
	if err != nil {
		log.Fatalf("bug: failed to parse timestamp %q: %v", str, err)
	}

	return u
}

func parseVerElement(str string) uint {
	u, err := strconv.ParseUint(str, 10, 32)
	if err != nil {
		log.Fatalf("bug: failed to parse version element %q: %v", str, err)
	}

	return uint(u)
}

func localTime(str string) time.Time {
	var unixTime time.Time

	const (
		base    = 10
		bitSize = 64
	)

	i, err := strconv.ParseInt(str, base, bitSize)
	if err != nil {
		return time.Time{}
	}

	unixTime = time.Unix(i, 0)

	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		return time.Time{}
	}

	return unixTime.In(loc)
}

// GetVersionShort returns a simple, short form of the fsoc version
func GetVersionShort() string {
	res := version.Version
	if version.Version == defaultVersion || version.IsDev {
		gitRef := version.GitBranch
		if version.GitHash != "" {
			gitRef += "@" + version.GitHash
		}
		if gitRef == "" {
			gitRef = "local"
		}
		res += " (" + gitRef + ")"
	}
	if version.GitDirty {
		res += " [dirty]"
	}

	return res
}

// GetVersionDetails provides an ordered list of version info fields
func GetVersionDetailsHuman() [][]string {
	res := [][]string{}

	// populate list using "human" tag and value of each field
	verValue := reflect.ValueOf(version)
	verType := reflect.TypeOf(version)
	verFieldNum := verValue.NumField()
	for fieldIndex := 0; fieldIndex < verFieldNum; fieldIndex++ {
		label := verType.Field(fieldIndex).Tag.Get("human")
		if label == "" {
			continue // skip fields without a label
		}

		val := verValue.Field(fieldIndex)
		valStr := ""
		if strings.HasSuffix(label, "Timestamp") {
			epoch := val.Uint()
			if epoch != 0 {
				valStr = fmt.Sprintf("%v", time.Unix(int64(epoch), 0).Local())
			}
		} else {
			switch val.Kind() {
			case reflect.Bool:
				valStr = fmt.Sprintf("%v", val.Bool())
			case reflect.Uint:
				valStr = fmt.Sprintf("%v", val.Uint())
			case reflect.String:
				valStr = val.String()
			default:
				valStr = val.String() // converts non-string types to <type X>
			}
		}
		if valStr == "" { // skip fields with empty values
			continue
		}
		res = append(res, []string{label, valStr})
	}
	return res
}

// GetVersion returns a copy of the fsoc version details
func GetVersion() VersionData {
	return version // copy
}

// Fields builds the structure required by the apex/log to log the version detail
// fields as part of a log message
func (v VersionData) Fields() log.Fields {
	var versionInfo log.Fields // map[string]interface{}
	err := mapstructure.Decode(GetVersion(), &versionInfo)
	if err != nil {
		// fallback to the short string
		log.Warnf("Cannot format version for logging: %v; using short version instead", err)
		return map[string]any{"version": GetVersionShort()}
	}

	return versionInfo
}
