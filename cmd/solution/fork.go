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
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/apex/log"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/cisco-open/fsoc/config"
	"github.com/cisco-open/fsoc/output"
)

const pseudoIsolationSuffix = "${$toSuffix(env.tag)}"

var solutionForkCmd = &cobra.Command{
	Use:   "fork [<solution-name>|--source-dir=<directory>] <target-name> [flags]",
	Args:  cobra.MaximumNArgs(2),
	Short: "Fork a solution into the specified directory",
	Long:  `This command downloads the specified solution into the current directory and changes its name to <target-name>`,
	Example: `  fsoc solution fork spacefleet myfleet
  fsoc solution fork --source-dir=spacefleet myfleet`,
	Run: solutionForkCommand,
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
	solutionForkCmd.MarkFlagsMutuallyExclusive("source-dir", "tag")
	solutionForkCmd.MarkFlagsMutuallyExclusive("source-dir", "source-name")

	solutionForkCmd.Flags().BoolP("quiet", "q", false, "suppress output")

	solutionForkCmd.Flags().Bool("legacy-replace", false, "use pre-v0.68 fork algorithm (string replacement) (DEPRECATED)")
	solutionForkCmd.MarkFlagsMutuallyExclusive("legacy-replace", "quiet") // legacy code doesn't support quiet mode

	return solutionForkCmd
}

func solutionForkCommand(cmd *cobra.Command, args []string) {
	// get and check arguments & flags
	sourceDir, _ := cmd.Flags().GetString("source-dir")
	solutionName, _ := cmd.Flags().GetString("source-name")
	forkName, _ := cmd.Flags().GetString("name")
	solutionTag, _ := cmd.Flags().GetString("tag")
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
		log.Fatal("Incorrect argument syntax.")
	}
	if (solutionName == "" && sourceDir == "") || forkName == "" {
		_ = cmd.Help()
		log.Fatalf("A source and target must be specified")
	}
	if !IsValidSolutionName(forkName) {
		log.Fatalf("Invalid solution name %q: must start with a lowercase letter and contain only lowercase letters and digits", forkName)
	}
	if !IsValidSolutionTag(solutionTag) {
		log.Fatalf("Invalid solution tag %q: must start with a lowercase letter and contain only lowercase letters and digits", solutionTag)
	}

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
	if createSolutionDirectoryOk(fileSystemRoot, forkName) { // TODO: use code from init
		log.Fatalf(fmt.Sprintf("A non empty directory with the name %s already exists", forkName))
	}
	fileSystem := afero.NewBasePathFs(afero.NewOsFs(), currentDirectory+"/"+forkName)

	// fork from disk (always using the new algorithm with proper word boundaries in values only)
	if sourceDir != "" {
		err := forkFromDisk(sourceDir, fileSystem, forkName, statusPrint)
		if err != nil {
			log.Fatalf("Failed to fork solution from disk: %v", err)
		}
		statusPrint("Successfully forked %q into %q.", sourceDir, forkName)
		return
	}

	// Download & fork from solution that's already in the platform
	// (nb: this works only if the tenant has access to the solution)

	// backward compatibility: use the old algorithm that does global string replacement
	if legacyForkFlag, _ := cmd.Flags().GetBool("legacy-replace"); legacyForkFlag {
		// use the old algorithm that does global string replacement
		legacyFork(cmd, solutionName, solutionTag, forkName, fileSystemRoot, fileSystem)
		statusPrint("Successfully forked %s to current directory as %s using deprecated legacy replace.", solutionName, forkName)
		return
	}

	// download solution archive to the temporary directory
	archivePath, err := DownloadSolutionPackage(solutionName, solutionTag, "")
	if err != nil {
		log.Fatalf("Failed to download solution %q with tag %q: %v", solutionName, solutionTag, err)
	}

	// create a temp directory to extract files to
	sourceDir, err = os.MkdirTemp("", solutionName+"."+solutionTag+"-")
	if err != nil {
		log.Fatalf("Failed to create a temporary directory: %v", err)
	}
	log.WithField("temp_solution_dir", sourceDir).Info("Extracting downloaded solution in temp target directory")

	// extract files from the archive into the source directory
	// Note that archives have a top level directory that should be skipped at extraction)
	sourceDirFs := afero.NewBasePathFs(afero.NewOsFs(), sourceDir)
	if err = UnzipToAferoFs(archivePath, sourceDirFs, 1); err != nil {
		log.Fatalf("Failed to extract downloaded solution archive: %v", err)
	}

	// fork solution from the extracted directory
	err = forkFromDisk(sourceDir, fileSystem, forkName, statusPrint)
	if err != nil {
		log.Fatalf("Failed to fork solution (consider using --legacy-replace flag as a workaround): %v", err)
	}

	statusPrint("Successfully forked %s to current directory as %s.", solutionName, forkName)
}

func createSolutionDirectoryOk(fileSystem afero.Fs, forkName string) bool {
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
	solution.Manifest.SolutionVersion = "1.0.0" // reset version
	statusPrint("Forking %q to %q (version %v)...", oldName, solutionName, solution.Manifest.SolutionVersion)

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

	// change namespace of objects referenced in the manifest
	nManifestReplaces := 1 // #1 is the solution name
	for i := 0; i < len(solution.Manifest.Objects); i++ {
		key := fmt.Sprintf("objects[%v].type", i)
		obj := &solution.Manifest.Objects[i]
		oldType := obj.Type
		nReplacements := 0
		newType := replaceValues(oldType, oldNameRe, solutionName, key, &nReplacements).(string) // should be string, bug otherwise
		if nReplacements > 0 {
			obj.Type = newType

			// update the type in the file/dir that are being referenceds
			solution.SetComponentDefType(obj, obj.Type)
		}
		nManifestReplaces += nReplacements
	}
	pluralSuffix := ""
	if nManifestReplaces != 1 {
		pluralSuffix = "s"
	}
	statusPrint(`Made %v change%v in "manifest.%s"`, nManifestReplaces, pluralSuffix, solution.Manifest.ManifestFormat)

	// write new solution to disk
	statusPrint("The fsoc log file contains all changes made")
	statusPrint("Writing solution %q...", solutionName)
	err = solution.Write(fileSystem)
	if err != nil {
		return fmt.Errorf("error writing solution to disk: %w", err)
	}

	// mini-lint:
	// warn about antipattern of naming files or directories as the solution name
	nAntipatterns := 0
	for _, object := range solution.Manifest.Objects {
		if oldNameRe.MatchString(object.ObjectsDir) {
			log.WithFields(log.Fields{
				"dir":  object.ObjectsDir,
				"type": object.Type,
			}).Warn("Directory name contains the old solution name; unchanged")
			nAntipatterns++
		}
		if oldNameRe.MatchString(object.ObjectsFile) {
			log.WithFields(log.Fields{
				"file": object.ObjectsFile,
				"type": object.Type,
			}).Warn("File name contains the old solution name; unchanged")
			nAntipatterns++
		}
	}
	for _, knowledgeType := range solution.Manifest.Types {
		if oldNameRe.MatchString(knowledgeType) {
			log.WithField("file", knowledgeType).Warn("Knowledge type filename contains the old solution name; unchanged")
			nAntipatterns++
		}
	}
	if nAntipatterns > 0 {
		statusPrint("Using the solution name in directory or file name is not a good practice; it makes renaming the solution harder; consider using generic names instead")
	}

	//solution.Dump(nil) //@@ debug

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
