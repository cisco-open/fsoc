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
// The function returns the directory to use (original content if not isolating or
// rendered values if isolating), the tag value and error.
// To perform isolation without command dependencies, use isolateSolution().
func embeddedConditionalIsolate(cmd *cobra.Command, sourceDir string) (string, string, error) {
	// finalize flags (regardless of isolation)
	tag, envVarsFile := DetermineTagEnvFile(cmd, sourceDir)

	// don't try to isolate if --no-isolate is specified (ignored if flag not defined)
	noIsolate, _ := cmd.Flags().GetBool("no-isolate")
	if noIsolate {
		if envVarsFile != "" {
			log.Warnf("--no-isolate flag specified while an isolation env file %q is present", envVarsFile)
		}
		return sourceDir, tag, nil
	}

	// return the solution folder as is if the manifest does not use isolation
	manifest, err := getSolutionManifest(sourceDir)
	if err != nil {
		return "", "", err
	}
	if !strings.Contains(manifest.Name, "${") {
		if envVarsFile != "" {
			log.Warnf("Isolation env file %q is present for a solution that doesn't use isolation variables", envVarsFile)
		}
		return sourceDir, tag, nil
	}

	// reject if manifest is in YAML format (pseudo-isolation is supported only for JSON files)
	if manifest.ManifestFormat != FileFormatJSON {
		return "", "", fmt.Errorf("pseudo-isolation is supported only for JSON-formatted solutions")
	}

	log.Warnf("This solution uses fsoc-provided pseudo-isolation, which is now deprecated; please transition your solutions to native isolation soon to get the full isolation benefits.")

	// prepare target directory
	// TODO: instead of fsoc as prefix, use as much as we can extract from the solution name
	//       in the manifest (assuming "<solution-name>${<something>}", the idea is to extract <solution-name>
	//       and use that as a prefix). This will have only cosmetic advantages.
	targetDir, err := os.MkdirTemp("", "fsoc")
	if err != nil {
		return "", "", fmt.Errorf("failed to create a temporary directory: %v", err)
	}
	log.WithField("temp_solution_dir", targetDir).Info("Assembling solution in temp target directory")

	// render templates to produce the final solution
	name, tag, err := isolateSolution(cmd, sourceDir, targetDir, "", tag, envVarsFile)
	if err != nil {
		os.RemoveAll(targetDir)
		return "", "", err
	}

	log.WithFields(log.Fields{
		"isolated_solution_name": name,
		"from_directory":         sourceDir,
		"to_directory":           targetDir,
		"isolation_tag":          tag,
		"isolation_env_file":     envVarsFile,
	}).Info("Isolated solution")

	return targetDir, tag, nil
}

// DetermineTagEnvFile returns the tag value and the optional env file path.
// Note that the --env-file flag has priority over the FSOC_SOLUTION_TAG env var and the .tag file, just like --tag.
// The priority is:
//  1. --tag value or --stable flag
//  2. env file if specified explicitly with the --env-file flag
//  3. FSOC_SOLUTION_TAG env var
//  4. .tag file in the solution directory
//  5. env.json file in the solution directory (implied name)
//
// Deprecated:
// This code duplicates the logic of getEmbeddedTag() to allow support for env files.
// The function, along with the entire source file, will be removed once the pseudo-isolation support is removed.
// This function is exported for use by `melt model` when modeling data from pseudo-isolated solutions.
func DetermineTagEnvFile(cmd *cobra.Command, sourceDir string) (string, string) {
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

	// prepare the env file filename, but don't read it yet
	envFileSpecified := cmd.Flags().Changed("env-file")
	envFilePath := ""
	if envFileSpecified {
		envFilePath, _ = cmd.Flags().GetString("env-file")
	} else {
		envFilePath = filepath.Join(sourceDir, "env.json")
	}

	// ignore other methods if env file is explicitly specified
	if !envFileSpecified {
		// if env var with tag is defined, it overrides env file
		envTag, found := os.LookupEnv("FSOC_SOLUTION_TAG")
		if found {
			return envTag, ""
		}

		tagFile := filepath.Join(sourceDir, TagFileName)
		tagBytes, err := os.ReadFile(tagFile) // ok if no file or empty file
		if err == nil {
			return strings.TrimSpace(string(tagBytes)), ""
		}
	}

	// return env file
	if envFilePath != "" {
		_, err := os.Stat(envFilePath)
		if err == nil {
			return "", envFilePath
		}
	}
	if envFileSpecified {
		log.Fatalf("Env file %q specified but not found", envFilePath)
	}

	log.Fatalf("A tag for pseudo-isolation must be specified (--tag, --stable, FSOC_SOLUTION_TAG env var, .tag or env.json file)")
	return "", "" // should never happen, keep linters happy
}
