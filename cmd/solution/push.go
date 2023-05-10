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
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
	"os"
	"strings"
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
The solution manifest for the solution must be in the current directory.

Important details on solution tags:
(1) A tag must be associated with the solution being uploaded.  All subsequent solution upload requests should use this same tag
(2) Use caution when supplying the tag value to the solution to upload as typos can result in misleading validation results
(3) 'stable' is a reserved tag value keyword for production-ready versions and hence should be used appropriately
(4) For more info on tags, please visit: https://developer.cisco.com/docs/fso/#!tag-a-solution
`,
	Example: `
  fsoc solution push --tag=stable
  fsoc solution push --wait --tag=dev
  fsoc solution push --bump --wait=60
  fsoc solution push --stable --wait`,
	Run:              pushSolution,
	TraverseChildren: true,
}

func getSolutionPushCmd() *cobra.Command {
	solutionPushCmd.Flags().
		String("tag", "", "Free-form string tag to associate with provided solution")

	solutionPushCmd.Flags().
		Bool("stable", false, "Mark the solution as production-ready.  This is equivalent to supplying --tag=stable")

	solutionPushCmd.Flags().IntP("wait", "w", -1, "Wait (in seconds) for the solution to be deployed")
	solutionPushCmd.Flag("wait").NoOptDefVal = "300"

	solutionPushCmd.Flags().
		BoolP("bump", "b", false, "Increment the patch version before deploying")

	solutionPushCmd.Flags().
		String("bundle-path", "", "fully qualified path name for the solution bundle (can be .zip or a folder)")
	solutionPushCmd.MarkFlagsMutuallyExclusive("bundle-path", "wait")
	solutionPushCmd.MarkFlagsMutuallyExclusive("bundle-path", "bump")
	solutionPushCmd.MarkFlagsMutuallyExclusive("tag", "stable")

	return solutionPushCmd
}

func pushSolution(cmd *cobra.Command, args []string) {
	manifestPath := ""
	var message string
	var solutionName string
	var solutionVersion string

	waitFlag, _ := cmd.Flags().GetInt("wait")
	bumpFlag, _ := cmd.Flags().GetBool("bump")
	solutionTagFlag, _ := cmd.Flags().GetString("tag")
	pushWithStableTag, _ := cmd.Flags().GetBool("stable")
	solutionBundlePath, _ := cmd.Flags().GetString("bundle-path")
	var solutionArchivePath string
	var solutionBundleAlreadyZipped bool

	if pushWithStableTag {
		solutionTagFlag = "stable"
	}

	if solutionBundlePath == "" {
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
		message = fmt.Sprintf("Deploying solution with name %s and version %s and tag %s", solutionName, solutionVersion, solutionTagFlag)
	} else {
		manifestPath = solutionBundlePath
		solutionArchivePath = manifestPath
		message = fmt.Sprintf("Zipping and deploying solution specified with path %s with tag %s", solutionArchivePath, solutionTagFlag)
	}

	solutionBundleAlreadyZipped = strings.HasSuffix(solutionArchivePath, ".zip")

	if !solutionBundleAlreadyZipped {
		solutionArchive := generateZip(cmd, manifestPath, "")
		solutionArchivePath = solutionArchive.Name()
	} else {
		message = fmt.Sprintf("Deploying already zipped solution with tag %s", solutionTagFlag)
	}

	log.Infof(message)

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
	message = fmt.Sprintf("Solution with tag %s was successfully deployed.\n", solutionTagFlag)
	output.PrintCmdStatus(cmd, message)
}

func getSolutionPushUrl() string {
	return "solnmgmt/v1beta/solutions"
}
