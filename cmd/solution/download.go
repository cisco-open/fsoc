// Copyright 2023 Cisco Systems, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package solution

import (
	"archive/zip"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/apex/log"
	"github.com/spf13/afero"
	"github.com/spf13/afero/zipfs"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/config"
	"github.com/cisco-open/fsoc/output"
	"github.com/cisco-open/fsoc/platform/api"
)

var solutionDownloadCmd = &cobra.Command{
	Use:              "download <solution-name>",
	Args:             cobra.MaximumNArgs(1),
	Short:            "Download solution",
	Long:             `This downloads the indicated solution into the current directory. Also see the "fork" command.`,
	Example:          `  fsoc solution download spacefleet`,
	Run:              downloadSolution,
	TraverseChildren: true,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		config.SetActiveProfile(cmd, args, false)
		return getSolutionNames(toComplete), cobra.ShellCompDirectiveDefault
	},
}

func getSolutionDownloadCmd() *cobra.Command {
	solutionDownloadCmd.Flags().String("name", "", "name of the solution to download (required)")
	_ = solutionDownloadCmd.Flags().MarkDeprecated("name", "please use argument instead.")

	solutionDownloadCmd.Flags().String("tag", "stable", "tag related to the solution to download")
	return solutionDownloadCmd
}

func downloadSolution(cmd *cobra.Command, args []string) {
	solutionName := getSolutionNameFromArgs(cmd, args, "name")
	solutionTagFlag, _ := cmd.Flags().GetString("tag")

	if _, err := DownloadSolutionPackage(solutionName, solutionTagFlag, "."); err != nil {
		log.Fatal(err.Error())
	}

	message := fmt.Sprintf("Solution %q with tag %s downloaded successfully.\n", solutionName, solutionTagFlag)
	output.PrintCmdStatus(cmd, message)
}

// DownloadSolutionPackage downloads the solution package into the specified target path
// targetPath may be one of the following:
// - the empty string: download to a temporary file
// - a directory: download to a file in that directory
// - a file: download to that file
// The function returns the path to the downloaded file and error.
func DownloadSolutionPackage(name string, tag string, targetPath string) (string, error) {
	// validate name and tag
	if !IsValidSolutionName(name) {
		return "", fmt.Errorf("invalid solution name %q", name)
	}
	if !IsValidSolutionTag(tag) {
		return "", fmt.Errorf("invalid solution tag %q", tag)
	}

	// determine the target file path
	if targetPath != "" {
		// absolutize path
		targetPath = absolutizePath(targetPath)

		// if targetPath is an existing directory, place zip there; otherwise, treat as file path
		fileInfo, err := os.Stat(targetPath)
		if err == nil && fileInfo.IsDir() {
			targetPath = filepath.Join(filepath.Dir(targetPath), name+".zip")
		} else if err != nil && !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("failed to access target path %q: %v", targetPath, err)
		} // else treat as file path, possibly overwriting existing file
	} else {
		// create unique file name in the temporary directory
		archive, err := os.CreateTemp("", name+"*.zip") // "*" will be replaced with unique string
		if err != nil {
			log.Fatalf("failed to create temporary archive file: %v", err)
		}
		targetPath = archive.Name()
	}

	headers := map[string]string{
		"stage":            "STABLE", // TODO: check if needed
		"tag":              tag,
		"solutionFileName": targetPath,
	}
	httpOptions := api.Options{Headers: headers}
	bufRes := make([]byte, 0)
	if err := api.HTTPGet(getSolutionDownloadUrl(name), &bufRes, &httpOptions); err != nil {
		return "", fmt.Errorf("Solution download command failed: %v", err)
	}

	log.WithFields(log.Fields{"solution": name, "tag": tag, "path": targetPath}).Info("Solution archive downloaded successfully")

	return targetPath, nil
}

func getSolutionDownloadUrl(solutionName string) string {
	return fmt.Sprintf("solution-manager/v1/solutions/%s", solutionName)
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
