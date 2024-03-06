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

// JSONGetCollection performs a GET request and parses the response as JSON,
// handling pagination per https://www.rfc-editor.org/rfc/rfc5988,
// https://developer.cisco.com/api-guidelines/#rest-style/API.REST.STYLE.25 and
// https://developer.cisco.com/api-guidelines/#rest-style/API.REST.STYLE.24
func JSONGetCollection[T any](path string, out *CollectionResult[T], options *Options) (err error) {

	subOptions := Options{}
	if options != nil {
		subOptions = *options // shallow copy
	}

	var pageNo, pageItemsCount, pageTotalCount int
	for pageNo = 0; true; pageNo += 1 {
		var page CollectionResult[T]
		// request collection
		err := httpRequest("GET", path, nil, &page, &subOptions)
		if err != nil {
			if pageNo > 0 {
				return fmt.Errorf("Error retrieving non-first page #%v in collection at %q: %v. All data discarded", pageNo+1, path, err)
			}
			return err
		}

		// handle case where out.Items is uninitialized (nil) and page.Items is an initalized but empty slice
		// append results in a nil slice instead of an empty slice in this case
		if out.Items == nil && page.Items != nil {
			out.Items = page.Items
		} else {
			out.Items = append(out.Items, page.Items...)
		}

		pageItemsCount = len(page.Items)
		pageTotalCount = page.Total

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
			return fmt.Errorf("failed to parse collection iterator link(s) %v: %v ", links, err)
		}
		nextQuery := nextUrl.RawQuery
		nextUrl, err = url.Parse(path)
		if err != nil {
			return fmt.Errorf("failed to parse path %q: %v", path, err)
		}
		nextUrl.RawQuery = nextQuery
		path = nextUrl.String()
	}
	log.Infof("Collection page #%v at %q returned %v items (last page)", pageNo+1, path, pageItemsCount)

	out.Total = len(out.Items)
	if out.Total != pageTotalCount {
		log.Warnf("Collection at %q returned %v items vs. expected %v items", path, out.Total, pageTotalCount)
	}

	return nil
}
