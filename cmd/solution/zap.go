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
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/config"
	"github.com/cisco-open/fsoc/output"
)

var solutionZapCmd = &cobra.Command{
	Use:   "zap <solution-name>",
	Args:  cobra.ExactArgs(1),
	Short: "Upload an empty version of a solution to clean it up",
	Long: `This command creates an empty version of an existing solution and uploads it.

This is for the purpose of cleaning up a solution by removing the knowledge types and knowledge objects
associated with it.  Use this command with caution.`,
	Example:          `  fsoc solution zap mysolution`,
	Run:              zapSolution,
	TraverseChildren: true,
}

func getSolutionZapCmd() *cobra.Command {

	solutionZapCmd.Flags().
		String("tag", "", "Tag associated with the solution to zap (required)")
	_ = solutionZapCmd.MarkFlagRequired("tag")

	solutionZapCmd.Flags().
		Int("wait", -1, "Wait to terminate the command until the solution is successfully zapped")

	solutionZapCmd.Flags().
		Bool("yes", false, "Skip warning message and bypass confirmation step")

	return solutionZapCmd
}

func zapSolution(cmd *cobra.Command, args []string) {
	var confirmationAnswer string
	var solutionZipPath string
	var solutionId string
	var solutionInstallObjectQuery string
	var lastSolutionInstallVersion string
	var solutionName string

	solutionInstallObjectChan := make(chan StatusItem)
	solutionObjectChan := make(chan ExtensibilitySolutionObjectData)

	cfg := config.GetCurrentContext()
	solutionTag, _ := cmd.Flags().GetString("tag")
	skipConfirmationMessage, _ := cmd.Flags().GetBool("yes")

	solutionName = getSolutionNameFromArgs(cmd, args, "")
	solutionName = strings.ToLower(solutionName)

	if !skipConfirmationMessage {
		fmt.Printf("WARNING! This command will remove all of the objects and types that are associated with this solution and will purge all data related to those objects and types.  Proceed with caution!  \nPlease type the name of the solution you want to delete and hit enter confirm that you want to zap the solution with name: %s and tag: %s \n", solutionName, solutionTag)
		fmt.Scanln(&confirmationAnswer)

		if confirmationAnswer != solutionName {
			log.Fatal("Solution zap not confirmed, exiting command")
		}
	}

	headers := map[string]string{
		"layer-type": "TENANT",
		"layer-id":   cfg.Tenant,
	}

	if solutionTag == "dev" {
		solutionId = solutionName
	} else if solutionTag == "stable" {
		log.Fatal("Error: stable solutions are not able to be zapped.  Please specify a tag other that 'stable'")
	} else {
		solutionId = fmt.Sprintf("%s.%s", solutionName, solutionTag)
	}

	solutionInstallObjectFilterQuery := fmt.Sprintf(`data.solutionID eq "%s" and data.tag eq "%s"`, solutionId, solutionTag)
	solutionInstallObjectQuery = fmt.Sprintf("?order=%s&filter=%s&max=1", url.QueryEscape("desc"), url.QueryEscape(solutionInstallObjectFilterQuery))

	go func() {
		solutionInstallObjectChan <- getObjects(fmt.Sprintf(getSolutionInstallUrl(), solutionInstallObjectQuery), headers)
	}()
	go func() {
		solutionObjectChan <- getExtensibilitySolutionObject(getSolutionObjectUrl(solutionId), headers)
	}()

	solutionInstallObject := <-solutionInstallObjectChan
	solutionObject := <-solutionObjectChan
	solutionInstallObjectData := solutionInstallObject.StatusData
	solutionType := solutionObject.SolutionType
	if solutionInstallObjectData.IsEmpty() {
		log.WithFields(
			log.Fields{"solution": solutionName, "tag": solutionTag, "solutionID": solutionId},
		).Info("Previously installed version of the solution not detected.  Install with version 1.0.0.")
		lastSolutionInstallVersion = "1.0.0"
	} else {
		lastSolutionInstallVersion = solutionInstallObjectData.SolutionVersion
	}

	tempDirRoot, err := os.MkdirTemp("", "")
	if err != nil {
		log.Fatal(err.Error())
	}
	solutionRootDirectory := filepath.Join(tempDirRoot, solutionName)

	// Create the directory inside the temporary directory
	if err := os.Mkdir(solutionRootDirectory, 0755); err != nil {
		log.Fatal(err.Error())
	}

	// remove the temporary blank solution created so that the zap command can be run again in the same directory without
	// any hiccups
	defer os.RemoveAll(tempDirRoot)

	// create a new solution manifest and bump the version to the next
	manifest := createInitialSolutionManifest(
		solutionName,
		WithSolutionType(solutionType),
		WithSolutionVersion(lastSolutionInstallVersion))
	if err := bumpManifestPatchVersion(manifest); err != nil {
		log.Fatalf(err.Error())
	}
	createSolutionManifestFile(solutionRootDirectory, manifest)
	updatedManifestVersion := manifest.SolutionVersion

	solutionArchive := generateZip(cmd, solutionRootDirectory, "")
	solutionZipPath = solutionArchive.Name()
	defer os.RemoveAll(solutionZipPath)

	// upload the hollowed-out solution
	// Note that since the tag flag is REQUIRED in this command, the FSOC_SOLUTION_TAG env var and any locally present .tag file will be ignored
	uploadSolution(cmd, true, WithSolutionName(solutionName), WithSolutionZipPath(solutionZipPath), WithSolutionInstallVersion(updatedManifestVersion))

	output.PrintCmdStatus(cmd, fmt.Sprintf("Solution with name: %s and tag: %s zapped\n", solutionName, solutionTag))
}

func (s StatusData) IsEmpty() bool {
	return reflect.DeepEqual(s, StatusData{})
}
