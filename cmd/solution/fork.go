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
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/apex/log"
	"github.com/spf13/afero"
	"github.com/spf13/afero/zipfs"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/cisco-open/fsoc/config"
	"github.com/cisco-open/fsoc/output"
	"github.com/cisco-open/fsoc/platform/api"
)

const pseudoIsolationSuffix = "${$toSuffix(env.tag)}"

var solutionForkCmd = &cobra.Command{
	Use:     "fork [<solution-name>|--source-dir=<directory>] <target-name> [flags]",
	Args:    cobra.MaximumNArgs(2),
	Short:   "Fork a solution into the specified directory",
	Long:    `This command downloads the specified solution into the current directory and changes its name to <target-name>`,
	Example: `  fsoc solution fork spacefleet myfleet`,
	Run:     solutionForkCommand,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) >= 1 {
			return nil, cobra.ShellCompDirectiveDefault
		} else {
			config.SetActiveProfile(cmd, args, false)
			return getSolutionNames(toComplete), cobra.ShellCompDirectiveDefault
		}
	},
}

func GetSolutionForkCommand() *cobra.Command {
	solutionForkCmd.Flags().String("source-name", "", "name of the solution that needs to be forked and downloaded")
	_ = solutionForkCmd.Flags().MarkDeprecated("source-name", "please use argument instead.")
	solutionForkCmd.Flags().String("name", "", "name of the solution to copy it to")
	_ = solutionForkCmd.Flags().MarkDeprecated("name", "please use argument instead.")

	solutionForkCmd.Flags().StringP("source-dir", "s", "", "directory with a solution to fork from disk")
	solutionForkCmd.Flags().String("tag", "stable", "tag for the solution to download and fork")
	solutionForkCmd.Flags().BoolP("quiet", "q", false, "suppress output")

	return solutionForkCmd
}

