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

package cmdkit

import (
	"reflect"
	"strings"

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/output"
	"github.com/cisco-open/fsoc/platform/api"
)

type FetchAndPrintOptions struct {
	Method       *string           // default "GET"; "GET", "POST" and "PATCH" are currently supported
	Headers      map[string]string // http headers to send with the request
	Body         any               // body to send with the request (nil for no body)
	ResponseType *reflect.Type     // structure type to parse response into (for schema validation & fields) (nil for none)
	IsCollection bool              // set to true for GET to request a collection that may be paginated (see platform/api/collection.go)
	Filters      []string
}

// FetchAndPrint consolidates the common sequence of fetching from the server and
// displaying the output of a command in the user-selected output format. If
// a human display format is selected, the function automatically converts the value
// to one of the supported formats (within limits); if it cannot be converted, YAML is displayed instead.
// If a cmd is not provided or it has no `output` flag, human output is assumed (table)
// If a human format is requested/assumed but no table is provided, it displays YAML
// If the object cannot be converted to the desired format, shows the object in Go's %+v format
// In addition, if the fetch API command fails, this function prints the error and exits with failure.
func FetchAndPrint(cmd *cobra.Command, path string, options *FetchAndPrintOptions) {
	// finalize override fields
	method := "GET"
	if options != nil && options.Method != nil {
		method = *options.Method
	}
	var body any = nil
	if options != nil {
		body = options.Body
	}
	var httpOptions *api.Options = nil
	if options != nil {
		httpOptions = &api.Options{Headers: options.Headers}
	}
	var res any
	if options != nil && options.ResponseType != nil {
		res = reflect.New(*options.ResponseType)
	}

	// fetch data
	var err error

	if options != nil && options.Filters != nil {
		// If there are filters, apply them to query path
		numberOfFilters := len(strings.Split(path, "?"))
		if numberOfFilters != 1 && numberOfFilters != 0 {
			// Case 1: There is already a query in path append to the path
			path += "&" + strings.Join(options.Filters, "&")
		} else {
			// Case 2: There is no query in path
			path += "?" + strings.Join(options.Filters, "&")
		}
	}

	if options != nil && options.IsCollection {
		if method != "GET" {
			log.Fatalf("bug: cannot request %q for a collection at %q, only GET is supported for collections", method, path)
		}
		items, err := api.JSONGetCollection[any](path, httpOptions)
		if err != nil {
			log.Fatalf("Platform API call failed: %v", err)
		}
		res = struct {
			Items []any `json:"items"`
		}{
			Items: items,
		}

	} else {
		err = api.JSONRequest(method, path, body, &res, httpOptions)
	}
	if err != nil {
		log.Fatalf("Platform API call failed: %v", err)
	}

	// print command output data
	output.PrintCmdOutput(cmd, res)
}
