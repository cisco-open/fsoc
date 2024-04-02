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
	"net/url"
	"reflect"
	"strings"

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/config"
	"github.com/cisco-open/fsoc/output"
	"github.com/cisco-open/fsoc/platform/api"
)

type StatusData struct {
	InstallTime       string `json:"installTime,omitempty"`
	InstallMessage    string `json:"installMessage,omitempty"`
	SuccessfulInstall bool   `json:"isSuccessful,omitempty"`
	SolutionName      string `json:"solutionName,omitempty"`
	SolutionVersion   string `json:"solutionVersion,omitempty"`
}

type StatusItem struct {
	StatusData StatusData `json:"data"`
	CreatedAt  string     `json:"createdAt"`
}

type ResponseBlob struct {
	Items []StatusItem `json:"items"`
}

type GetExtensibilitySolutionObjectByIdResponse struct {
	Id        string                          `json:"id,omitempty"`
	LayerId   string                          `json:"layerId,omitempty"`
	LayerType string                          `json:"layerType,omitempty"`
	Data      ExtensibilitySolutionObjectData `json:"data,omitempty"`
}

type ExtensibilitySolutionObjectData struct {
	IsSubscribed bool   `json:"isSubscribed,omitempty"`
	SolutionType string `json:"solutionType,omitempty"`
}

var solutionStatusCmd = &cobra.Command{
	Use:   "status <solution-name> [flags]",
	Args:  cobra.MaximumNArgs(1),
	Short: "Get the status of a solution",
	Long:  `This command provides the ability to see the installation and upload status of a solution.`,
	Example: `  fsoc solution status spacefleet
  fsoc solution status spacefleet --solution-version 1.0.0`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := getSolutionStatus(cmd, args); err != nil {
			log.Fatalf(err.Error())
		}
	},
	TraverseChildren: true,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		config.SetActiveProfile(cmd, args, false)
		return getSolutionNames(toComplete), cobra.ShellCompDirectiveDefault
	},
}

func getSolutionStatusCmd() *cobra.Command {
	solutionStatusCmd.Flags().
		String("name", "", "The name of the solution for which you would like to retrieve the upload status")
	_ = solutionStatusCmd.Flags().MarkDeprecated("name", "please use argument instead.")

	solutionStatusCmd.Flags().
		String("solution-version", "", "The version of the solution for which you would like to retrieve the upload status")
	solutionStatusCmd.Flags().
		String("status-type", "", "The status type that you want to see.  This can be one of [upload, install, all] and will default to all if not specified")

	_ = solutionStatusCmd.Flags().MarkDeprecated("status-type", "full status will be displayed from now on by default.")

	solutionStatusCmd.Flags().
		String("tag", "", "The tag associated with the solution for which you would like to view the status for")

	return solutionStatusCmd
}

func getObjects(url string, headers map[string]string) StatusItem {
	var res ResponseBlob
	var emptyData StatusItem

	err := api.JSONGet(url, &res, &api.Options{Headers: headers})

	if err != nil {
		log.Fatalf("Error fetching solution object %q: %v", url, err)
	}

	if len(res.Items) > 0 {
		return res.Items[0]
	} else {
		return emptyData
	}
}

func getExtensibilitySolutionObject(url string, headers map[string]string) (ExtensibilitySolutionObjectData, error) {
	var res GetExtensibilitySolutionObjectByIdResponse

	err := api.JSONGet(url, &res, &api.Options{Headers: headers, ExpectedErrors: []int{404}})

	if err != nil {
		return ExtensibilitySolutionObjectData{}, err
	}

	return res.Data, nil
}

func checkIfSolutionDeleted(solutionName string, solutionTag string) (bool, SolutionDeletionData) {
	log.Infof("Checking if solution with name: %s and tag: %s has been deleted", solutionName, solutionTag)
	solutionDeletionObject := getSolutionDeletionObject(solutionTag, solutionName)

	if solutionDeletionObject.IsEmpty() {
		log.Infof("Solution with name: %s and tag: %s not found in deletion object", solutionName, solutionTag)
		return false, SolutionDeletionData{}
	} else {
		return true, solutionDeletionObject.DeletionData
	}
}

