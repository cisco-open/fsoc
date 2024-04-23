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

package solution

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/apex/log"
	"gopkg.in/yaml.v2"

	"github.com/cisco-open/fsoc/output"
)

type editCallback func(filePath string, key string, value string, nReplacements *int) (string, error)

// replaceValuesInBuffer replaces file contents' values that match a given
// regexp, using a callback to modify the matched value.
// The directory is provided as a context and may be nil for root files.
// The file encoding must be YAML or JSON (or an error is returned).
// The function modifies the buffer (if it made any changes); it returns the number of
// replacements made or an error.
func ReplaceValuesInFileBuffer(file *SolutionFile, dir *SolutionSubDirectory, matchRe *regexp.Regexp, callback editCallback) (int, error) {
	filePath := file.Name
	if dir != nil {
		filePath = dir.Name + "/" + file.Name
	}

	// decode buffer to map[string]interace{}
	var contents any
	var err error
	switch file.Encoding {
	case EncodingJSON:
		err = json.Unmarshal(file.Contents.Bytes(), &contents)
	case EncodingYAML:
		err = yaml.Unmarshal(file.Contents.Bytes(), &contents)
	default:
		return 0, fmt.Errorf("unsupported encoding: %v", file.Encoding)
	}
	if err != nil {
		return 0, fmt.Errorf("error decoding %v file: %w", file.Encoding, err)
	}

	// replace solution name in values, starting from the root base key ("")
	nReplacements := 0
	contents, err = replaceValuesRecursively(filePath, contents, matchRe, "", callback, &nReplacements)
	if err != nil {
		return 0, fmt.Errorf("error replacing values: %w", err)
	}

	// return original buffer if no replacements were made
	if nReplacements == 0 {
		return 0, nil
	}

	// encode map back to buffer
	var newBuffer bytes.Buffer
	switch file.Encoding {
	case EncodingJSON:
		enc := json.NewEncoder(&newBuffer)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", output.JsonIndent)
		err = enc.Encode(contents)
	case EncodingYAML:
		err = yaml.NewEncoder(&newBuffer).Encode(contents)
	}
	if err != nil {
		return nReplacements, fmt.Errorf("error re-encoding %v file with %v modifications: %w", file.Encoding, nReplacements, err)
	}

	// update file contents
	file.Contents = newBuffer

	return nReplacements, nil
}

func replaceValuesRecursively(filePath string, contents any, matchRe *regexp.Regexp, base string, callback editCallback, nReplacements *int) (any, error) {
	switch val := contents.(type) {
	case map[string]any:
		for k, v := range val {
			var key string
			if base == "" {
				key = k
			} else {
				key = base + "." + k
			}
			newVal, err := replaceValuesRecursively(filePath, v, matchRe, key, callback, nReplacements)
			if err != nil {
				return nil, err
			}
			val[k] = newVal
		}
	case []any:
		for i, v := range val {
			key := base + fmt.Sprintf("[%d]", i)
			newVal, err := replaceValuesRecursively(filePath, v, matchRe, key, callback, nReplacements)
			if err != nil {
				return nil, err
			}
			val[i] = newVal
		}
	case string:
		// skip value if it doesn't match the target expression
		if !matchRe.MatchString(val) {
			break
		}

		// invoke callback to make replacements
		replacements := 0
		newVal, err := callback(filePath, base, val, &replacements)
		if err != nil {
			return nil, err
		}
		if replacements > 0 {
			*nReplacements += replacements
			log.WithFields(log.Fields{
				"key": base,
				"old": val,
				"new": newVal,
			}).Info("Replaced solution name")
			contents = newVal
		}
	default:
		// other value types are not modified
	}

	return contents, nil
}
