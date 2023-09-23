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

package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func parseTest(s string, valid bool) bool {
	// Check that parsing the string returns the expected result (pass/fail)
	v, err := NewVersion(s)
	//fmt.Printf("%v> %q : %v %v\n", valid, s, v, err)
	if valid != (err == nil) {
		//fmt.Printf("\tfailing: err %v expected pass: %v\n", err, valid)
		return false
	}
	if !valid { // if not expected to be valid, don't check value
		return true
	}

	// Ensure that the stringified value matches the input (always)
	if v.String() != s {
		//fmt.Printf("\tfailing due to stringified %q not matching original %q\n", s, v.String())
		return false
	}

	return true
}

func TestParsingValidVersions(t *testing.T) {
	good := []string{
		"v1", "v2", "v1beta", "v1beta2", "v2beta1", "v11beta12",
	}
	for _, v := range good {
		assert.Truef(t, parseTest(v, true), "%q is valid but failed", v)
	}
}

func TestParsingInvalidVersions(t *testing.T) {
	bad := []string{
		"1", "2", "1beta", "1beta2", "2beta1", "11beta12",
		"beta", "beta2",
		"b1", "b2beta1", "-1",
		"vabeta1",
		"v1.2", "v1.2.3", "v1zetta",
		"v1a", "v1beta-fix", "v1beta2b",
	}
	for _, v := range bad {
		assert.Truef(t, parseTest(v, false), "%q is invalid but passed", v)
	}
}