func fetchInstallationAndReleaseObjects(solutionReleaseObjectQuery string, solutionInstallObjectQuery string, successfulSolutionInstallObjectQuery string, requestHeaders map[string]string) (StatusItem, StatusItem, StatusItem) {
	// Initialize channels for each type of status
	uploadStatusChan := make(chan StatusItem)
	installStatusChan := make(chan StatusItem)
	successfulSolutionInstallStatusChan := make(chan StatusItem)

	// Launch goroutines to fetch status objects in parallel
	go func() {
		uploadStatusChan <- getObjects(fmt.Sprintf(getSolutionReleaseUrl(), solutionReleaseObjectQuery), requestHeaders)
	}()
	go func() {
		installStatusChan <- getObjects(fmt.Sprintf(getSolutionInstallUrl(), solutionInstallObjectQuery), requestHeaders)
	}()
	go func() {
		successfulSolutionInstallStatusChan <- getObjects(fmt.Sprintf(getSolutionInstallUrl(), successfulSolutionInstallObjectQuery), requestHeaders)
	}()

	// Wait for and receive the status objects from the channels
	uploadStatusItem := <-uploadStatusChan
	installStatusItem := <-installStatusChan
	successfulInstallStatusItem := <-successfulSolutionInstallStatusChan

	// Return the received status objects
	return uploadStatusItem, installStatusItem, successfulInstallStatusItem

}

