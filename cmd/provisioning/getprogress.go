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

package provisioning

import (
	"fmt"

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/output"
)

func newCmdGetProgress() *cobra.Command {

	var cmd = &cobra.Command{
		Use:   "get-workflow-progress",
		Short: "Get progress of workflow.",
		Example: `
  fsoc provisioning get-progress --tenantId=6d3cd2c0-a2b6-49b9-a41b-f2966eec87ec --workflowId=589f9804-4f43-4236-9c71-0dd8df0c07f3`,
		Aliases:          []string{"get-workflow", "get-progress", "status"},
		Args:             cobra.ExactArgs(0),
		Run:              getWorkflowProgress,
		TraverseChildren: true,
	}
	cmd.Flags().String("tenantId", "", "Tenant Id.")
	_ = cmd.MarkFlagRequired("tenantId")
	cmd.Flags().String("workflowId", "", "Workflow Id.")
	_ = cmd.MarkFlagRequired("workflowId")
	return cmd
}

func getWorkflowProgress(cmd *cobra.Command, _ []string) {
	tenantId, _ := cmd.Flags().GetString("tenantId")
	workflowId, _ := cmd.Flags().GetString("workflowId")
	var workflow, err = getWorkflow(tenantId, workflowId)
	if err == nil {
		if jsonVewEnabled(cmd) {
			output.PrintCmdOutput(cmd, workflow)
		}
		output.PrintCmdStatus(cmd, fmt.Sprintf("Current workflow state is: %v.\n", workflow.StateDescription))
	} else {
		log.Fatal(fmt.Sprintf("Failed to check workflow status because of error. %v", err))
	}
}
