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
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/output"
)

type SolutionDirectoryContents struct {
	Manifest    Manifest
	Directories []SolutionSubDirectory
	RootFiles   []SolutionFile // any files in the root dir except the manifest
}

type SolutionDirectoryContentsType string

const (
	KnowledgeTypes SolutionDirectoryContentsType = "types"
	ObjectsDir     SolutionDirectoryContentsType = "objectsDir"
	ObjectsFile    SolutionDirectoryContentsType = "objectsFile"
)

type SolutionSubDirectory struct {
	Name        string // full name relative to the solution root
	Type        SolutionDirectoryContentsType
	ObjectsType string // used only for ObjecsDir directory types
	Files       []SolutionFile
}

type SolutionFileEncoding string

const (
	EncodingUnknown SolutionFileEncoding = ""
	EncodingJSON    SolutionFileEncoding = "json"
	EncodingYAML    SolutionFileEncoding = "yaml"
)

type SolutionFileKinds string

const (
	KindUnknown       SolutionFileKinds = ""
	KindKnowledgeType SolutionFileKinds = "knowledge type"
	KindObjectType    SolutionFileKinds = "object"
)

type SolutionFile struct {
	Name       string // file name relative to the directory
	FileKind   SolutionFileKinds
	ObjectType string // empty if not known or not KindObjectType
	Encoding   SolutionFileEncoding
	Contents   bytes.Buffer
}

var extensionMap = map[string]SolutionFileEncoding{
	".json": EncodingJSON,
	".yaml": EncodingYAML,
	".yml":  EncodingYAML,
}

// New creates a new SolutionDirectoryContents object with a simple manifest
func NewSolutionDirectoryContents(name string, solutionType SolutionType) (*SolutionDirectoryContents, error) {
	manifest := *createInitialSolutionManifest(name, string(solutionType))
	return &SolutionDirectoryContents{
		Manifest: manifest,
	}, nil
}

// Read reads the full contents of the solution directory from the specified root
func NewSolutionDirectoryContentsFromDisk(path string) (*SolutionDirectoryContents, error) {
	// init empty object
	s := SolutionDirectoryContents{}

	// get absolute path (deals with relative paths, .., ~, etc.)
	rootPath := absolutizePath(path)

	// read manifest
	manifest, err := getSolutionManifest(path)
	if err != nil {
		return nil, err
	}
	s.Manifest = *manifest
	s.Directories = make([]SolutionSubDirectory, 0)
	s.RootFiles = make([]SolutionFile, 0)

	// read contents
	err = s.readContents(rootPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read solution objects: %w", err)
	}

	// process objects from the manifest
	err = s.annotateFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to process solution objects: %w", err)
	}

	return &s, nil
}

// Write writes the full contents of the solution directory to the specified path
// If overwrite is true, it will overwrite the existing directory; otherwise it will
// require that the directory is either empty or does not exist (it will create it)
func (s *SolutionDirectoryContents) Write(path string, overwrite bool) error {
	panic("not implemented")
}

// Dump displays the contents of the solution contents object
func (s *SolutionDirectoryContents) Dump(cmd *cobra.Command) {
	t := output.Table{
		Headers: []string{"Solution Name", "Solution Version", "Solution Type", "Manifest Version", "Description"},
		Lines: [][]string{
			{s.Manifest.Name},
			{s.Manifest.SolutionVersion},
			{string(s.Manifest.SolutionType)},
			{s.Manifest.ManifestVersion},
			{s.Manifest.Description},
		},
		Detail: true,
	}
	output.PrintCmdOutputCustom(cmd, nil, &t)
}

// --- Internal methods