func solutionForkCommand(cmd *cobra.Command, args []string) {
	// get and check arguments & flags
	sourceDir, _ := cmd.Flags().GetString("source-dir")
	solutionName, _ := cmd.Flags().GetString("source-name")
	forkName, _ := cmd.Flags().GetString("name")
	if len(args) == 2 {
		if sourceDir != "" {
			_ = cmd.Help()
			log.Fatal("Cannot specify both source solution and source directory; use only one.")
		}
		solutionName, forkName = args[0], args[1]
	} else if len(args) == 1 && sourceDir != "" {
		forkName = args[0]
	} else if len(args) != 0 {
		_ = cmd.Help()
		log.Fatal("Exactly 2 arguments required.")
	}
	if (solutionName == "" && sourceDir == "") || forkName == "" {
		_ = cmd.Help()
		log.Fatalf("A source and target must be specified")
	}
	forkName = strings.ToLower(forkName)

	// create a status printer function closure, based on the quiet flag
	quiet, _ := cmd.Flags().GetBool("quiet")
	var statusPrint func(fmt string, args ...interface{})
	if quiet {
		statusPrint = func(fmt string, args ...interface{}) {}
	} else {
		statusPrint = func(format string, args ...interface{}) {
			s := fmt.Sprintf(format+"\n", args...)
			output.PrintCmdStatus(cmd, s)
		}
	}

	// create afero filesystem for the target directory
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

	if sourceDir != "" {
		err := forkFromDisk(sourceDir, fileSystem, forkName, statusPrint)
		if err != nil {
			log.Fatalf("Failed to fork solution from disk: %v", err)
		}
		statusPrint("Successfully forked %q into %q.", sourceDir, forkName)
		return
	}

	// Download & fork from solution, using legacy fork

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

	statusPrint("Successfully forked %s to current directory.", solutionName)
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

// -- New style fork
// 1. Replaces the solution name only in values but not in keys
// 2. Replaces the solution name only on word boundaries
// 3. Omits special files, such as .tag, from the output solution
// 4. Renames the namespace file if it is the same as the solution name
// 5. Logs detailed list of changes made to the solution, with key names and old/new values

func forkFromDisk(sourceDir string, fileSystem afero.Fs, solutionName string, statusPrint func(string, ...any)) error {
	// load solution from disk
	solution, err := NewSolutionDirectoryContentsFromDisk(sourceDir)
	if err != nil {
		return fmt.Errorf("error loading solution from disk: %w", err)
	}

	// get and replace solution name in the manifest, respecting pseudo-isolation
	oldName, pseudoIsolated := strings.CutSuffix(solution.Manifest.Name, pseudoIsolationSuffix)
	if pseudoIsolated {
		solution.Manifest.Name = solutionName + pseudoIsolationSuffix
	} else {
		solution.Manifest.Name = solutionName
	}
	statusPrint("Forking %q to %q...", oldName, solutionName)

	// remove files that should not be carried over, e.g., a .tag file in the root directory
	err = solution.WalkFiles(func(file *SolutionFile, dir *SolutionSubDirectory) error {
		if dir == nil && file.Name == ".tag" {
			log.WithField("file", file.Name).Info("Removing special file")
			statusPrint("Removed special file %q", file.Name)
			return ErrDeleteWalkedFile
		}
		return nil
	})
	if err != nil {
		log.Fatalf("(bug) Failed to remove special files from solution: %v", err)
	}

	// rename the namespace file if it uses the solution name and update the manifest
	err = solution.WalkFiles(func(file *SolutionFile, dir *SolutionSubDirectory) error {
		if file.FileKind != KindObjectType ||
			file.ObjectType != "fmm:namespace" ||
			file.Name != oldName+"."+string(file.Encoding) {
			return nil
		}
		// rename file to match the new solution name
		oldFileName := file.Name
		file.Name = solutionName + "." + string(file.Encoding)

		// log change
		dirName := "."
		if dir != nil {
			dirName = dir.Name
		}
		oldPath := filepath.Join(dirName, oldFileName)
		newPath := filepath.Join(dirName, file.Name)
		statusPrint("Renamed namespace file %q to %q", oldPath, newPath)
		log.WithFields(log.Fields{
			"old_file": oldPath,
			"new_file": newPath,
		}).Info("Renamed namespace file")
		return nil
	})
	if err != nil {
		log.Fatalf("(bug) Failed to rename namespace file: %v", err)
	}

	// prepare regexp for replacing solution name in all files
	oldNameRe, err := regexp.Compile(`\b` + regexp.QuoteMeta(oldName) + `\b`) // \b is a word boundary
	if err != nil {
		log.Fatalf("(bug) Failed to compile regexp for solution name replacement: %v", err)
		panic("") // unreachable
	}

	// modify solution name in all files
	err = solution.WalkFiles(func(file *SolutionFile, dir *SolutionSubDirectory) error {
		// skip hidden files
		if file.FileKind == KindHidden {
			return nil // nothing to do
		}

		// prepare root-relative file name
		dirName := "."
		if dir != nil {
			dirName = dir.Name
		}
		filePath := filepath.Join(dirName, file.Name)

		// replace solution name in file buffer
		newContents, nReplacements, err := forkFileInBuffer(file.Contents, file.Encoding, oldNameRe, solutionName)
		if err != nil {
			dirName := "."
			if dir != nil {
				dirName = dir.Name
			}
			return fmt.Errorf("error forking file %q: %w", filepath.Join(dirName, file.Name), err)
		}
		if nReplacements > 0 {
			// replace file contents
			file.Contents = newContents

			// log and print changes
			pluralSuffix := ""
			if nReplacements != 1 {
				pluralSuffix = "s"
			}
			statusPrint("Made %v change%v in %q", nReplacements, pluralSuffix, filePath)
			log.WithFields(log.Fields{
				"file": filePath,
				"repl": nReplacements,
			}).Info("Forked file")
		} else {
			log.WithField("file", file.Name).Info("No changes in file")
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("error updating solution files: %w", err)
	}

	// write new solution to disk
	statusPrint("The fsoc log file contains all changes made")
	statusPrint("Writing solution %q...", solutionName)
	err = solution.Write(fileSystem)
	if err != nil {
		return fmt.Errorf("error writing solution to disk: %w", err)
	}

	return nil
}

// forkFileInBuffer replaces oldName with newName in the contents values, respecting
// word boundaries. The old name is specified as a regular expression.
// The function returns the new buffer with the replacements and the number of
// replacements made, as well as an error.
func forkFileInBuffer(buffer bytes.Buffer, encoding SolutionFileEncoding, oldNameRe *regexp.Regexp, newName string) (bytes.Buffer, int, error) {
	// decode buffer to map[string]interace{}
	var contents any
	var err error
	switch encoding {
	case EncodingJSON:
		err = json.Unmarshal(buffer.Bytes(), &contents)
	case EncodingYAML:
		err = yaml.Unmarshal(buffer.Bytes(), &contents)
	default:
		return bytes.Buffer{}, 0, fmt.Errorf("unsupported encoding: %v", encoding)
	}
	if err != nil {
		return bytes.Buffer{}, 0, fmt.Errorf("error decoding %v file: %w", encoding, err)
	}

	// replace solution name in values, starting from the root base key ("")
	nReplacements := 0
	contents = replaceValues(contents, oldNameRe, newName, "", &nReplacements)

	// return original buffer if no replacements were made
	if nReplacements == 0 {
		return buffer, 0, nil
	}

	// encode map back to buffer
	var newBuffer bytes.Buffer
	switch encoding {
	case EncodingJSON:
		err = json.NewEncoder(&newBuffer).Encode(contents)
	case EncodingYAML:
		err = yaml.NewEncoder(&newBuffer).Encode(contents)
	}
	if err != nil {
		return bytes.Buffer{}, nReplacements, fmt.Errorf("error re-encoding %v file with %v modifications: %w", encoding, nReplacements, err)
	}

	return newBuffer, nReplacements, nil
}

func replaceValues(contents any, oldNameRe *regexp.Regexp, newName string, base string, nReplacements *int) any {
	switch val := contents.(type) {
	case map[string]any:
		for k, v := range val {
			var key string
			if base == "" {
				key = k
			} else {
				key = base + "." + k
			}
			val[k] = replaceValues(v, oldNameRe, newName, key, nReplacements)
		}
	case []any:
		for i, v := range val {
			key := base + fmt.Sprintf("[%d]", i)
			val[i] = replaceValues(v, oldNameRe, newName, key, nReplacements)
		}
	case string:
		// replace old name with the new one, counting the number of replacements
		replacements := 0
		newVal := oldNameRe.ReplaceAllStringFunc(val, func(match string) string {
			replacements++
			return newName
		})
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

	return contents
}
