// Copyright 2024 Cisco Systems, Inc.
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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/apex/log"
	"github.com/spf13/afero"
)

// UnzipToAferoFs extracts files from a zip file to an afero file system.
// skipLevels specifies how many directories from the top level should be skipped
// over when constructing the target path (similar to the -p flag of the patch command).
// Note that there should be no files in the skipped levels; otherwise, this function will
// return an error.
func UnzipToAferoFs(zipFile string, targetFs afero.Fs, skipLevels int) error {
	// Open the zip file
	zipReader, err := zip.OpenReader(zipFile)
	if err != nil {
		return fmt.Errorf("failed to open zip file %q: %w", zipFile, err)
	}

	skippedPrefix := ""
	skippedLevels := 0

	// Iterate through the files in the zip archive
	for _, file := range zipReader.File {
		permissions := file.Mode() & os.ModePerm // ensure only permissions bits are used
		// fmt.Printf("Extracting %q with permissions %o, isDir=%v\n", file.Name, permissions, file.FileInfo().IsDir())

		// Construct the target file path
		filePath := file.Name
		switch filepath.Separator { // convert to the target file system's separator
		case '\\':
			filePath = strings.Replace(filePath, "/", "\\", -1) // Linux zips on Windows
		case '/':
			filePath = strings.Replace(filePath, "\\", "/", -1) // Windows zips on Linux
		default:
			// unknown separator; assume it's the same as the source
		}

		// Check for ZipSlip (a security vulnerability)
		// This is a belt-and-suspenders-type check; one reason we use afero in the first
		// place is to guard against this type of issues. Afero doesn't allow reach outside
		// the targetFs.
		// Note the filePath was `Clean`-ed, so if it leaves the root, it will start with "../"
		if strings.HasPrefix(filePath, ".."+string(os.PathSeparator)) { // definitely illegal
			return fmt.Errorf("%q: illegal file path in zip file %q", filePath, zipFile)
		}
		if strings.Contains(filePath, ".."+string(os.PathSeparator)) { // possible false positive for files like `test../test`
			return fmt.Errorf("%q: possibly illegal file path in zip file %q", filePath, zipFile)
		}

		// accumulate directory prefix until the target skip level is reached
		// Note that there the skipped level directories may be accumulated over several
		// entries but there can be no files until the target skip level is reached.
		if skippedLevels < skipLevels {
			// ensure that only directories are skipped
			if !file.FileInfo().IsDir() {
				return fmt.Errorf("found a file, %q, in skipped levels (%v); not supported", file.Name, skipLevels)
			}

			// ensure we're building up the prefix
			if !strings.HasPrefix(file.Name, skippedPrefix) {
				log.Warnf("skip directory %q does not include accumulated prefix %q", file.Name, skippedPrefix)
			}

			// update prefix
			dirList := strings.Split(file.Name, string(filepath.Separator))
			if len(dirList) > skipLevels {
				dirList = dirList[:skipLevels] // limit skip to specified level
			}
			skippedPrefix = filepath.Join(dirList...)
			skippedLevels = len(dirList)
			// fmt.Printf("Updated skippedPrefix to %q (%d)\n", skippedPrefix, skippedLevels)
			continue
		}

		// remove prefix (if any)
		filePath, _ = strings.CutPrefix(filePath, skippedPrefix)

		// Create directories if this is a directory entry
		if file.FileInfo().IsDir() {
			// fmt.Printf("Creating directory %q\n", filePath)
			err := targetFs.MkdirAll(filePath, permissions|0o100) // ensure enumerable by owner
			if err != nil {
				return fmt.Errorf("failed to create directory %q: %w", filePath, err)
			}
			continue
		}

		// fmt.Printf("Unzipping %q into %q\n", file.Name, filePath)

		// Open the file within the zip archive
		srcFile, err := file.Open()
		if err != nil {
			return fmt.Errorf("failed to open zipped file %q: %w", file.Name, err)
		}
		defer srcFile.Close()

		// Create the destination file
		dstFile, err := targetFs.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, permissions)
		if err != nil {
			return fmt.Errorf("failed to create destination file: %w", err)
		}
		defer dstFile.Close()

		// Copy the file's contents to the new file
		if _, err = io.Copy(dstFile, srcFile); err != nil {
			return fmt.Errorf("failed to copy file contents of %q: %w", file.Name, err)
		}
	}

	return nil
}
