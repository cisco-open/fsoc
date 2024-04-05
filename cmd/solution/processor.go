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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/apex/log"
	"github.com/spf13/afero"
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

func (s SolutionSubDirectory) String() string {
	return fmt.Sprintf("%q (type %q, objects type %q, %v files)", s.Name, s.Type, s.ObjectsType, len(s.Files))
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
	KindHidden        SolutionFileKinds = "hidden"
)

type SolutionFile struct {
	Name       string // file name relative to the directory
	FileKind   SolutionFileKinds
	ObjectType string // empty if not known or not KindObjectType
	Encoding   SolutionFileEncoding
	Contents   bytes.Buffer
}

func (s SolutionFile) String() string {
	return fmt.Sprintf("%q (kind %q, object type %q, format %q)", s.Name, s.FileKind, s.ObjectType, s.Encoding)
}

var extensionMap = map[string]SolutionFileEncoding{
	".json": EncodingJSON,
	".yaml": EncodingYAML,
	".yml":  EncodingYAML,
}

// NewSolutionDirectoryContents creates a new SolutionDirectoryContents object with a simple manifest
func NewSolutionDirectoryContents(name string, solutionType SolutionType) (*SolutionDirectoryContents, error) {
	manifest := *createInitialSolutionManifest(name, WithSolutionType(string(solutionType))) // TODO: streamline WithSolutionType to use SolutionType
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

// Write writes the full contents of the solution directory to the specified filesystem.
// Using Afero's filesystem abstraction allows for safe writing (as well as testing).
// The directory must be empty, otherwise an error is returned.
func (s *SolutionDirectoryContents) Write(fs afero.Fs) error {
	// check if the directory is empty
	empty, err := afero.IsEmpty(fs, ".")
	if err != nil {
		return fmt.Errorf("failed to check if the directory is empty: %w", err)
	}
	if !empty {
		return fmt.Errorf("target directory is not empty; it must be empty to write the solution")
	}

	// write the root files
	for _, file := range s.RootFiles {
		// skip manifest, it's written separately
		if file.Name == "manifest.json" || file.Name == "manifest.yaml" || file.Name == "manifest.yml" {
			continue
		}

		// write the file
		err := afero.WriteFile(fs, file.Name, file.Contents.Bytes(), os.FileMode(0666)) // +rw for all
		if err != nil {
			log.WithFields(log.Fields{"file": file.Name, "error": err.Error()}).Error("failed to write file")
			return fmt.Errorf("failed to write file %q: %w", file.Name, err)
		}
	}

	// write files in sub-directories
	for _, dir := range s.Directories {
		// create directory and intermediate directories (if needed)
		err := fs.MkdirAll(dir.Name, os.ModePerm)
		if err != nil {
			return fmt.Errorf("failed to create directory %q: %w", dir.Name, err)
		}

		// write files
		for _, file := range dir.Files {
			relPath := filepath.Join(dir.Name, file.Name)
			err := afero.WriteFile(fs, relPath, file.Contents.Bytes(), os.FileMode(0666)) // +rw for all
			if err != nil {
				log.WithFields(log.Fields{"file": relPath, "error": err.Error()}).Error("failed to write file")
			}
		}
	}

	// save manifest
	err = saveSolutionManifestToAferoFs(fs, &s.Manifest)
	if err != nil {
		return fmt.Errorf("failed to save the manifest: %w", err)
	}

	return nil
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
	if cmd != nil {
		output.PrintCmdOutputCustom(cmd, nil, &t)
	} else {
		for i, line := range t.Lines {
			debugTrace("%v: %v\n", t.Headers[i], line[0])
		}
	}

	// display directories
	for _, dir := range s.Directories {
		debugTrace("- Directory %v\n", dir)
		for _, file := range dir.Files {
			debugTrace("  File %v\n", file)
		}
	}
}

// --- Internal methods

// readContents reads the contents of the solution directory from the specified root
func (s *SolutionDirectoryContents) readContents(rootPath string) error {
	// read directories & files
	err := filepath.Walk(rootPath,
		func(path string, info os.FileInfo, err error) error {
			debugTrace("Walking %q dir=%v err=%v\n", path, info.IsDir(), err)
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
			if strings.HasPrefix(relPath, "..") {
				return fmt.Errorf("found %v %q that not under the root %q: %q", entryType, path, rootPath, relPath)
			}
			if !isAllowedPath(path, info) {
				log.Warnf("Found %v %q which cannot be bundled; it will still be processed", entryType, relPath)
			}

			if info.IsDir() {
				dir := SolutionSubDirectory{
					Name:  relPath,
					Files: make([]SolutionFile, 0),
				}
				s.Directories = append(s.Directories, dir)
				debugTrace("Appended a directory %v\n", dir)
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
					debugTrace("Appended a root file %v\n", file)
				} else {
					// find directory
					found := false
					for index := 0; index < len(s.Directories); index++ {
						dir := &s.Directories[index]
						if dir.Name == dirName {
							found = true
							dir.Files = append(dir.Files, file)
							debugTrace("Appended file %v to directory %v\n", file, dir)
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

	// process objects in the manifest (files and directories explicitly listed)
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

	// process files in object directories
	for _, dir := range s.Directories {
		if len(dir.Files) == 0 {
			continue // nothing to do, don't even check them
		}
		objectsType := ""
		fileKind := KindUnknown
		if dir.Type == ObjectsDir {
			if dir.ObjectsType == "" {
				log.Warnf("directory %q is marked as objectsDir but has no object type; not assigning type to its files", dir.Name)
			} else {
				fileKind = KindObjectType
				objectsType = dir.ObjectsType
			}
			for fileIndex := 0; fileIndex < len(dir.Files); fileIndex++ {
				f := &dir.Files[fileIndex]
				f.FileKind = fileKind
				f.ObjectType = objectsType
			}

		}
	}

	// process files in the root directory
	hiddenFiles := []string{"manifest.json", "manifest.yaml", "manifest.yml", ".tag"}
	for fileIndex := 0; fileIndex < len(s.RootFiles); fileIndex++ {
		f := &s.RootFiles[fileIndex]
		if slices.Contains(hiddenFiles, f.Name) {
			f.FileKind = KindHidden
		} else {
			f.FileKind = KindUnknown
		}
	}

	return nil
}

func (s *SolutionDirectoryContents) annotateFile(name string, kind SolutionFileKinds, objectType string) error {
	var file *SolutionFile

	dirName, fileName := filepath.Split(name)
	dirName = filepath.Clean(dirName) // removes the trailing separator
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
				debugTrace("found directory %q for %v %v %q: %v files\n", dirName, kind, objectType, fileName, len(dir.Files))
				for fileIndex := 0; fileIndex < len(dir.Files); fileIndex++ {
					f := &dir.Files[fileIndex]
					if f.Name == fileName {
						debugTrace("found %v %v %q against %q in directory %q\n", kind, objectType, fileName, f.Name, dirName)
						file = f
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
	for i := 0; i < len(s.Directories); i++ {
		d := &s.Directories[i]
		if d.Name == name {
			dir = d
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

// ErrDeleteWalkedFile is a special error that can be returned by the callback to WalkFiles
// to cause the file to be deleted. This error will never be returned by WalkFiles.
var ErrDeleteWalkedFile = errors.New("delete walked file")

// WalkFiles goes through each file, including root files and provides a callback
// to process each file. Each file is passed by reference, allowing the callback
// to modify the file contents.
// The subdirectory is passed by reference as well, but the callback should not modify it;
// it will be nil for the root files.
// The callback can return an error to stop the walk. It can also return the special
// error ErrDeleteWalkedFile to delete the file from the solution.
// TODO: when deleting a file, the file should be removed from the manifest, if referenced
func (s *SolutionDirectoryContents) WalkFiles(callback func(*SolutionFile, *SolutionSubDirectory) error) error {

	// enumerate the root directory files
	for i := 0; i < len(s.RootFiles); i++ {
		file := &s.RootFiles[i]
		err := callback(file, nil)
		if errors.Is(err, ErrDeleteWalkedFile) {
			log.Warnf("deleting files is not supported yet; file %q retained", file.Name)
		} else if err != nil {
			return err
		}
	}

	// enumerate files in subdirectories
	for i := 0; i < len(s.Directories); i++ {
		d := &s.Directories[i]
		for j := 0; j < len(d.Files); j++ {
			file := &d.Files[j]
			err := callback(file, d)
			if errors.Is(err, ErrDeleteWalkedFile) {
				log.Warnf("deleting files is not supported yet; file %q retained", filepath.Join(d.Name, file.Name))
			} else if err != nil {
				return err
			}
		}
	}

	return nil
}

func debugTrace(format string, args ...interface{}) {
	//fmt.Printf(format, args...)
}
