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

type UqlClient interface {
	// ExecuteQuery sends an execute request to the UQL service
	ExecuteQuery(query *Query) (*Response, error)

	// ContinueQuery sends a continue request to the UQL service
	ContinueQuery(dataSet *DataSet, rel string) (*Response, error)
}

type defaultClient struct {
	backend    uqlService
	apiVersion *ApiVersion
}

type UqlClientOption func(c *defaultClient)

func NewClient(options ...UqlClientOption) UqlClient {
	client := &defaultClient{backend: &defaultBackend{}}
	for _, option := range options {
		option(client)
	}
	return client
}

func WithClientApiVersion(version ApiVersion) UqlClientOption {
	return func(c *defaultClient) {
		c.apiVersion = &version
	}
}

func (c defaultClient) ExecuteQuery(query *Query) (*Response, error) {
	apiVersion := ApiVersion("")
	if c.apiVersion != nil {
		apiVersion = *c.apiVersion
	}
	return executeUqlQuery(query, apiVersion, c.backend)
}

func (c defaultClient) ContinueQuery(dataSet *DataSet, rel string) (*Response, error) {
	return continueUqlQuery(dataSet, rel, c.backend)
}
