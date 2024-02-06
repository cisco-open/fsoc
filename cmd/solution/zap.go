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
	Args:  cobra.MaximumNArgs(1),
	Short: "Upload an empty version of a solution to clean it up",
	Long: `This command creates an empty version of an existing solution and uploads it.

This is for the purpose of cleaning up a solution by removing the knowledge types and knowledge objects
associated with it.  Use this command with caution.`,
	Example:          `  fsoc solution zap mysolution`,
	Run:              zapSolution,
	Annotations:      map[string]string{config.AnnotationForConfigBypass: ""},
	TraverseChildren: true,
}

func getSolutionZapCmd() *cobra.Command {

	solutionZapCmd.Flags().
		String("tag", "", "Tag associated with the solution to zap (required)")

	solutionZapCmd.Flags().
		Int("wait", -1, "Wait to terminate the command until the solution is install succesfully")

	solutionZapCmd.Flags().
		Bool("yes", false, "Skip warning message and bypass confirmation step")

	return solutionZapCmd
}

func zapSolution(cmd *cobra.Command, args []string) {
	var confirmationAnswer string
	var solutionPathInFlag string
	var solutionId string
	var solutionInstallObjectQuery string
	var lastSolutionInstallVersion string
	var solutionName string

	solutionInstallObjectChan := make(chan StatusItem)
	solutionObjectChan := make(chan ExtensibilitySolutionObjectData)

	cfg := config.GetCurrentContext()
	solutionTag, _ := cmd.Flags().GetString("tag")
	skipConfirmationMessage, _ := cmd.Flags().GetBool("yes")
	cmd.Flags().StringVar(&solutionName, "solution-name", "", "")
	cmd.Flags().StringVar(&solutionPathInFlag, "solution-bundle", "", "")
	cmd.Flags().StringVar(&lastSolutionInstallVersion, "solution-version", "", "")

	solutionName = getSolutionNameFromArgs(cmd, args, "")
	solutionName = strings.ToLower(solutionName)

	if !skipConfirmationMessage {
		fmt.Print(fmt.Sprintf("Please type YES and hit enter confirm that you want to zap the solution with name: %s and tag: %s ", solutionName, solutionTag))
		fmt.Scanln(&confirmationAnswer)
	
		if confirmationAnswer != "YES" {
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
	log.Infof(`solution type: %s`, solutionType)
	if solutionInstallObjectData.IsEmpty() {
		log.WithFields(
			log.Fields{"solution": solutionName, "tag": solutionTag, "solutionID": solutionId},
		).Info("Previously installed version of the solution not detected.  Install with version 1.0.0.")
		lastSolutionInstallVersion = "1.0.0"
	} else {
		lastSolutionInstallVersion = solutionInstallObjectData.SolutionVersion
	}
	log.Infof(`last solution install version: %s`, lastSolutionInstallVersion)

	if err := os.Mkdir(solutionName, os.ModePerm); err != nil {
		log.Fatal(err.Error())
	}

	manifest := createInitialSolutionManifest(solutionName, solutionType, lastSolutionInstallVersion)
	if err := bumpManifestPatchVersion(manifest); err != nil {
		log.Fatalf(err.Error())
	}
	createSolutionManifestFile(solutionName, manifest)

	currentDirectory, err := os.Getwd()
	if err != nil {
		log.Fatal(err.Error())
	}

	solutionRootDirectory := filepath.Join(currentDirectory, solutionName)
	log.Infof(`current solutionRootDirectoryPath: %s`, solutionRootDirectory)
	solutionArchive := generateZip(cmd, solutionRootDirectory, "")
	log.Infof(`current solutionArchive path after package in .zap: %v`, solutionArchive.Name())
	solutionPathInFlag = solutionArchive.Name()

	uploadSolution(cmd, true)

	output.PrintCmdStatus(cmd, fmt.Sprintf("Solution with name: %s and tag: %s zapped\n", solutionName, solutionTag))

	// remove the temporary blank solution created so that the zap command can be run again in the same directory without
	// any hiccups
	os.RemoveAll(solutionRootDirectory)
}

func (s StatusData) IsEmpty() bool {
	return reflect.DeepEqual(s, StatusData{})
}
