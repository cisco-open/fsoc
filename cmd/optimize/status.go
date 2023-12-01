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

package optimize

import (
	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/cmdkit"
	"github.com/cisco-open/fsoc/output"
)

func init() {
	// TODO move this logic to optimize root when implementing unit tests
	optimizeCmd.AddCommand(NewCmdStatus())
}

func NewCmdStatus() *cobra.Command {
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "List onboarded optimizer configuration and status",
		Long: `
List optimization status and configuration

If no flags are provided, all onboarded optimizations will be listed
You can optionally filter optimizations by cluster, namespace and/or workload name
You may also specify a particular optimizer ID to fetch details for a single optimization (recommended with -o detail or -o yaml)
`,
		Example:          "fsoc optimize status --workload-name frontend",
		Args:             cobra.NoArgs,
		RunE:             listStatus,
		TraverseChildren: true,
		Annotations: map[string]string{
			output.TableFieldsAnnotation:  "OPTIMIZERID: .id, WORKLOADNAME: .data.optimizer.target.k8sDeployment.workloadName, STATUS: .data.optimizerState, SUSPENDED: .data.suspended, STAGE: .data.optimizationState, AGENT: .data.agentState, TUNING: .data.tuningState, BLOCKERS: (.data.optimizer.ignoredBlockers? // \"false\" | select(. == \"false\") // \"true\")",
			output.DetailFieldsAnnotation: "OPTIMIZERID: .id, CONTAINER: .data.optimizer.target.k8sDeployment.containerName, WORKLOADNAME: .data.optimizer.target.k8sDeployment.workloadName, NAMESPACE: .data.optimizer.target.k8sDeployment.namespaceName, CLUSTER: .data.optimizer.target.k8sDeployment.clusterName, STATUS: .data.optimizerState, SUSPENDED: .data.suspended, SUSPENSIONS: .data.optimizer.suspensions, RESTARTEDAT: .data.optimizer.restartTimestamp, STAGE: .data.optimizationState, AGENT: .data.agentState, TUNING: .data.tuningState, BLOCKERS: (.data.optimizer.ignoredBlockers?.blockers? // {} | keys)",
		},
	}
	statusCmd.Flags().StringP("cluster", "c", "", "Filter statuses by kubernetes cluster name")
	statusCmd.Flags().StringP("namespace", "n", "", "Filter statuses by kubernetes namespace")
	statusCmd.Flags().StringP("workload-name", "w", "", "Filter statuses by name of kubernetes workload")

	statusCmd.Flags().StringP("optimizer-id", "i", "", "Retrieve status for a specific optimizer by its ID (best used with -o detail)")
	statusCmd.MarkFlagsMutuallyExclusive("optimizer-id", "cluster")
	statusCmd.MarkFlagsMutuallyExclusive("optimizer-id", "namespace")
	statusCmd.MarkFlagsMutuallyExclusive("optimizer-id", "workload-name")

	statusCmd.Flags().StringP("solution-name", "", "optimize", "Intended for developer usage, overrides the name of the solution defining the Orion types for reading/writing")
	if err := statusCmd.LocalFlags().MarkHidden("solution-name"); err != nil {
		log.Warnf("Failed to set statusCmd solution-name flag hidden: %v", err)
	}

	registerOptimizerCompletion(statusCmd, optimizerFlagCluster)
	registerOptimizerCompletion(statusCmd, optimizerFlagNamespace)
	registerOptimizerCompletion(statusCmd, optimizerFlagOptimizerId)
	registerOptimizerCompletion(statusCmd, optimizerFlagWorkloadName)

	return statusCmd
}

func listStatus(cmd *cobra.Command, args []string) error {
	objStoreUrl := getKnowledgeURL(cmd, "status", "data.optimizer")

	headers := getOrionTenantHeaders()
	cmdkit.FetchAndPrint(cmd, objStoreUrl, &cmdkit.FetchAndPrintOptions{Headers: headers, IsCollection: true})
	return nil
}
