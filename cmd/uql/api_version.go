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

package uql

import (
	"fmt"
	"slices"
	"strings"
)

// UQL API version type, supporting a limited set of values.
// Implements the StringEnumer interface defined by the fsoc config package in order
// to support parsing apiver from an fsoc config file.
type ApiVersion string

// constants for direct use
const (
	ApiVersion1 ApiVersion = ApiVersion("v1")
	//ApiVersion2Beta ApiVersion = ApiVersion("v2beta")

	ApiVersionDefault ApiVersion = ApiVersion1
)

// note: when changing the set of supported values and/or the default,
// don't forget to update the fsoc-help tag in the GlobalConfig (see uql.go)

var supportedApiVersions = []string{
	string(ApiVersion1),
	//string(ApiVersion2Beta),
}

func (a *ApiVersion) ValidateAndSet(v any) error {
	s, ok := v.(string)
	if !ok {
		return fmt.Errorf("the API version value must be a string, found %T instead", v)
	}
	if !slices.Contains(supportedApiVersions, s) {
		return fmt.Errorf(`API version %q is not supported; valid value(s): "%v"`, v, strings.Join(supportedApiVersions, `", "`))
	}
	*a = ApiVersion(s)
	return nil
}

func (a *ApiVersion) String() string {
	if a == nil || string(*a) == "" {
		return string(ApiVersionDefault)
	} else {
		return string(*a)
	}
}

func GetAPIEndpoint(apiVersion ApiVersion) string {
	return fmt.Sprintf("/monitoring/%v/query/execute", apiVersion)
}
