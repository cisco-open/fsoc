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
	"fmt"
	"net/url"
	"reflect"

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/cmd/config"
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
	IsSubscribed bool `json:"isSubscribed,omitempty"`
}

var solutionStatusCmd = &cobra.Command{
	Use:   "status <solution-name> [flags]",
	Args:  cobra.MaximumNArgs(1),
	Short: "Get the installation/upload status of a solution",
	Long:  `This command provides the ability to see the current installation and upload status of a solution.`,
	Example: `  fsoc solution status spacefleet
  fsoc solution status spacefleet --status-type=install`,
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

	solutionStatusCmd.Flags().
		String("tag", "stable", "The tag associated with the solution for which you would like to view the status for.  Defaults to 'stable' unless otherwise specified")

	return solutionStatusCmd
}

func getObjects(url string, headers map[string]string) StatusItem {
	var res ResponseBlob
	var emptyData StatusItem

	err := api.JSONGet(url, &res, &api.Options{Headers: headers})

	if err != nil {
		log.Fatalf("Error fetching status object %q: %v", url, err)
	}

	if len(res.Items) > 0 {
		return res.Items[0]
	} else {
		return emptyData
	}
}

func getExtensibilitySolutionObject(url string, headers map[string]string) ExtensibilitySolutionObjectData {
	var res GetExtensibilitySolutionObjectByIdResponse

	err := api.JSONGet(url, &res, &api.Options{Headers: headers})

	if err != nil {
		log.Fatalf("Error fetching extensibility:solution object %q: %v", url, err)
	}

	return res.Data

}

func fetchValuesAndPrint(operation string, solutionNameAndVersionQuery string, subscriptionStatusQuery string, solutionName string, requestHeaders map[string]string, cmd *cobra.Command) {
	// finalize solution name (incl. the solution object name which includes the tag value)
	var solutionNameWithTag string
	solutionTag, _ := cmd.Flags().GetString("tag")
	if solutionTag != "stable" {
		solutionNameWithTag = fmt.Sprintf(`%s.%s`, solutionName, solutionTag)
	} else {
		solutionNameWithTag = solutionName
	}

	uploadStatusChan := make(chan StatusItem)
	installStatusChan := make(chan StatusItem)
	solutionStatusChan := make(chan ExtensibilitySolutionObjectData)

	go func() {
		uploadStatusChan <- getObjects(fmt.Sprintf(getSolutionReleaseUrl(), solutionNameAndVersionQuery), requestHeaders)
	}()
	go func() {
		installStatusChan <- getObjects(fmt.Sprintf(getSolutionInstallUrl(), solutionNameAndVersionQuery), requestHeaders)
	}()
	go func() {
		solutionStatusChan <- getExtensibilitySolutionObject(fmt.Sprintf(getExtensibilitySolutionUrl(), solutionNameWithTag), requestHeaders)
	}()

	uploadStatusItem := <-uploadStatusChan
	installStatusItem := <-installStatusChan
	solutionStatusItem := <-solutionStatusChan

	installStatusData := installStatusItem.StatusData
	uploadStatusData := uploadStatusItem.StatusData
	uploadStatusTimestamp := uploadStatusItem.CreatedAt
	isTenantSubscribedToSolution := (!solutionStatusItem.IsEmpty() && solutionStatusItem.IsSubscribed)

	headers := []string{"Solution Name"}
	values := []string{solutionName}

	appendValue := func(header, value string) {
		headers = append(headers, header)
		values = append(values, value)
	}

	if isTenantSubscribedToSolution {
		appendValue("Solution Subscription Status", "Subscribed")
	} else {
		appendValue("Solution Subscription Status", "Not Subscribed")
	}

	if operation == "upload" {
		appendValue("Solution Upload Version", uploadStatusData.SolutionVersion)
		appendValue("Upload Timestamp", uploadStatusTimestamp)
	} else if operation == "install" {
		appendValue("Solution Install Version", installStatusData.SolutionVersion)
		if isTenantSubscribedToSolution {
			appendValue("Solution Install Successful?", fmt.Sprintf("%v", installStatusData.SuccessfulInstall))
		} else {
			appendValue("Solution Install Successful?", "")
		}
		appendValue("Solution Install Time", installStatusData.InstallTime)
		appendValue("Solution Install Message", installStatusData.InstallMessage)
	} else {
		appendValue("Solution Upload Version", uploadStatusData.SolutionVersion)
		appendValue("Upload Timestamp", uploadStatusTimestamp)
		appendValue("Solution Install Version", installStatusData.SolutionVersion)
		if isTenantSubscribedToSolution {
			appendValue("Solution Install Successful?", fmt.Sprintf("%v", installStatusData.SuccessfulInstall))
		} else {
			appendValue("Solution Install Successful?", "")
		}
		appendValue("Solution Install Time", installStatusData.InstallTime)
		appendValue("Solution Install Message", installStatusData.InstallMessage)
	}

	output.PrintCmdOutputCustom(cmd, installStatusData, &output.PrintRequest{Table: &output.Table{
		Headers: headers,
		Lines:   [][]string{values},
		Detail:  true,
	}})
}

func getSolutionStatus(cmd *cobra.Command, args []string) error {
	var solutionVersionFilter string
	var solutionNameFilter string
	var solutionTagFilter string
	var solutionNameAndVersionFilter string
	var solutionNameAndTenantIdFilter string
	cfg := config.GetCurrentContext()

	layerType := "TENANT"
	solutionName := getSolutionNameFromArgs(cmd, args, "name")

	headers := map[string]string{
		"layer-type": layerType,
		"layer-id":   cfg.Tenant,
	}
	solutionVersion, _ := cmd.Flags().GetString("solution-version")
	statusTypeToFetch, _ := cmd.Flags().GetString("status-type")
	solutionTag, _ := cmd.Flags().GetString("tag")
	solutionVersionFilter = fmt.Sprintf(`data.solutionVersion eq "%s"`, solutionVersion)
	solutionNameFilter = fmt.Sprintf(`data.solutionName eq "%s"`, solutionName)
	solutionTagFilter = fmt.Sprintf(`data.tag eq "%s"`, solutionTag)
	log.Infof(`value of solution tag filter: %s`, solutionTagFilter)

	if solutionVersion != "" {
		solutionNameAndVersionFilter = fmt.Sprintf(`%s and %s and %s`, solutionNameFilter, solutionVersionFilter, solutionTagFilter)
	} else {
		solutionNameAndVersionFilter = fmt.Sprintf(`%s and %s`, solutionNameFilter, solutionTagFilter)
	}
	solutionNameAndTenantIdFilter = fmt.Sprintf(`%s and data.tenantId eq "%s" and %s`, solutionNameFilter, cfg.Tenant, solutionTagFilter)

	solutionNameAndVersionQuery := fmt.Sprintf("?order=%s&filter=%s&max=1", url.QueryEscape("desc"), url.QueryEscape(solutionNameAndVersionFilter))
	solutionNameAndTenantIdQuery := fmt.Sprintf("?order=%s&filter=%s", url.QueryEscape("desc"), url.QueryEscape(solutionNameAndTenantIdFilter))

	log.Infof(`solution name and version query: %s`, solutionNameAndVersionQuery)
	log.Infof(`solution name and tenant id query: %s`, solutionNameAndTenantIdQuery)

	fetchValuesAndPrint(statusTypeToFetch, solutionNameAndVersionQuery, solutionNameAndTenantIdQuery, solutionName, headers, cmd)

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

func getExtensibilitySolutionUrl() string {
	return "knowledge-store/v1/objects/extensibility:solution/%s"
}
