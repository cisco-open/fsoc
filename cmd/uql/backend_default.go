// Copyright 2023 Cisco Systems, Inc.
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

package uql

import (
	"encoding/json"
	"fmt"

	"github.com/apex/log"
	"github.com/pkg/errors"

	"github.com/cisco-open/fsoc/platform/api"
)

type defaultBackend struct {
	apiOptions *api.Options
}

func (b defaultBackend) Execute(query *Query, apiVersion ApiVersion) (parsedResponse, error) {
	log.WithFields(log.Fields{"query": query.Str, "apiVersion": apiVersion}).Info("executing UQL query")

	var rawJson json.RawMessage
	err := api.JSONPost(GetAPIEndpoint(apiVersion), query, &rawJson, b.apiOptions)
	if err != nil {
		if problem, ok := err.(api.Problem); ok {
			return parsedResponse{}, makeUqlProblem(problem)
		}
		return parsedResponse{}, errors.Wrap(err, fmt.Sprintf("failed to execute UQL Query: '%s'", query.Str))
	}
	var chunks []parsedChunk
	err = json.Unmarshal(rawJson, &chunks)
	if err != nil {
		return parsedResponse{}, errors.Wrap(err, fmt.Sprintf("failed to parse response for UQL Query: '%s'", query.Str))
	}
	return parsedResponse{
		chunks:  chunks,
		rawJson: &rawJson,
	}, nil
}

func (b defaultBackend) Continue(link *Link) (parsedResponse, error) {
	log.WithFields(log.Fields{"query": link.Href}).Info("continuing UQL query")

	var rawJson json.RawMessage
	err := api.JSONGet(link.Href, &rawJson, b.apiOptions)
	if err != nil {
		return parsedResponse{}, errors.Wrap(err, fmt.Sprintf("failed follow link: '%s'", link.Href))
	}
	var chunks []parsedChunk
	err = json.Unmarshal(rawJson, &chunks)
	if err != nil {
		return parsedResponse{}, errors.Wrap(err, fmt.Sprintf("failed to parse response for link: '%s'", link.Href))
	}
	return parsedResponse{
		chunks:  chunks,
		rawJson: &rawJson,
	}, nil
}

func NewDefaultBackend(options ...BackendOption) defaultBackend {
	return defaultBackend{}
}

type BackendOption func(c *defaultBackend)

func WithBackendApiOptions(options *api.Options) BackendOption {
	return func(b *defaultBackend) {
		b.apiOptions = options
	}
}
