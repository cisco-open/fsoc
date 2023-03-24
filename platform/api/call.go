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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/apex/log"

	"github.com/cisco-open/fsoc/cmd/config"
)

// --- Public Interface -----------------------------------------------------

type Options struct {
	Headers         map[string]string
	ResponseHeaders map[string][]string // headers as returned by the call
}

// JSONGet performs a GET request and parses the response as JSON
func JSONGet(path string, out any, options *Options) error {
	return httpRequest("GET", path, nil, out, options)
}

// JSONDelete performs a DELETE request and parses the response as JSON
func JSONDelete(path string, out any, options *Options) error {
	return httpRequest("DELETE", path, nil, out, options)
}

// JSONPost performs a POST request with JSON command and response
func JSONPost(path string, body any, out any, options *Options) error {
	return httpRequest("POST", path, body, out, options)
}

// HTTPPost performs a POST request with HTTP command and response - Accept and Content-Type headers are provided by the caller
func HTTPPost(path string, body []byte, out any, options *Options) error {
	return httpRequest("POST", path, body, out, options)
}

// HTTPGet performs a GET request with HTTP command and response - Accept and Content-Type headers are provided by the caller
func HTTPGet(path string, out any, options *Options) error {
	return httpRequest("GET", path, nil, out, options)
}

// JSONPut performs a PUT request with JSON command and response
func JSONPut(path string, body any, out any, options *Options) error {
	return httpRequest("PUT", path, body, out, options)
}

// JSONPatch performs a PATCH request and parses the response as JSON
func JSONPatch(path string, body any, out any, options *Options) error {
	return httpRequest("PATCH", path, body, out, options)
}

// JSONRequest performs an HTTP request and parses the response as JSON, allowing
// the http method to be specified
func JSONRequest(method string, path string, body any, out any, options *Options) error {
	return httpRequest(method, path, body, out, options)
}

// --- Internal methods -----------------------------------------------------

