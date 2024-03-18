// Copyright 2024 Cisco Systems, Inc.
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

package provisioning

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/exp/slices"
)

// Tenant Provisioning API version type, supporting a limited set of values.
// Implements the StringEnumer interface defined by the fsoc config package in order
// to support parsing apiver from an fsoc config file.
type ApiVersion string

const (
	ApiVersionDefault ApiVersion = ApiVersion("v1beta")
)

var supportedApiVersions = []string{
	string(ApiVersionDefault),
}

func (a *ApiVersion) ValidateAndSet(version any) error {
	s, ok := version.(string)
	if !ok {
		return errors.New(fmt.Sprintf(`the API version value must be a string, found %T instead`, version))
	}
	if !slices.Contains(supportedApiVersions, s) {
		return errors.New(fmt.Sprintf(`API version %q is not supported; valid value(s): "%v"`, version, strings.Join(supportedApiVersions, `", "`)))
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

func getBaseUrl() string {
	var version ApiVersion
	if GlobalConfig.ApiVersion != nil && *GlobalConfig.ApiVersion != "" {
		version = *GlobalConfig.ApiVersion // version from config file
	} else {
		version = ApiVersionDefault
	}
	return fmt.Sprintf("/provisioning/%v", version)
}

func getTenantLookupUrl(vanityUrl string) string {
	return fmt.Sprintf("%v/tenants/lookup/vanityUrl/%v", getBaseUrl(), vanityUrl)
}
