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
	"archive/zip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/apex/log"
	"github.com/spf13/afero"
	"github.com/spf13/afero/zipfs"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/output"
	"github.com/cisco-open/fsoc/platform/api"
)

// Legacy algorithm to fork a solution using global string replace and assuming json manifest
// Remove this code, together with the `--legacy-replace` flag, once the new solution is proven to work well

func legacyFork(cmd *cobra.Command, solutionName string, solutionTag string, forkName string, fileSystemRoot afero.Fs, fileSystem afero.Fs) {
	// download the solution zip file to the current directory
	downloadSolutionZip(cmd, solutionName, solutionTag, forkName)

	message := fmt.Sprintf("Solution %s was successfully downloaded in the this directory.\r\n", solutionName)
	output.PrintCmdStatus(cmd, message)

	message = fmt.Sprintf("Changing solution name in manifest to %s.\r\n", forkName)
	output.PrintCmdStatus(cmd, message)

	// extract files into the newly created solution directory
	err := extractZip(fileSystemRoot, fileSystem, solutionName)
	if err != nil {
		log.Fatalf("Failed to copy files from the zip file to current directory: %v", err)
	}

	// global replace of solution name in all files in place
	editManifest(fileSystem, forkName)

	// cleanup the zip file (TODO: move to temp folder and skip cleanup)
	err = fileSystemRoot.Remove("./" + solutionName + ".zip")
	if err != nil {
		log.Fatalf("Failed to remove zip file in current directory: %v", err)
	}
}

func editManifest(fileSystem afero.Fs, forkName string) {
	manifestFile, err := afero.ReadFile(fileSystem, "./manifest.json")
	if err != nil {
		log.Fatalf("Error opening manifest file: %v", err)
	}

	var manifest Manifest
	err = json.Unmarshal(manifestFile, &manifest)
	if err != nil {
		log.Errorf("Failed to parse solution manifest: %v", err)
	}

	err = refactorSolution(fileSystem, &manifest, forkName)
	if err != nil {
		log.Errorf("Failed to refactor component definition files within the solution: %v", err)
	}
	manifest.Name = forkName

	f, err := fileSystem.OpenFile("./manifest.json", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatalf("Can't open manifest file: %v", err)
	}
	defer f.Close()
	err = output.WriteJson(manifest, f)
	if err != nil {
		log.Errorf("Failed to to write to solution manifest: %v", err)
	}
}

func refactorSolution(fileSystem afero.Fs, manifest *Manifest, forkName string) error {
	objDefs := manifest.Objects
	var err error
	for _, objDef := range objDefs {
		if objDef.ObjectsFile != "" {
			err = ReplaceStringInFile(fileSystem, objDef.ObjectsFile, manifest.Name, forkName)
		} else {
			wkDir, _ := os.Getwd()
			dirPath := fmt.Sprintf("%s/%s/%s", wkDir, forkName, objDef.ObjectsDir)
			err = filepath.Walk(dirPath,
				func(path string, info os.FileInfo, err error) error {
					// if err != nil {
					// 	return err
					// }
					if !info.IsDir() {
						removeStr := fmt.Sprintf("%s/%s/", wkDir, forkName)
						filePath := strings.ReplaceAll(path, removeStr, "")
						err = ReplaceStringInFile(fileSystem, filePath, manifest.Name, forkName)
					}
					return err
				})
		}
	}
	return err
}

func ReplaceStringInFile(fileSystem afero.Fs, filePath string, searchValue string, replaceValue string) error {
	data, err := afero.ReadFile(fileSystem, filePath)
	if err != nil {
		return err
	}
	newFileContent := string(data)
	newFileContent = strings.ReplaceAll(newFileContent, searchValue, replaceValue)
	err = afero.WriteFile(fileSystem, filePath, []byte(newFileContent), os.FileMode(os.O_RDWR))
	if err != nil {
		return err
	}
	return nil
}

