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
	"errors"
	"fmt"
	"os"
	"slices"

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/config"
	"github.com/cisco-open/fsoc/output"
)

var solutionFixCmd = &cobra.Command{
	Use:   "fix [flags]",
	Args:  cobra.NoArgs,
	Short: "Fixes common solution issues and upgrades solution format version",
	Long: `Rewrites aspects of the solution to:
	1. Upgrade the solution format to recent changes (e.g., manifest version)
	2. Fix common issues in the solution (e.g., missing required fields)
	3. Make common changes and refactoring (e.g., change file format from JSON to YAML)`,
	Example:     `  fsoc solution fix --manifest-format=yaml --manifest-version --solution-type=module`,
	Run:         solutionFix,
	Annotations: map[string]string{config.AnnotationForConfigBypass: ""}, // this command does not require a valid context
}

var ErrNoEffect = errors.New("requested operation is not required as it will have no effect")

var availableFixes = []string{"manifest-version", "manifest-format", "isolate"}

func getSolutionFixCmd() *cobra.Command {
	solutionFixCmd.Flags().
		Bool("manifest-version", false, "Upgrade manifest to the latest format (requires --solution-type)")
	solutionFixCmd.Flags().
		String("manifest-format", "", "Change manifest format (should be json or yaml)")
	solutionFixCmd.Flags().
		String("solution-type", "", "Define type for the solution (should be one of component, module, or application)")
	solutionFixCmd.Flags().
		Bool("isolate", false, "Isolate non-isolated or pseudo-isolated solution")
	solutionFixCmd.Flags().
		Bool("deisolate", false, "De-isolate a natively isolated solution")
	solutionFixCmd.Flags().
		Bool("formatting", false, "Automatically format YAML and JSON files to opinionated solution formatting")

	solutionFixCmd.MarkFlagsMutuallyExclusive("isolate", "deisolate") // either one or the other
	_ = solutionFixCmd.Flags().MarkHidden("deisolate")                // not yet implemented
	_ = solutionFixCmd.Flags().MarkHidden("formatting")               // not yet implemented

	// TODO: consider supporting source directory override
	solutionFixCmd.Flags().
		StringP("directory", "d", "", "Path to the solution root directory (defaults to current dir)")
	_ = solutionFixCmd.Flags().MarkHidden("directory") // not yet implemented

	// TODO: consider supporting fix target-dir as alternative to overwriting the solution
	solutionFixCmd.Flags().
		Bool("target-dir", false, "Write fixed solution to a new directory")
	_ = solutionFixCmd.Flags().MarkHidden("target-dir") // not yet implemented

	return solutionFixCmd
}

