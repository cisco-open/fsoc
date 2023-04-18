// Copyright 2022 Cisco Systems, Inc.
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
	// "archive/zip"
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/cmd/config"
	"github.com/cisco-open/fsoc/output"
	"github.com/cisco-open/fsoc/platform/api"
)

var solutionPushCmd = &cobra.Command{
	Use:   "push",
	Args:  cobra.ExactArgs(0),
	Short: "Deploy your solution",
	Long: `This command allows the current tenant specified in the profile to deploy a solution to the FSO Platform.
The solution manifest for the solution must be in the current directory.`,
	Example: `
  fsoc solution push
  fsoc solution push -w
  fsoc solution push -w=60
  fsoc solution push --solution-tag=dev`,
	Run:              pushSolution,
	TraverseChildren: true,
}

func getSolutionPushCmd() *cobra.Command {

	solutionPushCmd.Flags().IntP("wait", "w", -1, "Wait (in seconds) for the solution to be deployed (not supported when uisng --solution-bundle)")
	solutionPushCmd.Flag("wait").NoOptDefVal = "300"

	solutionPushCmd.Flags().
		BoolP("bump", "b", false, "Increment the patch version before deploying")

	solutionPushCmd.Flags().
		String("solution-tag", "stable", "Tag to associate with provided solution bundle.  If no value is provided, it will default to 'stable'.")

	solutionPushCmd.Flags().
		String("solution-bundle", "", "fully qualified path name for the solution bundle .zip file")
	_ = solutionPushCmd.Flags().MarkDeprecated("solution-bundle", "it is no longer available.")
	solutionPushCmd.MarkFlagsMutuallyExclusive("solution-bundle", "wait")
	solutionPushCmd.MarkFlagsMutuallyExclusive("solution-bundle", "bump")

	return solutionPushCmd

}

func pushSolution(cmd *cobra.Command, args []string) {
	manifestPath := ""
	var solutionName string
	var solutionVersion string

	waitFlag, _ := cmd.Flags().GetInt("wait")
	bumpFlag, _ := cmd.Flags().GetBool("bump")
	solutionTagFlag, _ := cmd.Flags().GetString("solution-tag")
	solutionBundlePath, _ := cmd.Flags().GetString("solution-bundle")
	var solutionArchivePath string

	if solutionBundlePath != "" {
		log.Fatalf("The --solution-bundle flag is no longer available; please use direct push instead.")
	}

	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatal(err.Error())
	}
	manifestPath = currentDir
	if !isSolutionPackageRoot(manifestPath) {
		log.Fatalf("No solution manifest found in %q; please run this command in a folder with a solution or use the --solution-bundle flag", manifestPath)
	}

	manifest, err := getSolutionManifest(manifestPath)
	if err != nil {
		log.Fatalf("Failed to read the solution manifest in %q: %v", manifestPath, err)
	}

	if bumpFlag {
		if err := bumpManifestPatchVersion(manifest); err != nil {
			log.Fatal(err.Error())
		}
		if err := writeSolutionManifest(manifestPath, manifest); err != nil {
			log.Fatalf("Failed to update solution manifest in %q after version bump: %v", manifestPath, err)
		}
		output.PrintCmdStatus(cmd, fmt.Sprintf("Solution version updated to %v\n", manifest.SolutionVersion))
	}

	solutionName = manifest.Name
	solutionVersion = manifest.SolutionVersion

	// create a temporary solution archive
	// solutionArchive := generateZipNoCmd(manifestPath)
	solutionArchive := generateZip(cmd, manifestPath)
	solutionArchivePath = filepath.Base(solutionArchive.Name())

	message := fmt.Sprintf("Deploying solution %s version %s with tag %s", manifest.Name, manifest.SolutionVersion, solutionTagFlag)
	log.WithFields(log.Fields{"solution": manifest.Name, "version": manifest.SolutionVersion}).Info("Deploying solution")

	file, err := os.Open(solutionArchivePath)
	if err != nil {
		log.Fatalf("Failed to open file %q: %v", solutionArchivePath, err)
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	fw, err := writer.CreateFormFile("file", solutionArchivePath)
	if err != nil {
		log.Fatalf("Failed to create form file: %v", err)
	}

	_, err = io.Copy(fw, file)
	if err != nil {
		log.Fatalf("Failed to copy file %q into file writer: %v", solutionArchivePath, err)
	}

	writer.Close()

	headers := map[string]string{
		"tag":          solutionTagFlag,
		"operation":    "UPLOAD",
		"Content-Type": writer.FormDataContentType(),
	}

	var res any

	output.PrintCmdStatus(cmd, fmt.Sprintf("%v\n", message))

	err = api.HTTPPost(getSolutionPushUrl(), body.Bytes(), &res, &api.Options{Headers: headers})

	if err != nil {
		log.Fatalf("Solution command failed: %v", err)
	}

	if waitFlag >= 0 && solutionName != "" && solutionVersion != "" {
		var duration string
		if waitFlag > 0 {
			duration = fmt.Sprintf("for %d seconds", waitFlag)
		} else {
			duration = "indefinitely"
		}
		fmt.Printf("Waiting %s for solution %s version %s to be installed...", duration, solutionName, solutionVersion)

		filter := fmt.Sprintf(`data.solutionName eq "%s" and data.solutionVersion eq "%s"`, solutionName, solutionVersion)
		query := fmt.Sprintf("?order=%s&filter=%s&max=1", url.QueryEscape("desc"), url.QueryEscape(filter))

		headers := map[string]string{
			"layer-type": "TENANT",
			"layer-id":   config.GetCurrentContext().Tenant,
		}
		var statusData StatusData
		waitStartTime := time.Now()
		for statusData.SolutionVersion != solutionVersion {
			if waitFlag > 0 {
				if time.Since(waitStartTime).Seconds() > float64(waitFlag) {
					fmt.Println("Timeout")
					log.Fatalf("Failed to validate solution %s version %s was installed", solutionName, solutionVersion)
				}
			}
			fmt.Printf(".")
			status := getObject(fmt.Sprintf(getSolutionInstallUrl(), query), headers)
			statusData = status.StatusData
			time.Sleep(3 * time.Second)
		}
		if !statusData.SuccessfulInstall {
			fmt.Println("Failed")
			log.Fatalf("Installation failed: %s", statusData.InstallMessage)
		}
		fmt.Println(" Done")
	}
	message = fmt.Sprintf("Solution %s version %s was successfully deployed.", manifest.Name, manifest.SolutionVersion)
	output.PrintCmdStatus(cmd, message)
}

func getSolutionPushUrl() string {
	return "solnmgmt/v1beta/solutions"
}
