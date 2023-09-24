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
	"fmt"
	"reflect"
	"regexp"

	"github.com/mitchellh/mapstructure"

	"github.com/cisco-open/fsoc/config"
)

var versionRegExp = regexp.MustCompile(`v\d+(beta(\d+)?)?$`)

func init() {
	config.RegisterTypeDecodeHooks(versionDecodeHookFunc())
}

// NewVersion parses a string value into an API version, ensuring that the
// string matches the required pattern
func NewVersion(s string) (Version, error) {
	ok := versionRegExp.MatchString(s)
	if !ok {
		return "", fmt.Errorf(`API version %q does not match the required pattern, vN[beta[M]], where N and M are integers`, s)
	}
	return Version(s), nil
}

// String converts an API version to string, implementing the Stringer interface
func (v *Version) String() string {
	return string(*v)
}

func versionDecodeHookFunc() mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{}) (interface{}, error) {
		// return if not from string or not to api.Version
		if f.Kind() != reflect.String {
			return data, nil
		}
		if t != reflect.TypeOf(Version("v1")) {
			return data, nil
		}

		// parse, returning the tuple (value, err)
		return NewVersion(data.(string))
	}
}