func extractZip(rootFileSystem afero.Fs, fileSystem afero.Fs, solutionName string) error {
	zipFile, err := rootFileSystem.OpenFile("./"+solutionName+".zip", os.O_RDONLY, os.FileMode(0644))
	if err != nil {
		log.Fatalf("Error opening zip file: %v", err)
	}
	fileInfo, err := rootFileSystem.Stat("./" + solutionName + ".zip")
	if err != nil {
		log.Errorf("Err reading zip: %v", err)
	}
	reader, _ := zip.NewReader(zipFile, fileInfo.Size())
	zipFileSystem := zipfs.New(reader)
	dirInfo, _ := afero.ReadDir(zipFileSystem, "./")
	err = copyFolderToLocal(zipFileSystem, fileSystem, dirInfo[0].Name())
	return err
}

func downloadSolutionZip(cmd *cobra.Command, solutionName string, solutionTag string, forkName string) {
	var solutionNameWithZipExtension = solutionName + ".zip"

	headers := map[string]string{
		"stage":            "STABLE", // TODO: check if still needed
		"tag":              solutionTag,
		"solutionFileName": solutionNameWithZipExtension,
	}
	httpOptions := api.Options{Headers: headers}
	bufRes := make([]byte, 0)
	if err := api.HTTPGet(getSolutionDownloadUrl(solutionName), &bufRes, &httpOptions); err != nil {
		log.Fatalf("Solution download failed: %v", err)
	}
}

func ExtractZipToDirectory(archive string, targetFs afero.Fs) error {
	archiveFile, err := os.OpenFile(archive, os.O_RDONLY, os.FileMode(0644))
	if err != nil {
		return fmt.Errorf("error opening zip file: %w", err)
	}
	defer archiveFile.Close()

	archiveFileInfo, err := os.Stat(archive)
	if err != nil {
		return fmt.Errorf("error determining zip file size: %w", err)
	}

	reader, _ := zip.NewReader(archiveFile, archiveFileInfo.Size())
	zipFileSystem := zipfs.New(reader)
	dirInfo, _ := afero.ReadDir(zipFileSystem, "./")
	err = copyFolderToLocal(zipFileSystem, targetFs, dirInfo[0].Name())
	return err
}

func copyFolderToLocal(zipFileSystem afero.Fs, localFileSystem afero.Fs, subDirectory string) error {
	dirInfo, err := afero.ReadDir(zipFileSystem, subDirectory)
	if err != nil {
		return err
	}
	for i := range dirInfo {
		zipLoc := subDirectory + "/" + dirInfo[i].Name()
		localLoc := convertZipLocToLocalLoc(subDirectory + "/" + dirInfo[i].Name())
		if !dirInfo[i].IsDir() {
			err = copyFile(zipFileSystem, localFileSystem, zipLoc, localLoc)
			if err != nil {
				return err
			}
		} else {
			err = localFileSystem.Mkdir(localLoc, os.ModeDir)
			if err != nil {
				return err
			}
			println(localLoc)
			err = localFileSystem.Chmod(localLoc, 0700)
			if err != nil {
				return err
			}
			err = copyFolderToLocal(zipFileSystem, localFileSystem, zipLoc)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func copyFile(zipFileSystem afero.Fs, localFileSystem afero.Fs, zipLoc string, localLoc string) error {
	data, err := afero.ReadFile(zipFileSystem, zipLoc)
	if err != nil {
		return err
	}
	_, err = localFileSystem.Create(localLoc)
	if err != nil {
		return err
	}
	err = afero.WriteFile(localFileSystem, localLoc, data, os.FileMode(os.O_RDWR))
	if err != nil {
		return err
	}
	return nil
}

func convertZipLocToLocalLoc(zipLoc string) string {
	secondSlashIndex := strings.Index(zipLoc[2:], "/")
	return zipLoc[secondSlashIndex+3:]
}
