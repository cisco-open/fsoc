// Copyright 2024 Cisco Systems, Inc.
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
	"fmt"
	"regexp"
	"strings"

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/output"
)

const nativeIsolationSolutionId = "@INSTALL ${$sys.solutionId}"

func fixIsolate(cmd *cobra.Command, contents *SolutionDirectoryContents) error {
	// construct regexp to match solution name as a separate word (after removing
	// the pseudo-isolation suffix, if any)
	solutionName, pseudoIsolated := strings.CutSuffix(contents.Manifest.Name, pseudoIsolationSuffix)
	if !IsValidSolutionName(solutionName) {
		return fmt.Errorf("invalid solution name: %v", solutionName)
	}
	matchRe := regexp.MustCompile(fmt.Sprintf(`(\b%s\b|%s)`, solutionName, regexp.QuoteMeta(pseudoIsolationSolutionId)))

	// define callback closure
	editValue := func(filePath string, key string, value string, nReplacements *int) (string, error) {
		oldValue := value

		// TODO:
		// partial: add support for matches where the name is part of but not the whole value
		if value == solutionName || value == pseudoIsolationSolutionId {
			value = nativeIsolationSolutionId
		}
		if oldValue != value && nReplacements != nil {
			*nReplacements++
		}
		output.PrintCmdStatus(cmd, fmt.Sprintf("%q %s: %q -> %q\n", filePath, key, oldValue, value))
		return value, nil
	}

	// --- Update manifest
	manifestReplacements := 0

	// fix solution name
	if pseudoIsolated {
		// remove pseudo-isolation
		contents.Manifest.Name = solutionName
		manifestReplacements++
	}

	// update object types
	for i := range contents.Manifest.Objects {
		obj := &contents.Manifest.Objects[i]
		if matchRe.MatchString(obj.Type) {
			var err error
			obj.Type, err = editValue("manifest."+contents.Manifest.ManifestFormat.String(),
				fmt.Sprintf("objects[%d].type", i),
				obj.Type,
				&manifestReplacements)
			if err != nil {
				log.Fatalf("Error replacing type %q in manifest: %v", obj.Type, err)
			}
		}
	}
	if manifestReplacements > 0 {
		log.WithFields(log.Fields{
			"nReplacements": manifestReplacements,
		}).Info("replaced values in manifest")
	}

	// --- Update files
	nFilesChanged := 0
	err := contents.WalkFiles(func(file *SolutionFile, dir *SolutionSubDirectory) error {
		// construct file path for display/warnings
		filePath := file.Name
		if dir != nil {
			filePath = dir.Name + "/" + file.Name
		}

		// skip hidden files
		if file.FileKind == KindHidden {
			return nil
		}

		// skip file if not a json or yaml file
		if file.Encoding != EncodingJSON && file.Encoding != EncodingYAML {
			log.Warnf("skipping non-json/yaml file %v", filePath)
			return nil
		}

		// TODO: decide whether to skip files that are not type or object

		// edit values that contain the solution name as a separate word
		nReplacements, err := ReplaceValuesInFileBuffer(file, dir, matchRe, editValue)
		if err != nil {
			return fmt.Errorf("error replacing values in %v: %w", filePath, err)
		}
		if nReplacements > 0 {
			nFilesChanged++
			log.WithFields(log.Fields{
				"file":          filePath,
				"nReplacements": nReplacements,
			}).Info("replaced values in file")
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("error processing solution files: %w", err)
	}

	return nil
}

func fixDeisolate(cmd *cobra.Command, contents *SolutionDirectoryContents) error {
	return fmt.Errorf("fixDeisolate not implemented")
}
