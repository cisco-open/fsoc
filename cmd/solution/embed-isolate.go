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
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/apex/log"
	"github.com/spf13/cobra"
)

// embeddedConditionalIsolate prepares a finalized version of a solution directory
// in a way that's ready to pass to the platform. If the solution supports isolation
// (i.e., uses templates in the manifest/objects), then this function ensures that
// template vars are fully replaced. The function is specifically made to be convenient
// to call from handlers of cobra commands and rely on common definition of flags.
// To perform isolation without command dependencies, use isolateSolution()
func embeddedConditionalIsolate(cmd *cobra.Command, sourceDir string) (string, error) {
	// don't try to isolate if --no-isolate is specified (ignored if flag not defined)
	noIsolate, _ := cmd.Flags().GetBool("no-isolate")
	if noIsolate {
		return sourceDir, nil
	}

	// return the solution folder as is if the manifest does not use isolation
	manifest, err := getSolutionManifest(sourceDir)
	if err != nil {
		return "", err
	}
	if !strings.Contains(manifest.Name, "${") {
		//TODO: consider failing if env vars file is specified/present
		//      as this is doesn't make sense for solutions that don't support isolation
		return sourceDir, nil
	}

	// finalize flags
	tag, envVarsFile := determineTagEnvFile(cmd, sourceDir)

	// prepare target directory
	// TODO: instead of fsoc as prefix, use as much as we can extract from the solution name
	//       in the manifest (assuming "<solution-name>${<something>}", the idea is to extract <solution-name>
	//       and use that as a prefix). This will have only cosmetic advantages.
	targetDir, err := os.MkdirTemp("", "fsoc")
	if err != nil {
		return "", fmt.Errorf("Failed to create a temporary directory: %v", err)
	}
	log.WithField("temp_solution_dir", targetDir).Info("Assembling solution in temp target directory")

	// render templates to produce the final solution
	name, err := isolateSolution(cmd, sourceDir, targetDir, "", tag, envVarsFile)
	if err != nil {
		os.RemoveAll(targetDir)
		return "", nil
	}

	log.WithFields(log.Fields{
		"isolated_solution_name": name,
		"from_directory":         sourceDir,
		"to_directory":           targetDir,
	}).Info("Isolated solution")

	return targetDir, nil
}

func determineTagEnvFile(cmd *cobra.Command, sourceDir string) (string, string) {
	// if --tag flag is specified, this overrides everything
	if cmd.Flags().Changed("tag") {
		tag, _ := cmd.Flags().GetString("tag")
		return tag, ""
	}

	// if --stable is specified, treat the same as tag
	stable, _ := cmd.Flags().GetBool("stable")
	if stable {
		return "stable", ""
	}

	// if env var with tag is defined, it overrides env file
	envTag, found := os.LookupEnv("FSOC_SOLUTION_TAG")
	if found {
		return envTag, ""
	}

	// try to determine env.json file path
	fnameSpecified := cmd.Flags().Changed("env-file")
	fname := ""
	if fnameSpecified {
		fname, _ = cmd.Flags().GetString("env-file")
	} else {
		fname = filepath.Join(sourceDir, "env.json")
	}
	if fname != "" {
		_, err := os.Stat(fname)
		if err == nil {
			return "", fname
		}
	}
	if fnameSpecified {
		log.Fatalf("Env file %q not found", fname)
	}

	log.Fatalf("Tag must be specified (--tag, --stable, FSOC_SOLUTION_TAG env var or env.json file)")
	return "", "" // should never happen, keep linters happy
}
