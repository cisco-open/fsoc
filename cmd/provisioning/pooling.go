package provisioning

import (
	"fmt"
	"os"
	"slices"
	"time"

	"github.com/briandowns/spinner"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/output"
)

func workflowProgressPolling(cmd *cobra.Command, tenantId string, workflowId string, timeout time.Duration) (WorkflowResponse, error) {
	deadline := time.Now().Add(timeout)
	workflow, err := getWorkflow(tenantId, workflowId)
	newState := workflow.State
	var prevState string
	throbber := spinner.New(spinner.CharSets[21], 50*time.Millisecond, spinner.WithWriterFile(os.Stderr))
	_ = throbber.Color("cyan")
	throbber.Start()
	for err == nil && !isFinalState(workflow) && !hasWorkflowInProgress(workflow.Tenant) && time.Now().Before(deadline) {
		if prevState != newState {
			// it shows only unique workflow states
			output.PrintCmdStatus(cmd, fmt.Sprintf("Workflow is in progress: %v.\n", newState))
		}
		prevState = newState
		// wait for next status check
		time.Sleep(10 * time.Second)
		workflow, err = getWorkflow(tenantId, workflowId)
		newState = workflow.State
	}
	throbber.Stop()
	return workflow, err
}

// We need to check tenant has been updated as well otherwise new workflow cannot be started.
// See LIP-940.
func hasWorkflowInProgress(tenant TenantDetails) bool {
	return tenant.HasWorkflowInProgress
}

// Final state is enough criteria of the workflow end.
func isFinalState(workflow WorkflowResponse) bool {
	finalStates := []string{successState, "ERROR"}
	return slices.Contains(finalStates, workflow.State)
}