// readContents reads the contents of the solution directory from the specified root
func (s *SolutionDirectoryContents) readContents(rootPath string) error {
	// read directories & files
	err := filepath.Walk(rootPath,
		func(path string, info os.FileInfo, err error) error {
			fmt.Printf("Walking %q dir=%v err=%v\n", path, info.IsDir(), err)
			if err != nil {
				return err
			}
			if path == rootPath {
				return nil // skip the root directory itself
			}
			entryType := map[bool]string{true: "directory", false: "file"}[info.IsDir()]
			relPath, err := filepath.Rel(rootPath, path)
			if err != nil {
				return fmt.Errorf("found %v %q that not under the root %q: %v", entryType, path, rootPath, err)
			}
			if strings.HasPrefix(relPath, ".") {
				return fmt.Errorf("found %v %q that not under the root %q: %q", entryType, path, rootPath, relPath)
			}
			if !isAllowedPath(path, info) {
				log.Warnf("Found %v %q which cannot be bundled; it will still be included", entryType, relPath)
			}

			if info.IsDir() {
				dir := SolutionSubDirectory{
					Name:  relPath,
					Files: make([]SolutionFile, 0),
				}
				s.Directories = append(s.Directories, dir)
				fmt.Printf("Appended a directory %#v\n", dir)
			} else {
				contents, err := os.ReadFile(path)
				if err != nil {
					return fmt.Errorf("error reading file %q: %w", relPath, err)
				}
				dirName, name := filepath.Split(relPath)
				dirName = filepath.Clean(dirName) // removes the trailing separator
				encoding := extensionMap[filepath.Ext(name)]
				file := SolutionFile{
					Name:     name,
					Encoding: encoding,
					Contents: *bytes.NewBuffer(contents),
				}
				if dirName == "" || dirName == "." {
					s.RootFiles = append(s.RootFiles, file)
					fmt.Printf("Appended a root file %#v\n", file)
				} else {
					// find directory
					found := false
					for _, dir := range s.Directories {
						if dir.Name == dirName {
							found = true
							dir.Files = append(dir.Files, file)
							fmt.Printf("Appended file %#v to directory %#v\n", file, dir)
							break
						}
					}
					if !found {
						return fmt.Errorf("file %q is in directory %q which was not found", name, dirName)
					}
				}
			}
			return nil
		})
	if err != nil {
		return fmt.Errorf("error walking the solution directory: %w", err)
	}

	return nil
}

// annotateFiles annotates the files with the object type based on the manifest
func (s *SolutionDirectoryContents) annotateFiles() error {
	// process types in the manifest
	for _, typeFile := range s.Manifest.Types {
		err := s.annotateFile(typeFile, KindKnowledgeType, "")
		if err != nil {
			return fmt.Errorf("failed to annotate knowledge type file %q: %w", typeFile, err)
		}
	}

	// process objects in the manifest
	for _, objectDef := range s.Manifest.Objects {
		// ensure that the object definition is valid
		nDescriptions := 0
		if objectDef.ObjectsDir != "" {
			nDescriptions++
		}
		if objectDef.ObjectsFile != "" {
			nDescriptions++
		}
		switch nDescriptions {
		case 0:
			return fmt.Errorf("object %q must have either objectsDir or objectsFile specified", objectDef.Type)
		case 1:
			// good, continue
		default:
			return fmt.Errorf("object %q must have only one of objectsDir or objectsFile", objectDef.Type)
		}

		if objectDef.ObjectsFile != "" {
			err := s.annotateFile(objectDef.ObjectsFile, KindObjectType, objectDef.Type)
			if err != nil {
				return fmt.Errorf("failed to annotate object file %q: %w", objectDef.ObjectsFile, err)
			}
		} else {
			err := s.annotateDirectory(objectDef.ObjectsDir, KindObjectType, objectDef.Type)
			if err != nil {
				return fmt.Errorf("failed to annotate object directory %q: %w", objectDef.ObjectsDir, err)
			}
		}
	}
	return nil
}

func (s *SolutionDirectoryContents) annotateFile(name string, kind SolutionFileKinds, objectType string) error {
	var file *SolutionFile

	dirName, fileName := filepath.Split(name)
	if dirName == "" {
		log.Warnf("File %q for %v %v is in the root directory; it should be in a subdirectory", name, kind, objectType)

		// find the file
		for _, f := range s.RootFiles {
			if f.Name == name {
				file = &f
				break
			}
		}
	} else {
		// find the directory & file
		found := false
		for _, dir := range s.Directories {
			if dir.Name == dirName {
				found = true
				for _, f := range dir.Files {
					if f.Name == fileName {
						file = &f
						break
					}
				}
				break
			}
		}
		if !found {
			return fmt.Errorf("directory %q for %v %v was not found", dirName, kind, objectType)
		}
	}
	if file == nil {
		return fmt.Errorf("file %q for %v %v was not found", name, kind, objectType)
	}

	// annotate file
	file.FileKind = kind
	file.ObjectType = objectType

	return nil
}

func (s *SolutionDirectoryContents) annotateDirectory(name string, kind SolutionFileKinds, objectType string) error {
	if kind != KindObjectType {
		panic(fmt.Sprintf("bug: annotateDirectory for %q is called for %v but allowed only for objects", name, kind))
	}

	// find the directory
	var dir *SolutionSubDirectory
	for _, d := range s.Directories {
		if d.Name == name {
			dir = &d
			break
		}
	}
	if dir == nil {
		return fmt.Errorf("directory %q for %v %v was not found", name, kind, objectType)
	}

	// propagate object type to directory and files
	dir.Type = ObjectsDir
	dir.ObjectsType = objectType
	for _, f := range dir.Files {
		f.FileKind = kind
		f.ObjectType = objectType
	}

	return nil
}
