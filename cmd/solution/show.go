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
	"os"
	"path/filepath"
	"sort"

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/output"
)

var solutionShowCmd = &cobra.Command{
	Use:     "show [-d <directory>]",
	Args:    cobra.NoArgs,
	Short:   "Show solution details",
	Long:    `Show all objects that are part of the solution in this directory`,
	Example: `  fsoc solution show`,
	Run:     solutionShow,
	Annotations: map[string]string{
		output.DetailFieldsAnnotation: "ManifestVersion:.ManifestVersion, Name:.SolutionName, Version:.SolutionVersion, Type:.SolutionType, Description:.Description, Dependencies:.Dependencies",
	},
}

type solutionManifestDisplay struct {
	ManifestVersion string
	SolutionName    string
	SolutionVersion string
	SolutionType    string
	Description     string
	Dependencies    []string
}

type solutionFileElement struct {
	Path string
	Kind string
	Type string
}

type solutionFileList struct {
	Items []solutionFileElement `json:"items"`
	Total int                   `json:"total"`
}

func getSolutionShowCmd() *cobra.Command {
	solutionShowCmd.Flags().
		StringP("directory", "d", "", "Path to the solution root directory (defaults to current dir)")

	return solutionShowCmd
}

func solutionShow(cmd *cobra.Command, args []string) {
	solutionDirectoryPath, _ := cmd.Flags().GetString("directory")

	// finalize solution path
	if solutionDirectoryPath == "" {
		currentDir, err := os.Getwd()
		if err != nil {
			log.Fatal(err.Error())
		}
		solutionDirectoryPath = currentDir
	}
	if !isSolutionPackageRoot(solutionDirectoryPath) {
		log.Fatal("Could not find solution manifest") //nb: isSolutionPackageRoot prints clear message
	}

	contents, err := NewSolutionDirectoryContentsFromDisk(solutionDirectoryPath)
	if err != nil {
		log.Fatal(err.Error())
	}

	manifest := contents.Manifest
	display := solutionManifestDisplay{
		ManifestVersion: manifest.ManifestVersion,
		SolutionName:    manifest.Name,
		SolutionVersion: manifest.SolutionVersion,
		SolutionType:    manifest.SolutionType,
		Description:     manifest.Description,
		Dependencies:    manifest.Dependencies,
	}

	_ = cmd.Flags().Set("output", "detail")
	_ = cmd.Flags().Set("fields", "ManifestVersion:.ManifestVersion, Name:.SolutionName, Version:.SolutionVersion, Type:.SolutionType, Description:.Description, Dependencies:.Dependencies")
	output.PrintCmdOutput(cmd, display)

	// create list of files
	var files []solutionFileElement
	for _, rootFile := range contents.RootFiles {
		if rootFile.FileKind == KindHidden {
			continue
		}
		files = append(files, solutionFileElement{
			Path: rootFile.Name,
			Kind: verboseFileKind(rootFile.FileKind),
			Type: rootFile.ObjectType,
		})
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})
	rootFileCount := len(files)
	for _, dir := range contents.Directories {
		for _, file := range dir.Files {
			files = append(files, solutionFileElement{
				Path: filepath.Join(dir.Name, file.Name),
				Kind: verboseFileKind(file.FileKind),
				Type: file.ObjectType,
			})
		}
	}
	sort.Slice(files[rootFileCount:], func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	_ = cmd.Flags().Set("output", "table")
	_ = cmd.Flags().Set("fields", "File:.Path, Kind:.Kind, Type:.Type")
	output.PrintCmdOutput(cmd, solutionFileList{Items: files, Total: len(files)})
}

func verboseFileKind(kind SolutionFileKinds) string {
	if kind == KindUnknown {
		return "unknown"
	} else {
		return string(kind)
	}
}