func getSolutionStatus(cmd *cobra.Command, args []string) error {
	var solutionVersionFilter string
	var solutionIDFilter string
	var solutionInstallSuccessfulFilter string
	var solutionInstallObjectFilter string
	var solutionReleaseObjectFilter string
	var lastSuccesfulInstallFilter string
	var solutionID string
	cfg := config.GetCurrentContext()

	layerType := "TENANT"
	solutionName := getSolutionNameFromArgs(cmd, args, "name")
	solutionTag, _ := cmd.Flags().GetString("tag")

	requestHeaders := map[string]string{
		"layer-type": layerType,
		"layer-id":   cfg.Tenant,
	}
	if solutionTag == "dev" || solutionTag == "stable" || solutionTag == "" {
		solutionID = solutionName
	} else {
		solutionID = fmt.Sprintf(`%s.%s`, solutionName, solutionTag)
	}
	solutionVersion, _ := cmd.Flags().GetString("solution-version")
	solutionInstallSuccessfulFilter = `data.isSuccessful eq "true"`
	solutionVersionFilter = fmt.Sprintf(`data.solutionVersion eq "%s"`, solutionVersion)
	solutionIDFilter = fmt.Sprintf(`data.solutionID eq "%s"`, solutionID)

	if solutionVersion != "" {
		solutionInstallObjectFilter = fmt.Sprintf(`%s and %s`, solutionVersionFilter, solutionIDFilter)
	} else {
		solutionInstallObjectFilter = solutionIDFilter
	}

	lastSuccesfulInstallFilter = fmt.Sprintf(`%s and %s`, solutionIDFilter, solutionInstallSuccessfulFilter)
	solutionReleaseObjectFilter = solutionInstallObjectFilter

	solutionInstallObjectQuery := fmt.Sprintf("?order=%s&filter=%s&max=1", url.QueryEscape("desc"), url.QueryEscape(solutionInstallObjectFilter))
	successfulSolutionInstallObjectQuery := fmt.Sprintf("?order=%s&filter=%s&max=1", url.QueryEscape("desc"), url.QueryEscape(lastSuccesfulInstallFilter))
	solutionReleaseObjectQuery := fmt.Sprintf("?order=%s&filter=%s&max=1", url.QueryEscape("desc"), url.QueryEscape(solutionReleaseObjectFilter))

	// finalize solution name (incl. the solution object name which includes the tag value)
	var solutionInstallationMessagePrefix string

	// ensure that the solution exists.
	// This also ensures that the user is logged in (to avoid a race condition on login for the subsequent parallel API calls)
	solutionStatusItem, err := getExtensibilitySolutionObject(getSolutionObjectUrl(solutionID), requestHeaders)

	if err != nil {
		isSolutionDeleted, solutionDeletionData := checkIfSolutionDeleted(solutionName, solutionTag)
		// If solution has been deleted previously, print out a helpful message to let the user know
		// else throw an error
		if isSolutionDeleted {
			if solutionDeletionData.Status == "successful" {
				output.PrintCmdStatus(cmd, fmt.Sprintf("Solution with name: %s and tag: %s previously deleted successfully.  \nA new solution with this name and tag can be uploaded.\n", solutionName, solutionTag))
			} else if solutionDeletionData.Status == "failed" {
				output.PrintCmdStatus(cmd, fmt.Sprintf("Solution with name: %s and tag: %s previously deleted but delete was not successful.  Delete message: %s  \nPlease try again.\n", solutionName, solutionTag, solutionDeletionData.DeleteMessage))
			} else {
				output.PrintCmdStatus(cmd, fmt.Sprintf("Deletion for solution with name: %s and tag: %s currently in progress.  \nPlease wait until the deletion completes for an updated status.\n", solutionName, solutionTag))
			}
			return nil
		} else {
			if strings.Contains(err.Error(), "404") {
				log.Fatalf("Solution with name: %s and tag: %s not found", solutionName, solutionTag)
			} else {
				log.Fatalf("Error fetching extensibility:solution object %q: %v", getSolutionObjectUrl(solutionID), err)
			}

		}
	}

	uploadStatusItem, installStatusItem, successfulInstallStatusItem := fetchInstallationAndReleaseObjects(solutionReleaseObjectQuery, solutionInstallObjectQuery, successfulSolutionInstallObjectQuery, requestHeaders)

	// process status & display
	installStatusData := installStatusItem.StatusData
	successfulInstallStatusData := successfulInstallStatusItem.StatusData
	uploadStatusData := uploadStatusItem.StatusData
	uploadStatusTimestamp := uploadStatusItem.CreatedAt
	isTenantSubscribedToSolution := (!solutionStatusItem.IsEmpty() && solutionStatusItem.IsSubscribed)

	if solutionVersion != "" {
		solutionInstallationMessagePrefix = "Viewing Solution"
	} else {
		solutionInstallationMessagePrefix = "Current Solution"
	}
	headers := []string{"Solution Name"}
	values := []string{solutionID}

	appendValue := func(header, value string) {
		headers = append(headers, header)
		values = append(values, value)
	}

	if solutionTag != "" {
		appendValue("Solution Tag", solutionTag)
	} else {
		if !strings.Contains(solutionID, ".") {
			appendValue("Solution Tag", "stable")
		} else {
			appendValue("Solution Tag", strings.SplitAfter(solutionID, ".")[1])
		}
	}

	if isTenantSubscribedToSolution {
		appendValue("Solution Subscription Status", "Subscribed")
	} else {
		appendValue("Solution Subscription Status", "Not Subscribed")
	}

	appendValue(fmt.Sprintf("%s Upload Version", solutionInstallationMessagePrefix), uploadStatusData.SolutionVersion)
	appendValue(fmt.Sprintf("%s Upload Timestamp", solutionInstallationMessagePrefix), uploadStatusTimestamp)
	appendValue("Last Successful Install Version", successfulInstallStatusData.SolutionVersion)
	appendValue(fmt.Sprintf("%s Install Version", solutionInstallationMessagePrefix), installStatusData.SolutionVersion)
	if isTenantSubscribedToSolution && installStatusData.SolutionVersion != "" {
		appendValue(fmt.Sprintf("%s Install Successful?", solutionInstallationMessagePrefix), fmt.Sprintf("%v", installStatusData.SuccessfulInstall))
	} else {
		appendValue(fmt.Sprintf("%s Install Successful?", solutionInstallationMessagePrefix), "")
	}
	appendValue(fmt.Sprintf("%s Install Time", solutionInstallationMessagePrefix), installStatusData.InstallTime)
	appendValue(fmt.Sprintf("%s Install Message", solutionInstallationMessagePrefix), installStatusData.InstallMessage)

	output.PrintCmdOutputCustom(cmd, installStatusData, &output.Table{
		Headers: headers,
		Lines:   [][]string{values},
		Detail:  true,
	})

	return nil
}

func (s ExtensibilitySolutionObjectData) IsEmpty() bool {
	return reflect.DeepEqual(s, ExtensibilitySolutionObjectData{})
}

func getSolutionReleaseUrl() string {
	return "knowledge-store/v1/objects/extensibility:solutionRelease%s"
}

func getSolutionInstallUrl() string {
	return "knowledge-store/v1/objects/extensibility:solutionInstall%s"
}
