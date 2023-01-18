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
	return jsonRequest("GET", path, nil, out, options)
}

// JSONGet performs a GET request and parses the response as JSON
func JSONDelete(path string, out any, options *Options) error {
	return jsonRequest("DELETE", path, nil, out, options)
}

// JSONPost performs a POST request with JSON command and response
func JSONPost(path string, body any, out any, options *Options) error {
	return jsonRequest("POST", path, body, out, options)
}

// HTTPPost performs a POST request with HTTP command and response - Accept and Content-Type headers are provided by the caller
func HTTPPost(path string, body []byte, out any, options *Options) error {
	return httpRequest("POST", path, body, out, options)
}

// HTTPGet performs a GET request with HTTP command and response - Accept and Content-Type headers are provided by the caller
func HTTPGet(path string, out any, options *Options) error {
	return httpRequest("GET", path, nil, out, options)
}

// JSONPost performs a POST request with JSON command and response
func JSONPut(path string, body any, out any, options *Options) error {
	return jsonRequest("PUT", path, body, out, options)
}

// JSONPatch performs a PATCH request and parses the response as JSON
func JSONPatch(path string, body any, out any, options *Options) error {
	return jsonRequest("PATCH", path, body, out, options)
}

// JSONPatch performs a http request and parses the response as JSON, allowing
// the http method to be specified
func JSONRequest(method string, path string, body any, out any, options *Options) error {
	return jsonRequest(method, path, body, out, options)
}

// --- Internal methods -----------------------------------------------------

func jsonRequest(method string, path string, body any, out any, options *Options) error {
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
	log.WithFields(log.Fields{"context": cfg.Name, "server": cfg.Server, "tenant": cfg.Tenant}).Info("Using context")

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
	req, err := prepareJSONRequest(cfg, client, method, path, body, options.Headers)
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
		req, err = prepareJSONRequest(cfg, client, method, path, body, options.Headers)
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

	// parse response body in case of error (special parsing logic, tolerate non-JSON responses)
	if resp.StatusCode/100 != 2 {
		var errobj any

		// try to unmarshal JSON
		err := json.Unmarshal(respBytes, &errobj)
		if err != nil {
			// process as a string instead, ignore parsing error
			errobj = bytes.NewBuffer(respBytes).String()
		}
		return fmt.Errorf("Error response: %+v", errobj)
	}

	if method != "DELETE" {
		if err := json.Unmarshal(respBytes, out); err != nil {
			return fmt.Errorf("Failed to JSON parse the response: %v (%q)", err, respBytes)
		}
		//log.Infof("API Response as struct %+v\n", out) //@@
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

func prepareJSONRequest(cfg *config.Context, client *http.Client, method string, path string, body any, headers map[string]string) (*http.Request, error) {
	// marshal body data into a io.Reader
	var bodyReader io.Reader = nil
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			log.Errorf("Failed to marshal data: %v", err.Error())
			return nil, err
		}
		//log.Infof("HTTP Request body: %q", bodyBytes)
		bodyReader = bytes.NewReader(bodyBytes)
	}

	// create a HTTP request
	url := &url.URL{
		Scheme: "https",
		Host:   cfg.Server,
		Path:   path,
	}

	// checking if there is a query in the path
	iQuery := strings.Index(path, "?")

	if iQuery > -1 {
		// extracting the query and updating the path

		purePath := path[:iQuery]
		query := path[iQuery+1:]
		url.RawQuery = query
		url.Path = purePath
	}

	req, err := http.NewRequest(method, url.String(), bodyReader)
	if err != nil {
		log.Errorf("Failed to create a request %q: %v", url.String(), err.Error())
		return nil, err
	}

	// add headers
	req.Header.Add("Accept", "application/json")

	// set Content-Type in case it hasn't been provided by the caller

	if headers["Content-Type"] == "" {
		contentType := "application/json"
		if method == "PATCH" {
			contentType = "application/merge-patch+json"
		}
		req.Header.Add("Content-Type", contentType)
	}

	req.Header.Add("Authorization", "Bearer "+cfg.Token)

	for k, v := range headers {
		req.Header.Add(k, v)
	}

	return req, nil
}

func httpRequest(method string, path string, body []byte, out any, options *Options) error {
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
	log.WithFields(log.Fields{"context": cfg.Name, "server": cfg.Server, "tenant": cfg.Tenant}).Info("Using context")

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

	// parse response body in case of error (special parsing logic, tolerate non-JSON responses)
	if resp.StatusCode/100 != 2 {
		var errobj any
		// try to unmarshal JSON
		err := json.Unmarshal(respBytes, &errobj)
		if err != nil {
			// process as a string instead, ignore parsing error
			errobj = bytes.NewBuffer(respBytes).String()
		}
		return fmt.Errorf("Error response: %+v", errobj)
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
	return nil
}

func prepareHTTPRequest(cfg *config.Context, client *http.Client, method string, path string, body []byte, headers map[string]string) (*http.Request, error) {
	// marshal body data into a io.Reader
	bodyReader := bytes.NewReader(body)

	// if body != nil {
	// 	bodyBytes, err := json.Marshal(body)
	// 	if err != nil {
	// 		log.Errorf("Failed to marshal data: %v", err.Error())
	// 		return nil, err
	// 	}
	// 	//log.Infof("HTTP Request body: %q", bodyBytes)
	// 	bodyReader = bytes.NewReader(bodyBytes)
	// }

	// create a HTTP request
	url := &url.URL{
		Scheme: "https",
		Host:   cfg.Server,
		Path:   path,
	}

	// checking if there is a query in the path
	iQuery := strings.Index(path, "?")

	if iQuery > -1 {
		// extracting the query and updating the path

		purePath := path[:iQuery]
		query := path[iQuery+1:]
		url.RawQuery = query
		url.Path = purePath
	}

	req, err := http.NewRequest(method, url.String(), bodyReader)
	if err != nil {
		log.Errorf("Failed to create a request %q: %v", url.String(), err.Error())
		return nil, err
	}

	req.Header.Add("Authorization", "Bearer "+cfg.Token)

	for k, v := range headers {
		req.Header.Add(k, v)
	}

	return req, nil
}