func prepareHTTPRequest(cfg *config.Context, client *http.Client, method string, path string, body any, headers map[string]string) (*http.Request, error) {
	// body will be JSONified if a body is given but no Content-Type is provided
	// (if a content type is provided, we assume the body is in the desired format)
	jsonify := body != nil && (headers == nil || headers["Content-Type"] == "")

	// prepare a body reader
	var bodyReader io.Reader = nil
	if jsonify {
		// marshal body data to JSON
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("Failed to marshal body data: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	} else if body != nil {
		// provide body data as a io.Reader
		bodyBytes, ok := body.([]byte)
		if !ok {
			return nil, fmt.Errorf("(bug) HTTP request body type must be []byte if Content-Type is provided, found %T instead", body)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	// create HTTP request
	path, query, _ := strings.Cut(path, "?")
	url, err := url.Parse(cfg.URL)
	if err != nil {
		log.Fatalf("Failed to parse the url provided in context (%q): %v", cfg.URL, err)
	}
	url.Path = path
	url.RawQuery = query
	req, err := http.NewRequest(method, url.String(), bodyReader)
	if err != nil {
		return nil, fmt.Errorf("Failed to create a request for %q: %w", url.String(), err)
	}

	// add headers that are not already provided
	if jsonify {
		contentType := "application/json"
		if method == "PATCH" {
			contentType = "application/merge-patch+json"
		}
		req.Header.Add("Content-Type", contentType)
	}
	if headers == nil || headers["Accept"] == "" {
		req.Header.Add("Accept", "application/json")
	}
	if headers == nil || headers["Authorization"] == "" {
		req.Header.Add("Authorization", "Bearer "+cfg.Token)
	}

	if cfg.AuthMethod == config.AuthMethodLocal {
		cfg.LocalAuthOptions.AddHeaders(req)
	}

	// add explicit headers
	for k, v := range headers {
		req.Header.Add(k, v)
	}

	return req, nil
}

func httpRequest(method string, path string, body any, out any, options *Options) error {
	log.WithFields(log.Fields{"method": method, "path": path}).Info("Calling FSO platform API")

	callCtx := newCallContext()
	cfg := callCtx.cfg               // quick access
	defer callCtx.stopSpinner(false) // ensure the spinner is not running when returning (belt & suspenders)

	// create a default options to avoid nil-checking
	if options == nil {
		options = &Options{}
	}

	// force login if no token
	if cfg.Token == "" {
		log.Info("No auth token available, trying to log in")
		if err := login(callCtx); err != nil {
			return err
		}
		cfg = callCtx.cfg // may have changed across login
	}

	// create http client for the request
	client := &http.Client{}

	// build HTTP request
	req, err := prepareHTTPRequest(cfg, client, method, path, body, options.Headers)
	if err != nil {
		return err // assume error messages provide sufficient info
	}

	// execute request, speculatively, assuming the auth token is valid
	callCtx.startSpinner(fmt.Sprintf("Platform API call (%v %v)", req.Method, urlDisplayPath(req.URL)))
	resp, err := client.Do(req)
	if err != nil {
		// nb: spinner will be stopped by defer
		return fmt.Errorf("%v request to %q failed: %w", method, req.URL.String(), err)
	}

	// collect response body (whether success or error)
	var respBytes []byte
	defer resp.Body.Close()
	respBytes, err = io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("Failed reading response to %v to %q (status %v): %w", method, req.URL.String(), resp.StatusCode, err)
	}

	// handle special case when access token needs to be refreshed and request retried
	if resp.StatusCode == http.StatusForbidden {
		callCtx.stopSpinnerHide()
		log.Warn("Current token is no longer valid; trying to refresh")
		err := login(callCtx)
		if err != nil {
			return fmt.Errorf("Failed to login: %w", err)
		}
		cfg = callCtx.cfg // may have changed across login

		// retry the request
		log.Info("Retrying the request with the refreshed token")
		req, err = prepareHTTPRequest(cfg, client, method, path, body, options.Headers)
		if err != nil {
			return err // error should have enough context
		}
		callCtx.startSpinner(fmt.Sprintf("Platform API call, retry after login (%v %v)", req.Method, urlDisplayPath(req.URL)))
		resp, err = client.Do(req)
		// leave the spinner until the outcome is finalized, return will stop/fail it
		if err != nil {
			return fmt.Errorf("%v request to %q failed: %w", method, req.URL.String(), err)
		}

		// collect response body (whether success or error)
		defer resp.Body.Close()
		respBytes, err = io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("Failed reading response to %v to %q (status %v): %w", method, req.URL.String(), resp.StatusCode, err)
		}
	}

	// return if API call response indicates error
	if resp.StatusCode/100 != 2 {
		callCtx.stopSpinner(false) // if still running
		log.WithFields(log.Fields{"status": resp.StatusCode}).Error("Platform API call failed")
		return parseIntoError(resp, respBytes)
	}

	// ensure spinner is stopped, API call has succeeded
	callCtx.stopSpinner(true)

	// process body
	contentType := resp.Header.Get("content-type")
	if method != "DELETE" {
		// for downloaded files, save them
		if contentType == "application/octet-stream" || contentType == "application/zip" {
			var solutionFileName = options.Headers["solutionFileName"]
			if solutionFileName == "" {
				return fmt.Errorf("(bug) filename not provided for response type %q", contentType)
			}

			// store the response data into specified file
			err := os.WriteFile(solutionFileName, respBytes, 0777)
			if err != nil {
				return fmt.Errorf("Failed to save the solution archive file as %q: %w", solutionFileName, err)
			}
		} else if len(respBytes) > 0 {
			// unmarshal response from JSON (assuming JSON data, even if the content-type is not set)
			if err := json.Unmarshal(respBytes, out); err != nil {
				return fmt.Errorf("Failed to JSON-parse the response: %w (%q)", err, respBytes)
			}
		}
	}

	// return response headers
	if options != nil {
		if resp.Header != nil {
			options.ResponseHeaders = map[string][]string(resp.Header)
		} else {
			options.ResponseHeaders = nil
		}
	}

	return nil
}

// parseError creates an error from HTTP response data
// method creates either an error with wrapped response body
// or a Problem struct in case the response is of type "application/problem+json"
func parseIntoError(resp *http.Response, respBytes []byte) error {
	// try various strategies for humanizing the error output, from the most
	// specific to the generic

	// try as "Content-Type: application/problem+json", even if
	// the content type is not set this way (some APIs don't set it)
	var problem Problem
	err := json.Unmarshal(respBytes, &problem)
	if err == nil {
		return problem
	}

	// attempt to parse response as a generic JSON object
	var errobj any
	err = json.Unmarshal(respBytes, &errobj)
	if err == nil {
		return fmt.Errorf("error response: %+v", errobj)
	}

	// fallback to string
	return fmt.Errorf("error response: %v", bytes.NewBuffer(respBytes).String())
}

// urlDisplayPath returns the URL path in a display-friendly form (may be abbreviated)
func urlDisplayPath(uri *url.URL) string {
	s := uri.Path
	if uri.RawQuery == "" {
		return s
	}
	return s + "?" + uri.RawQuery
}
