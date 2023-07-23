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

	"github.com/cisco-open/fsoc/cmd/config"
	"github.com/cisco-open/fsoc/output"
	"github.com/cisco-open/fsoc/platform/api"
)

var solutionForkCmd = &cobra.Command{
	Use:     "fork <solution-name> <target-name>",
	Args:    cobra.MaximumNArgs(2),
	Short:   "Fork a solution into the specified directory",
	Long:    `This command downloads the specified solution into the current directory and changes its name to <target-name>`,
	Example: `  fsoc solution fork spacefleet myfleet`,
	Run:     solutionForkCommand,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) >= 1 {
			return nil, cobra.ShellCompDirectiveDefault
		} else {
			config.SetCurrentProfile(cmd, args, false)
			return getSolutionNames(toComplete), cobra.ShellCompDirectiveDefault
		}
	},
}

func GetSolutionForkCommand() *cobra.Command {
	solutionForkCmd.Flags().String("source-name", "", "name of the solution that needs to be forked and downloaded")
	_ = solutionForkCmd.Flags().MarkDeprecated("source-name", "please use argument instead.")
	solutionForkCmd.Flags().String("name", "", "name of the solution to copy it to")
	_ = solutionForkCmd.Flags().MarkDeprecated("name", "please use argument instead.")
	solutionForkCmd.Flags().String("tag", "stable", "tag related to the solution to fork and download")
	return solutionForkCmd
}

func solutionForkCommand(cmd *cobra.Command, args []string) {
	solutionName, _ := cmd.Flags().GetString("source-name")
	forkName, _ := cmd.Flags().GetString("name")
	if len(args) == 2 {
		solutionName, forkName = args[0], args[1]
	} else if len(args) != 0 {
		_ = cmd.Help()
		log.Fatal("Exactly 2 arguments required.")
	}
	if solutionName == "" || forkName == "" {
		log.Fatalf("<solution-name> and <target-name> cannot be empty")
	}
	forkName = strings.ToLower(forkName)

	currentDirectory, err := filepath.Abs(".")
	if err != nil {
		log.Fatalf("Error getting current directory: %v", currentDirectory)
	}

	fileSystemRoot := afero.NewBasePathFs(afero.NewOsFs(), currentDirectory)

	if solutionNameFolderInvalid(fileSystemRoot, forkName) {
		log.Fatalf(fmt.Sprintf("A non empty directory with the name %s already exists", forkName))
	}

	fileSystem := afero.NewBasePathFs(afero.NewOsFs(), currentDirectory+"/"+forkName)

	if manifestExists(fileSystem, forkName) {
		log.Fatalf("There is already a manifest file in this directory")
	}

	downloadSolutionZip(cmd, solutionName, forkName)
	err = extractZip(fileSystemRoot, fileSystem, solutionName)
	if err != nil {
		log.Fatalf("Failed to copy files from the zip file to current directory: %v", err)
	}

	editManifest(fileSystem, forkName)

	err = fileSystemRoot.Remove("./" + solutionName + ".zip")
	if err != nil {
		log.Fatalf("Failed to remove zip file in current directory: %v", err)
	}

	message := fmt.Sprintf("Successfully forked %s to current directory.\r\n", solutionName)
	output.PrintCmdStatus(cmd, message)

}

func solutionNameFolderInvalid(fileSystem afero.Fs, forkName string) bool {
	exists, _ := afero.DirExists(fileSystem, forkName)
	if exists {
		empty, _ := afero.IsEmpty(fileSystem, forkName)
		return !empty
	} else {
		err := fileSystem.Mkdir(forkName, os.ModeDir)
		if err != nil {
			log.Fatalf("Failed to create directory in this directory")
		}
		err = os.Chmod(forkName, 0700)
		if err != nil {
			log.Fatalf("Failed to set permission on directory")
		}
	}
	return false
}

func manifestExists(fileSystem afero.Fs, forkName string) bool {
	exists, err := afero.Exists(fileSystem, forkName+"/manifest.json")
	if err != nil {
		log.Fatalf("Failed to read filesystem for manifest: %v", err)
	}
	return exists
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

func downloadSolutionZip(cmd *cobra.Command, solutionName string, forkName string) {
	var solutionNameWithZipExtension = getSolutionNameWithZip(solutionName)
	solutionTagFlag, _ := cmd.Flags().GetString("tag")
	var message string

	headers := map[string]string{
		"stage":            "STABLE",
		"tag":              solutionTagFlag,
		"solutionFileName": solutionNameWithZipExtension,
	}
	httpOptions := api.Options{Headers: headers}
	bufRes := make([]byte, 0)
	if err := api.HTTPGet(getSolutionDownloadUrl(solutionName), &bufRes, &httpOptions); err != nil {
		log.Fatalf("Solution download failed: %v", err)
	}

	message = fmt.Sprintf("Solution %s was successfully downloaded in the this directory.\r\n", solutionName)
	output.PrintCmdStatus(cmd, message)

	message = fmt.Sprintf("Changing solution name in manifest to %s.\r\n", forkName)
	output.PrintCmdStatus(cmd, message)
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
