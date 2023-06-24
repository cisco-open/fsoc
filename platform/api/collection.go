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

// Package api provides access to the platform API, in all forms supported
// by the config context (aka access profile)
package api

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/apex/log"
	"github.com/peterhellberg/link"
)

const (
	linkHeaderName = "Link"
	nextRelName    = "next"
)

const MAX_COMPLETION_RESULTS = 500

type dataPage struct {
	Items []any `json:"items"`
	Total int   `json:"total"`
}

// JSONGetCollection performs a GET request and parses the response as JSON,
// handling pagination per https://www.rfc-editor.org/rfc/rfc5988,
// https://developer.cisco.com/api-guidelines/#rest-style/API.REST.STYLE.25 and
// https://developer.cisco.com/api-guidelines/#rest-style/API.REST.STYLE.24
func JSONGetCollection(path string, out any, options *Options) error {

	// ensure we can return the data
	outPtr, ok := out.(*any)
	if !ok {
		return fmt.Errorf("bug: request for collection at %q does not provide buffer for data (type %T found instead of *any)", path, out)
	}

	subOptions := Options{}
	if options != nil {
		subOptions = *options // shallow copy
	}

	var result dataPage

	var page dataPage
	var pageNo int
	for pageNo = 0; true; pageNo += 1 {
		// request collection
		err := httpRequest("GET", path, nil, &page, &subOptions)
		if err != nil {
			if pageNo > 0 {
				return fmt.Errorf("Error retrieving non-first page #%v in collection at %q: %v. All data discarded", pageNo+1, path, err)
			}
			return err
		}

		// transfer received items
		//log.Infof("Collection page #%v returned %v items, with total of %v", pageNo+1, len(page.Items), page.Total)
		if result.Items == nil {
			// initialize slice for the full result size
			result.Items = make([]any, 0, page.Total)
		}
		result.Items = append(result.Items, page.Items...)

		// break if no more pages (no response headers, no links or no next link)
		if subOptions.ResponseHeaders == nil {
			break
		}
		links, found := subOptions.ResponseHeaders[linkHeaderName]
		if !found {
			break
		}
		next, found := link.Parse(strings.Join(links, ", "))[nextRelName]
		if !found {
			break
		}

		// compute path to the next page, working around incomplete paths usually returned by APIs
		// This is done by keeping the original path up to the query string and just replacing the query string
		log.Infof("Collection page #%v at %q returned %v items and indicated that more are available at %q for a total of %v", pageNo+1, path, len(page.Items), next, page.Total)
		nextUrl, err := url.Parse(next.String())
		if err != nil {
			return fmt.Errorf("Failed to parse collection iterator link(s) %v: %v ", links, err)
		}
		nextQuery := nextUrl.RawQuery
		nextUrl, err = url.Parse(path)
		if err != nil {
			return fmt.Errorf("Failed to parse path %q: %v", path, err)
		}
		nextUrl.RawQuery = nextQuery
		path = nextUrl.String()
	}
	log.Infof("Collection page #%v at %q returned %v items (last page)", pageNo+1, path, len(page.Items))

	result.Total = len(result.Items)
	if result.Total != page.Total {
		log.Warnf("Collection at %q returned %v items vs. expected %v items", path, result.Total, page.Total)
	}
	*outPtr = &result

	return nil
}
