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
	"time"

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/config"
	"github.com/cisco-open/fsoc/output"
	"github.com/cisco-open/fsoc/platform/api"
)

type SolutionDeletionData struct {
	DeleteTime    string `json:"deleteTime,omitempty"`
	DeleteMessage string `json:"deleteMessage,omitempty"`
	SolutionName  string `json:"solutionName,omitempty"`
	Tag           string `json:"tag,omitempty"`
	Status        string `json:"status,omitempty"`
}

type SolutionDeletionRecord struct {
	DeletionData SolutionDeletionData `json:"data,omitempty"`
	ID           string               `json:"id,omitempty"`
}

type SolutionDeletionResponseBlob struct {
	Items []SolutionDeletionRecord `json:"items"`
}

var solutionDeleteCmd = &cobra.Command{
	Use:   "delete <solution-name>",
	Args:  cobra.ExactArgs(1),
	Short: "Delete a solution.  Stable solutions cannot be deleted",
	Long: `This command deletes a solution uploaded by your tenant.  Stable solutions cannot be deleted.

This is for the purpose of deleting a solution that you no longer want to use.  
This will clean up all of objects/types defined by the solution as well as all of the solution metadata.  
Please note you must terminate all active subscriptions to the solution before issuing this command.
Please also note this is an asynchronous operation and thus it may take some time for the status to reflect properly.
If you issue this command while an active deletion is in progress, it will simply wait for that deletion to finish.`,
	Example:          `  fsoc solution delete mysolution --tag custom --wait 45 --yes`,
	Run:              deleteSolution,
	TraverseChildren: true,
}

func getSolutionDeleteCommand() *cobra.Command {

	solutionDeleteCmd.Flags().
		String("tag", "", "Tag associated with the solution to delete (required)")

	_ = solutionDeleteCmd.MarkFlagRequired("tag")

	solutionDeleteCmd.Flags().
		Int("wait", 60, "Wait to terminate the command until the solution the solution deletion process is completed.  Default time is 60 seconds.")

	solutionDeleteCmd.Flags().
		Bool("no-wait", false, "Don't wait for solution to be deleted after issuing delete request.")

	solutionDeleteCmd.Flags().
		BoolP("yes", "y", false, "Skip warning message and bypass confirmation step")

	solutionDeleteCmd.MarkFlagsMutuallyExclusive("wait", "no-wait")

	return solutionDeleteCmd
}

func deleteSolution(cmd *cobra.Command, args []string) {
	var confirmationAnswer string
	var solutionName string
	var solutionTag string
	var existingSolutionDeletionObjectId string
	var existingSolutionDeletionInProgress bool = false

	solutionTag, _ = cmd.Flags().GetString("tag")
	skipConfirmationMessage, _ := cmd.Flags().GetBool("yes")
	waitForDeletionDuration, _ := cmd.Flags().GetInt("wait")
	noWait, _ := cmd.Flags().GetBool("no-wait")

	solutionName = getSolutionNameFromArgs(cmd, args, "")

	headers := map[string]string{
		"tag": solutionTag,
	}

	if !skipConfirmationMessage {
		fmt.Printf("WARNING! This command will remove all objects and types that are associated with this solution and will purge all data related to those objects and types.  It will also remove all solution metadata (including, but not limited to, subscriptions and other related objects).\nProceed with caution!  \nPlease type the name of the solution you want to delete and hit enter confirm that you want to delete the solution with name: %s and tag: %s \n", solutionName, solutionTag)
		fmt.Scanln(&confirmationAnswer)

		if confirmationAnswer != solutionName {
			log.Fatal("Solution delete not confirmed, exiting command")
		}
	}

	existingDeletionObj := getSolutionDeletionObject(solutionTag, solutionName)

	if !existingDeletionObj.IsEmpty() {
		existingSolutionDeletionObjectId = existingDeletionObj.ID
		if existingDeletionObj.DeletionData.Status == "inProgress" {
			existingSolutionDeletionInProgress = true
		}
	}

	solutionDeleteUrl := fmt.Sprintf(getSolutionDeleteUrl(), solutionName)

	if !existingSolutionDeletionInProgress {
		var res any
		err := api.JSONDelete(solutionDeleteUrl, &res, &api.Options{Headers: headers})
		if err != nil {
			log.Fatalf("Solution delete command failed: %v", err)
		}
	}

	output.PrintCmdStatus(cmd, fmt.Sprintf("Solution deletion initiated for solution with name: %s and tag: %s\n", solutionName, solutionTag))

	if !noWait && waitForDeletionDuration > 0 {
		var deletionObjData SolutionDeletionData
		var newDeletionObjectId string
		waitStartTime := time.Now()

		for (newDeletionObjectId == existingSolutionDeletionObjectId && !existingSolutionDeletionInProgress) || deletionObjData.IsEmpty() || deletionObjData.Status == "inProgress" {
			output.PrintCmdStatus(cmd, fmt.Sprintf("Waited %f seconds for solution with name: %s and tag: %s to be marked as deleted\n", time.Since(waitStartTime).Seconds(), solutionName, solutionTag))
			if time.Since(waitStartTime).Seconds() > float64(waitForDeletionDuration) {
				log.Fatalf("Timed out waiting for solution with name %s and tag: %s to be deleted. Deletion continues, please check status for outcome.", solutionName, solutionTag)
			}
			deletionObj := getSolutionDeletionObject(solutionTag, solutionName)
			deletionObjData = deletionObj.DeletionData
			newDeletionObjectId = deletionObj.ID
			time.Sleep(3 * time.Second)
		}

		if deletionObjData.Status == "successful" {
			output.PrintCmdStatus(cmd, fmt.Sprintf("Solution with name: %s and tag: %s deleted successfully", solutionName, solutionTag))
		} else {
			output.PrintCmdStatus(cmd, fmt.Sprintf("Failed to delete solution with name: %s and tag %s.  Error message: %s", solutionName, solutionTag, deletionObjData.DeleteMessage))
		}
	}
}

func getSolutionDeleteUrl() string {
	return "solution-manager/v1/solutions/%s"
}

func getExtSolutionDeletionUrl() string {
	return "knowledge-store/v2beta/objects/extensibility:solutionDeletion%s"
}

func (s SolutionDeletionData) IsEmpty() bool {
	return reflect.DeepEqual(s, SolutionDeletionData{})
}

func (s SolutionDeletionRecord) IsEmpty() bool {
	return reflect.DeepEqual(s, SolutionDeletionRecord{})
}

func getSolutionDeletionObject(solutionTag string, solutionName string) SolutionDeletionRecord {
	var res SolutionDeletionResponseBlob
	var emptyData SolutionDeletionRecord

	cfg := config.GetCurrentContext()
	layerType := "TENANT"
	headers := map[string]string{
		"layer-type": layerType,
		"layer-id":   cfg.Tenant,
	}

	filter := fmt.Sprintf(`data.solutionName eq "%s" and data.tag eq "%s"`, solutionName, solutionTag)
	query := fmt.Sprintf("?order=%s&filter=%s&max=1", url.QueryEscape("desc"), url.QueryEscape(filter))

	url := fmt.Sprintf(getExtSolutionDeletionUrl(), query)

	err := api.JSONGet(url, &res, &api.Options{Headers: headers})

	if err != nil {
		log.Fatalf("Error fetching solution deletion object %q: %v", url, err)
	}

	if len(res.Items) > 0 {
		return res.Items[0]
	} else {
		return emptyData
	}
}
