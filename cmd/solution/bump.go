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

	"github.com/Masterminds/semver/v3"
	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/cmd/config"
	"github.com/cisco-open/fsoc/output"
)

var solutionBumpCmd = &cobra.Command{
	Use:   "bump",
	Short: "Increment the patch version of the solution",
	Long: `Increment the patch version of the solution in the manifest to prepare
it for validation or push.`,
	Example:          `  fsoc solution bump`,
	Args:             cobra.ExactArgs(0),
	Run:              bumpSolutionVersion,
	Annotations:      map[string]string{config.AnnotationForConfigBypass: ""},
	TraverseChildren: true,
}

func getSolutionBumpCmd() *cobra.Command {
	//TODO: consider adding a --minor flag to bump the minor version instead of
	//the patch.
	return solutionBumpCmd
}

func bumpSolutionVersion(cmd *cobra.Command, args []string) {
	manifestDir := "."

	manifest, err := getSolutionManifest(manifestDir)
	if err != nil {
		log.Fatalf("Failed to read solution manifest: %v", err)
	}
	oldVer := manifest.SolutionVersion

	if err = bumpManifestPatchVersion(manifest); err != nil {
		log.Fatalf(err.Error())
	}
	newVer := manifest.SolutionVersion

	if err = writeSolutionManifest(manifestDir, manifest); err != nil {
		log.Fatalf("Failed to update solution manifest: %v", err)
	}

	output.PrintCmdStatus(cmd, fmt.Sprintf("Successfully bumped solution version from %v to %v\n", oldVer, newVer))
}

func bumpManifestPatchVersion(m *Manifest) error {
	ver, err := semver.StrictNewVersion(m.SolutionVersion)
	if err != nil {
		return fmt.Errorf("failed to semver parse solution version %q: %w", m.SolutionVersion, err)
	}

	// refuse to bump if the version has prelease or metadata, as "bump"
	// is not clearly defined in this case (see semver.Version.IncPatch() for details)
	if ver.Prerelease() != "" || ver.Metadata() != "" {
		return fmt.Errorf("cannot bump current version %q because it has prelease and/or metadata info; please set the desired new version manually in the manifest", m.SolutionVersion)
	}

	// bump patch version and update into the manifest
	newVer := ver.IncPatch()
	m.SolutionVersion = newVer.String()

	return nil
}