func solutionFix(cmd *cobra.Command, args []string) {
	// check that at least one fix was requested
	work := false
	for _, fix := range availableFixes {
		if cmd.Flags().Changed(fix) {
			work = true
			break
		}
	}
	if !work {
		log.Fatal("At least one fix must be requested")
	}

	// collect flags and values
	manifestVersion, _ := cmd.Flags().GetBool("manifest-version")
	manifestFormat, _ := cmd.Flags().GetString("manifest-format")
	solutionType, _ := cmd.Flags().GetString("solution-type")

	// Read solution manifest
	manifest, err := GetManifest(".")
	if err != nil {
		log.WithError(err).Fatal("Failed to read solution manifest. Is this a solution directory?")
	}

	// backup manifest
	manifestBackupPath, err := backupManifest(manifest)
	if err != nil {
		log.WithError(err).Fatal("Failed to back up manifest; canceling fix")
	}
	log.WithField("path", manifestBackupPath).Info("Backed up original manifest")

	// save original format
	oldManifestFormat := manifest.ManifestFormat

	// create a counter for the fixes
	nFixes := 0

	// Upgrade manifest version
	if manifestVersion {
		if solutionType == "" {
			log.Fatal("Solution type is required to upgrade manifest version, use the --solution-type flag to specify.")
		}
		err := upgradeManifestVersion(cmd, manifest, solutionType)
		switch {
		case err == nil:
			output.PrintCmdStatus(cmd, fmt.Sprintf("Manifest version upgraded to %v.\n", manifest.ManifestVersion))
			nFixes += 1
		case errors.Is(err, ErrNoEffect):
			log.Warn("Manifest version is already up to date for this solution; not changed.")
		default:
			log.WithError(err).Fatal("Failed to upgrade manifest version")
		}
	}

	if manifestFormat != "" {
		err := changeManifestFormat(cmd, manifest, manifestFormat)
		switch {
		case err == nil:
			output.PrintCmdStatus(cmd, fmt.Sprintf("Manifest format changed to %v.\n", manifest.ManifestFormat))
			nFixes += 1
		case errors.Is(err, ErrNoEffect):
			log.Warn("Manifest format is already as requested; not changed.")
		default:
			log.WithError(err).Fatal("Failed to change manifest format")
		}
	}

	// TODO: properly support full solution changes
	isolate, _ := cmd.Flags().GetBool("isolate")
	if isolate {
		// read solution
		solution, err := NewSolutionDirectoryContentsFromDisk(".")
		if err != nil {
			log.Fatalf("Failed to load solution contents: %v", err)
		}
		err = fixIsolate(cmd, solution)
		if err != nil {
			log.Fatalf("Failed to isolate solution: %v", err)
		}
	}
	deisolate, _ := cmd.Flags().GetBool("deisolate")
	if deisolate {
		// read solution
		solution, err := NewSolutionDirectoryContentsFromDisk(".")
		if err != nil {
			log.Fatalf("Failed to load solution contents: %v", err)
		}
		err = fixDeisolate(cmd, solution)
		if err != nil {
			log.Fatalf("Failed to isolate solution: %v", err)
		}
	}

	// if no fix was applied, print a message and exit
	if nFixes == 0 {
		output.PrintCmdStatus(cmd, "No changes were made to the solution.\n")
		return
	}

	// update manifest
	err = saveSolutionManifest(".", manifest)
	if err != nil {
		log.WithError(err).Fatal("Failed to write the updated manifest file")
	}
	output.PrintCmdStatus(cmd, fmt.Sprintf("Manifest file manifest.%s updated successfully.\n", manifest.ManifestFormat))

	// if file format changed, delete the old manifest file
	if oldManifestFormat != manifest.ManifestFormat {
		oldManifestPath := fmt.Sprintf("manifest.%s", oldManifestFormat)
		output.PrintCmdStatus(cmd, fmt.Sprintf("Removing old manifest file %q (backed up in %q).\n", oldManifestPath, manifestBackupPath))
		err := os.Remove(oldManifestPath)
		if err != nil {
			log.WithError(err).Warnf("Failed to remove old manifest file %q, please delete manually to avoid conflict", oldManifestPath)
		}
	}

	output.PrintCmdStatus(cmd, fmt.Sprintf("%v change(s) made to the solution.\n", nFixes))
}

func upgradeManifestVersion(cmd *cobra.Command, manifest *Manifest, solutionType string) error {
	// Check if manifest version is already up to date
	if manifest.ManifestVersion == "1.1.0" {
		return ErrNoEffect
	}

	// Check solution type
	if !slices.Contains(knownSolutionTypes, solutionType) {
		return fmt.Errorf("unknown solution type %q; Should be one of %q", solutionType, knownSolutionTypes)
	}

	// Upgrade manifest version
	manifest.ManifestVersion = "1.1.0"
	manifest.SolutionType = solutionType
	// TODO: for type that require additional components, add them here
	var todo string
	switch manifest.SolutionType {
	case "module":
		todo = "add a module object"
	case "application":
		todo = "add an application object"
	}
	if todo != "" {
		log.Warnf("The solution type %q requires additional component(s): please %s", solutionType, todo)
	}
	return nil
}

func changeManifestFormat(cmd *cobra.Command, manifest *Manifest, format string) error {
	// Validate format value
	var fileFormat FileFormat
	switch format {
	case "json":
		fileFormat = FileFormatJSON
	case "yaml":
		fileFormat = FileFormatYAML
	default:
		return fmt.Errorf("unknown manifest format %q; should be json or yaml", format)
	}

	// Check if manifest format is already as requested
	if manifest.ManifestFormat == fileFormat {
		return ErrNoEffect
	}

	// Prevent changing pseudo-isolated solutions to YAML
	if fileFormat == FileFormatYAML {
		return fmt.Errorf("changing pseudo-isolated solutions to YAML is not supported; pseudo-isolation is supported only for JSON-formatted solutions")
	}

	// Change manifest format
	manifest.ManifestFormat = fileFormat
	// nb: writing the format back will change the format

	return nil
}

// backupManifest create a backup of the manifest file. Upon success, it returns the file
// name of the backup file; otherwise, it returns an error.
func backupManifest(manifest *Manifest) (string, error) {
	f, err := os.CreateTemp("", fmt.Sprintf("%s-manifest-backup-*.%s", manifest.Name, manifest.ManifestFormat))
	if err != nil {
		log.WithError(err).Fatal("Failed to create manifest backup file")
	}
	defer f.Close()

	err = writeSolutionManifest(manifest, f)
	if err != nil {
		log.WithError(err).Fatalf("Failed to write manifest backup file %q", f.Name())
	}
	return f.Name(), nil
}
