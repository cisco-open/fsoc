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

package api

// Version defines an API version, as used in URI paths. Use NewVersion() to
// create/parse from a string value and String() to convert back to string
type Version string

// CollectionResult is a structure that wraps API collections of type T.
// See JSONGetCollection for reference to API collection RFC/standards
type CollectionResult[T any] struct {
	Items []T `json:"items"`
	Total int `json:"total"`
}
