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
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/apex/log"
	"github.com/spf13/cobra"
)

const TagFileName = ".tag" // tag file in solution directory, similar to .env files; should NOT be version controlled

func addTagFlags(cmd *cobra.Command) {
	cmd.Flags().String("tag", "", "Tag to use for solution isolation")
	cmd.Flags().Bool("stable", false, "Use the stable tag for solution isolation. Equivalent to --tag=stable")
	cmd.MarkFlagsMutuallyExclusive("tag", "stable")
}

// getEmbeddedTag returns the tag to use for solution isolation for all commands that
// work with the solution directory.
// The tag is determined based on the following priority:
// 1. Specified flag `--tag=xyz` or `--stable“: use this tag, ignoring .tag file or env vars
// 2. A tag is defined in the FSOC_SOLUTION_TAG environment variable (ignores .tag file)
// 3. A tag is defined in the .tag file in the solution directory (usually not version controlled)
func getEmbeddedTag(cmd *cobra.Command, solutionPath string) (string, error) {
	tag, _ := cmd.Flags().GetString("tag")
	stable, _ := cmd.Flags().GetBool("stable")
	method := "--tag flag"

	if tag == "" && stable {
		tag = "stable"
		method = "--stable flag"
	}

	if tag == "" {
		tag = os.Getenv("FSOC_SOLUTION_TAG")
		method = "FSOC_SOLUTION_TAG env var"
	}

	if tag == "" {
		tagFile := filepath.Join(solutionPath, TagFileName)
		tagBytes, err := os.ReadFile(tagFile) // ok if no file or empty file
		if err == nil {
			tag = strings.TrimSpace(string(tagBytes))
			method = ".tag file"
			if tag != "" {
				checkTagFileIgnored(tagFile) // warn if .tag file may be checked in (unless empty)
			}
		}
	}

	if tag == "" {
		return "", fmt.Errorf("a non-empty tag must be specified for this command; see command help for options")
	}
	if !IsValidSolutionTag(tag) {
		return "", fmt.Errorf("tag %q, specified in the %s, is invalid", tag, method)
	}

	log.WithFields(log.Fields{"tag": tag, "source": method}).Info("Using tag for solution isolation")

	return tag, nil
}

// IsValidTag checks if a tag is valid for use in solution isolation.
// A valid tag is a non-empty string that starts with an ASCII letter and
// contains only lowercase ASCII letters and digits, for a max of 10 characters.
// TODO: add link to documentation on tags
func IsValidSolutionTag(tag string) bool {
	if tag == "" {
		return false
	}
	if len(tag) > 10 {
		return false
	}
	match, err := regexp.Match(`^[a-z][a-z0-9]*$`, []byte(tag))
	if !match || err != nil {
		return false
	}

	return true
}

// checkTagFileIgnored checks if the .tag file is properly excluded from solution's
// git repository; if not excluded, warns the user. This function assumes that the
// tag file exists (otherwise, it shouldn't be called).
func checkTagFileIgnored(tagFilePath string) {
	// use git to verify if the file is ignored
	cmd := exec.Command("git", "check-ignore", "-q", tagFilePath)
	if err := cmd.Run(); err != nil {
		// if git exits with a non-zero exit status, it could mean the file is not ignored or there was another error
		if exitErr, ok := err.(*exec.ExitError); ok {
			// exit code 1 means specifically that the file is not ignored
			if exitErr.ExitCode() == 1 {
				log.Warnf("Warning: The %s file is not ignored by git in this solution's repository. .tag files should not be checked in. Consider adding it to .gitignore", TagFileName)
			}
			// ignore other errors, including "not a git repo" or no git installed
		}
	}

	// If the command exited with status 0, the file is ignored, all good
}
