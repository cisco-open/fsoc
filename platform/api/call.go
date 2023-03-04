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
	"errors"
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
			log.Errorf("Failed to marshal data: %v", err.Error())
			return nil, err
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
		log.Errorf("Failed to create a request %q: %v", url.String(), err.Error())
		return nil, err
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

	// add explicit headers
	for k, v := range headers {
		req.Header.Add(k, v)
	}

	return req, nil
}

func httpRequest(method string, path string, body any, out any, options *Options) error {
	log.WithFields(log.Fields{"method": method, "path": path}).Info("Calling FSO platform API")

	// create a default options to avoid nil-checking
	if options == nil {
		options = &Options{}
	}

	// get current context to obtain the URL and token (TODO: consider supporting unauth access for local dev)
	cfg := config.GetCurrentContext()
	if cfg == nil {
		return errors.New("Missing context; use 'fsoc config set' to configure your context")
	}
	log.WithFields(log.Fields{"context": cfg.Name, "url": cfg.URL, "tenant": cfg.Tenant}).Info("Using context")

	// force login if no token
	if cfg.Token == "" {
		log.Infof("No token available, trying to log in")
		if err := Login(); err != nil {
			return err
		}
		cfg = config.GetCurrentContext()
		if cfg.Token == "" {
			return errors.New("Login succeeded but did not provide a token")
		}
	}

	// create http client for the request
	client := &http.Client{}

	// build HTTP request
	req, err := prepareHTTPRequest(cfg, client, method, path, body, options.Headers)
	if err != nil {
		return err // anything that needed logging has been logged
	}

	// execute request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("%v request to %q failed: %v", method, req.RequestURI, err)
	}

	// log error if it occurred
	if resp.StatusCode/100 != 2 {
		// log error before trying to parse body, more processing later
		log.Errorf("Request failed, status %q; more info to follow", resp.Status)
	}

	// collect response body (whether success or error)
	var respBytes []byte
	defer resp.Body.Close()
	respBytes, err = io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("Failed reading response to %v to %q: %v", method, req.RequestURI, err)
	}

	// handle special case when access token needs to be refreshed and request retried
	if resp.StatusCode == http.StatusForbidden {
		log.Info("Current token is no longer valid; trying to refresh")
		err := Login()
		if err != nil {
			// nb: sufficient logging from login should have occurred
			return err
		}

		// re-load context, including refreshed token
		cfg = config.GetCurrentContext()

		// retry the request
		log.Info("Retrying the request with the refreshed token")
		req, err = prepareHTTPRequest(cfg, client, method, path, body, options.Headers)
		if err != nil {
			return err // anything that needed logging has been logged
		}
		resp, err = client.Do(req)
		if err != nil {
			return fmt.Errorf("%v request to %q failed: %v", method, req.RequestURI, err.Error())
		}

		// log error if it occurred
		if resp.StatusCode/100 != 2 {
			// log error before trying to parse body, more processing later
			log.Errorf("Request failed, status %q; more info to follow", resp.Status)
		} else {
			log.Infof("Request completed successfully: %v", resp.Status)
		}

		// collect response body (whether success or error)
		defer resp.Body.Close()
		respBytes, err = io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("Failed reading response to %v to %q: %v", method, req.RequestURI, err)
		}
	}

	if resp.StatusCode/100 != 2 {
		return parseIntoError(resp, respBytes)
	}

	contentType := resp.Header.Get("content-type")
	if method != "DELETE" {
		// when command type is not SOLUTION_DOWNLOAD, parse the response to be JSON
		if contentType != "application/octet-stream" && contentType != "application/zip" {
			if err := json.Unmarshal(respBytes, out); err != nil {
				return fmt.Errorf("Failed to JSON parse the response: %v (%q)", err, respBytes)
			}
		} else {
			var solutionFileName = options.Headers["solutionFileName"]
			// zip the buffer data to a zip with solution name in current directory
			err := os.WriteFile(solutionFileName, respBytes, 0777)
			if err != nil {
				log.Fatalf("Failed to parse the Solution download API buffer response: %v (%q)", err, respBytes)
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
