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
	"path/filepath"
	"time"

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/cmd/config"
	"github.com/cisco-open/fsoc/output"
	"github.com/cisco-open/fsoc/platform/api"
)

func bumpSolutionVersionInManifest(cmd *cobra.Command, manifest *Manifest, manifestPath string) {
	if err := bumpManifestPatchVersion(manifest); err != nil {
		log.Fatal(err.Error())
	}
	if err := writeSolutionManifest(manifestPath, manifest); err != nil {
		log.Fatalf("Failed to update solution manifest in %q after version bump: %v", manifestPath, err)
	}
	output.PrintCmdStatus(cmd, fmt.Sprintf("Solution version updated to %v\n", manifest.SolutionVersion))
}

func uploadSolution(cmd *cobra.Command, push bool) {
	var err error
	var solutionName string
	var solutionVersion string
	var manifest *Manifest
	var solutionAlreadyZipped bool
	var solutionDisplayText string
	var logFields map[string]interface{}

	waitFlag, err := cmd.Flags().GetInt("wait")
	if err != nil { // if the "wait" flag is not defined for this command, set to no-wait
		waitFlag = -1
	}
	bumpFlag, _ := cmd.Flags().GetBool("bump")
	solutionTagFlag, _ := cmd.Flags().GetString("tag")
	pushWithStableTag, _ := cmd.Flags().GetBool("stable")
	solutionBundlePath, _ := cmd.Flags().GetString("solution-bundle")
	solutionRootDirectory, _ := cmd.Flags().GetString("directory")

	if pushWithStableTag {
		solutionTagFlag = "stable"
	}

	// prepare archive if needed
	solutionAlreadyZipped = solutionBundlePath != ""
	if solutionAlreadyZipped {
		solutionBundlePath = absolutizePath(solutionBundlePath)
		solutionFileName := filepath.Base(solutionBundlePath)
		solutionName = solutionFileName[:len(solutionFileName)-len(filepath.Ext(solutionFileName))] // TODO: extract from archive
		solutionVersion = ""                                                                        // TODO: extract from archive
		solutionDisplayText = fmt.Sprintf("solution archive %q", solutionBundlePath)
		logFields = map[string]interface{}{
			"zip_file":        solutionBundlePath,
			"zip_prepackaged": true,
		}
	} else {
		if solutionRootDirectory == "" {
			solutionRootDirectory, err = os.Getwd()
			if err != nil {
				log.Fatal(err.Error())
			}
		} else {
			solutionRootDirectory, err = filepath.Abs(solutionRootDirectory)
			if err != nil {
				log.Fatal(err.Error())
			}
		}
		if !isSolutionPackageRoot(solutionRootDirectory) {
			log.Fatalf("No solution manifest found in %q; please use -d or --solution-bundle flag", solutionRootDirectory)
		}

		// get manifest, bump version if needed
		manifest, err = getSolutionManifest(solutionRootDirectory)
		if err != nil {
			log.Fatalf("Failed to read the solution manifest from %q: %v", solutionRootDirectory, err)
		}
		if bumpFlag {
			bumpSolutionVersionInManifest(cmd, manifest, solutionRootDirectory)
		}

		// isolate if needed
		solutionIsolateDirectory, err := embeddedConditionalIsolate(cmd, solutionRootDirectory)
		if err != nil {
			log.Fatalf("Failed to isolate solution with tag: %v", err)
		}
		if solutionIsolateDirectory != solutionRootDirectory { // if isolated, post-process
			// set root directory to the isolated version's root
			solutionRootDirectory = solutionIsolateDirectory

			// re-read manifest, to get the isolated name
			manifest, err = getSolutionManifest(solutionRootDirectory)
			if err != nil {
				log.Fatalf("Failed to read the solution manifest from %q: %v", solutionRootDirectory, err)
			}

			// update tag to use supported values
			solutionTagFlag = "stable" // TODO: get it from the env file
		}

		// create archive
		solutionArchive := generateZip(cmd, solutionRootDirectory, "")
		solutionBundlePath = solutionArchive.Name()

		// fill in details
		solutionName = manifest.Name
		solutionVersion = manifest.SolutionVersion
		solutionDisplayText = fmt.Sprintf("solution %s version %s", solutionName, solutionVersion)
		logFields = map[string]interface{}{
			"name":            solutionName,
			"version":         solutionVersion,
			"zip_file":        solutionBundlePath,
			"zip_prepackaged": false,
		}
	}
	solutionDisplayText += " with tag " + solutionTagFlag
	if push {
		output.PrintCmdStatus(cmd, fmt.Sprintf("Deploying %s\n", solutionDisplayText))
	} else {
		output.PrintCmdStatus(cmd, fmt.Sprintf("Validating %s\n", solutionDisplayText))
	}
	log.WithFields(log.Fields(logFields)).Info("Solution details")

	// --- Upload archive

	// read zip file into a buffer
	file, err := os.Open(solutionBundlePath)
	if err != nil {
		log.Fatalf("Failed to open file %q: %v", solutionBundlePath, err)
	}
	defer file.Close()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	fw, err := writer.CreateFormFile("file", solutionBundlePath)
	if err != nil {
		log.Fatalf("Failed to create form file: %v", err)
	}
	_, err = io.Copy(fw, file)
	if err != nil {
		log.Fatalf("Failed to copy file %q into file writer: %v", solutionBundlePath, err)
	}
	writer.Close()

	// send request
	var operation string
	if push {
		operation = "UPLOAD"
	} else {
		operation = "VALIDATE"
	}
	headers := map[string]string{
		"tag":          solutionTagFlag,
		"operation":    operation,
		"Content-Type": writer.FormDataContentType(),
	}
	var res Result
	err = api.HTTPPost(getSolutionPushUrl(), body.Bytes(), &res, &api.Options{Headers: headers})
	if err != nil {
		log.Fatalf("Solution %s command failed: %v", operation, err)
	}
	if !push && !res.Valid {
		message := getSolutionValidationErrorsString(res.Errors.Total, res.Errors)
		output.PrintCmdStatus(cmd, message)
		log.Fatalf("%d error(s) found while validating the solution", res.Errors.Total)
	}

	// display result
	if push {
		output.PrintCmdStatus(cmd, fmt.Sprintf("Successfully uploaded %v.\n", solutionDisplayText))
	} else {
		output.PrintCmdStatus(cmd, fmt.Sprintf("Successfully validated %v.\n", solutionDisplayText))
	}

	// wait for installation, if requested (and possible)
	if push && waitFlag >= 0 && solutionName != "" && solutionVersion != "" {
		var duration string
		if waitFlag > 0 {
			duration = fmt.Sprintf("up to %d seconds", waitFlag)
		} else {
			duration = "indefinitely"
		}
		output.PrintCmdStatus(cmd, fmt.Sprintf("Waiting %s for %s to be installed...\n", duration, solutionDisplayText))

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
					log.Fatalf("Failed to validate %s was installed: timed out", solutionDisplayText)
				}
			}
			status := getObject(fmt.Sprintf(getSolutionInstallUrl(), query), headers)
			statusData = status.StatusData
			time.Sleep(3 * time.Second)
		}
		if !statusData.SuccessfulInstall {
			log.Fatalf("Failed to install %s: %s", solutionDisplayText, statusData.InstallMessage)
		}
		output.PrintCmdStatus(cmd, fmt.Sprintf("Installed %v successfully.\n", solutionDisplayText))
	}
	if cmd.Flag("subscribe").Changed {
		log.WithField("solution", solutionName).Info("Subscribing to solution")
		cfg := config.GetCurrentContext()
		layerID := cfg.Tenant
		headers = map[string]string{
			"layer-type": "TENANT",
			"layer-id":   layerID,
		}
		err = api.JSONPatch(getSolutionSubscribeUrl()+"/"+solutionName, &subscriptionStruct{IsSubscribed: true}, &res, &api.Options{Headers: headers})
		if err != nil {
			log.Fatalf("Solution command failed: %v", err)
		}
		output.PrintCmdStatus(cmd, fmt.Sprintf("Tenant %s has successfully subscribed to solution %s\n", layerID, solutionName))
	}
}

func getSolutionValidationErrorsString(total int, errors Errors) string {
	var message = fmt.Sprintf("\n%d errors detected while validating solution\n", total)
	for _, err := range errors.Items {
		message += fmt.Sprintf("- Error Content: %+v\n", err)
	}
	message += "\n"

	return message
}

func getSolutionPushUrl() string {
	return "solnmgmt/v1beta/solutions"
}
