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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/apex/log"

	"github.com/cisco-open/fsoc/cmd/config"
)

const RESOLVER_HOST = "observe-tenant-lookup-api"

// resolveTenant uses the server/url (vanity url) to obtain the tenant ID
func resolveTenant(ctx *config.Context) (string, error) {
	// find resolver endpoint based on server vanity url
	resolverUri, err := computeResolverEndpoint(ctx)
	if err != nil {
		return "", err
	}

	log.Infof("Looking up tenant ID for %v", ctx.Server)

	// create a GET HTTP request
	client := &http.Client{}
	req, err := http.NewRequest("GET", resolverUri, nil)
	if err != nil {
		return "", fmt.Errorf("Failed to create a request %q: %v", resolverUri, err.Error())
	}

	// execute request
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("GET request to %q failed: %v", req.URL, err.Error())
	}

	// log error if it occurred
	if resp.StatusCode/100 != 2 {
		// log error before trying to parse body, more processing later
		log.Errorf("Request to %q failed, status %q; more info to follow", req.URL, resp.Status)
		// fall through
	}

	// collect response body (whether success or error)
	var respBytes []byte
	defer resp.Body.Close()
	respBytes, err = io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("Failed reading response to GET to %q: %v", req.RequestURI, err.Error())
	}

	// parse response body in case of error (special parsing logic, tolerate non-JSON responses)
	if resp.StatusCode/100 != 2 {
		var errobj any

		// try to unmarshal JSON
		err := json.Unmarshal(respBytes, &errobj)
		if err != nil {
			// process as a string instead, ignore parsing error
			return "", fmt.Errorf("Error response: `%v`", bytes.NewBuffer(respBytes).String())
		}
		return "", fmt.Errorf("Error response: %+v", errobj)
	}

	// parse and update tenant ID
	var respObj tenantPayload
	if err := json.Unmarshal(respBytes, &respObj); err != nil {
		return "", fmt.Errorf("Failed to JSON parse the response as a tenant ID object: %v", err.Error())
	}

	return respObj.TenantId, nil
}

// computeResolverEndpoint figures out the URL for the tenant resolver API,
// given a tenant vanity URL
func computeResolverEndpoint(ctx *config.Context) (string, error) {
	uri, err := url.Parse("https://" + ctx.Server)
	if err != nil {
		return "", err
	}

	elements := strings.Split(uri.Host, ".") // last element may have ":<port>"
	if len(elements) != 4 {
		return "", fmt.Errorf("Cannot determine tenant resolver URI for %q, please specify --tenant on `fsoc config set`", ctx.Server)
	}

	elements[0] = RESOLVER_HOST
	// In case of production tenants, we need to manually insert "saas" into the url being calculated
	// in order to determine the lookup url correctly
	elements[1] = "saas"
	uri.Host = strings.Join(elements, ".")
	//TODO: enable for Go 1.19
	// uri = uri.JoinPath("tenants", "lookup", ctx.Server)
	uri.Path += "/tenants/lookup/" + ctx.Server // use this until Go 1.19 is supported in building
	return uri.String(), nil
}
