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
	"encoding/json"
	"fmt"
	"net/http"
)

// Problem type is a json object returned for content-type application/problem+json according to the RFC-7807
type Problem struct {
	Type       string `json:"type"`
	Title      string `json:"title"`
	Detail     string `json:"detail"`
	Status     int    `json:"status"`
	Extensions map[string]any
}

func (p *Problem) UnmarshalJSON(bs []byte) (err error) {
	type _Problem Problem
	commonFields := _Problem{}

	// If the commonFields was of unaliased type Problem, this method UnmarshalJSON would be called in recursion.
	// When we try to unmarshall data to struct _Problem, the default behavior based on json tags on struct fields
	// is used instead of this method.
	if err = json.Unmarshal(bs, &commonFields); err == nil {
		*p = Problem(commonFields)
	}

	extensions := make(map[string]interface{})

	// in the second go of the unmarshalling we are parsing all undefined fields that may be part of the JSON object
	if err = json.Unmarshal(bs, &extensions); err == nil {
		delete(extensions, "type")
		delete(extensions, "title")
		delete(extensions, "detail")
		delete(extensions, "status")
		p.Extensions = extensions
	}

	return err
}

func (p Problem) Error() string {
	s := p.Title
	if s == "" && p.Type != "" {
		s = p.Type // instead of the more specific Title
	}
	if p.Detail != "" {
		if s != "" {
			s += ": " + p.Detail
		} else {
			s = p.Detail
		}
	}
	if p.Status != 0 {
		if s != "" {
			s += fmt.Sprintf(" (status %d %v)", p.Status, http.StatusText(p.Status))
		} else {
			s = fmt.Sprintf("status %d %v", p.Status, http.StatusText(p.Status))
		}
	}
	if s == "" {
		s = "no error info provided"
	}
	return s
}
