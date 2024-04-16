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
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/output"
)

const licenseType = "TRIAL"

func newCmdApplyLicense() *cobra.Command {

	var cmd = &cobra.Command{
		Use:   "dev-license",
		Short: "Simplified license provisioning command.",
		Long: `Provision new trial license for existing tenant with predefined default values. 
This command is intended to be used for testing purposes.`,
		Example: `
  fsoc provisioning dev-license --tenantId=6d3cd2c0-a2b6-49b9-a41b-f2966eec87ec --valid-to=2023-12-18 --units=300`,
		Aliases:          []string{},
		Args:             cobra.ExactArgs(0),
		Run:              provisionDevLicense,
		TraverseChildren: true,
	}
	cmd.Flags().String("tenantId", "", "Tenant Id.")
	_ = cmd.MarkFlagRequired("tenantId")
	cmd.Flags().String("valid-to", "", "Date of the end of new license validity. Format [yyyy-mm-dd]. (By default for 1 year.)")
	cmd.Flags().Uint32("units", 100, "Amount of Platform units.")
	return cmd
}

func provisionDevLicense(cmd *cobra.Command, _ []string) {
	tenantId, _ := cmd.Flags().GetString("tenantId")
	validTo, _ := cmd.Flags().GetString("valid-to")
	units, _ := cmd.Flags().GetUint32("units")

	provisionNewLicense(cmd, tenantId, validTo, units)
}

func provisionNewLicense(cmd *cobra.Command, tenantId string, validTo string, units uint32) {
	request := buildLicenseProvisioningRequest(validTo, units)
	output.PrintCmdStatus(cmd, fmt.Sprintf("Sending license provisioning request: licenseId=%v.\n", request.LicenseId))
	var licRes LicenseProvisioningResponse
	err := postLicenseProvisioningRequest(cmd, tenantId, request, &licRes)
	if err != nil {
		output.PrintCmdOutput(cmd, fmt.Sprintf("Cannot start license provisioning because of an error. %v", err))
		return
	}
	workflowId := licRes.WorkflowId
	timeout := 2 * time.Minute
	workflow, progressErr := workflowProgressPolling(cmd, tenantId, workflowId, timeout)
	if progressErr != nil {
		log.Fatalf("Cannot check workflow status because of error: %v", progressErr)
		return
	}
	if workflow.State == successState {
		output.PrintCmdStatus(cmd, fmt.Sprintf("License has been succesfully provisioned: tenantId=%v, workflowId=%v.\n",
			tenantId, workflowId))
	} else {
		if isFinalState(workflow) {
			output.PrintCmdStatus(cmd, fmt.Sprintf("License provisioning failed, workflow state is: %v.\n",
				workflow.StateDescription))
		} else {
			output.PrintCmdStatus(cmd, fmt.Sprintf("License provisioning took too more than %v, "+
				"current workflow state is: %v. \nPlease use "+
				"[ fsoc provisioning get-progress --tenantId=%v --workflowId=%v ] command to verify further progress.\n",
				timeout, workflow.StateDescription, tenantId, workflow.Id,
			))
		}
		output.PrintCmdStatus(cmd, InternalSupportMessage)
	}
}

func calculateValidTo(validTo string, validFromTime time.Time) time.Time {
	var validToTime time.Time
	if len(validTo) == 0 {
		validToTime = validFromTime.AddDate(1, 0, 0)
	} else {
		validToTime, _ = time.Parse(time.DateOnly, validTo)
	}
	return validToTime
}

func buildLicenseProvisioningRequest(validTo string, units uint32) LicenseProvisioningRequest {
	validFromTime := time.Now().UTC()
	validToTime := calculateValidTo(validTo, validFromTime)
	startOfCurrentMonth := convert(calcStartOfMonth())
	var request = LicenseProvisioningRequest{
		LicenseId: uuid.New().String(),
		Revisions: []Revision{
			{
				RevisionId: uuid.New().String(),
				Validity: Validity{
					From: convert(validFromTime),
					To:   convert(validToTime),
				},
				DataRetention: []DataRetention{
					{
						DataType:        "metrics",
						RetentionPeriod: "P397D",
					},
					{
						DataType:        "events",
						RetentionPeriod: "P30D",
					},
					{
						DataType:        "logs",
						RetentionPeriod: "P30D",
					},
					{
						DataType:        "spans",
						RetentionPeriod: "P14D",
					},
				},
				Entitlement: Entitlement{
					Platform: Platform{
						LicenseType: licenseType,
						Pools: []Pool{
							{
								Name:    "default",
								Units:   uint64(units),
								Overage: 0,
								UsageCycle: UsageCycle{
									Period: "P1M",
									Start:  startOfCurrentMonth,
								},
								Meters: []Meter{
									{
										MeterRef:  "licensing:metrics",
										UnitRatio: 0.00000005,
									},
									{
										MeterRef:  "licensing:events",
										UnitRatio: 0.00000004,
									},
									{
										MeterRef:  "licensing:logs",
										UnitRatio: 0.0000000000232,
									},
									{
										MeterRef:  "licensing:spans",
										UnitRatio: 0.00000003125,
									},
								},
							}},
						Tiers: []Tier{
							{
								TierRef: "licensing:overall",
								Value:   calculateTier(units),
							},
						},
					},
					Solutions: []Solution{
						createCcoSolution(units, startOfCurrentMonth),
						createDemSolution(units, startOfCurrentMonth),
					},
				},
			},
		},
	}
	return request
}

/*
This solution license based on Black Swan best guess.
TODO: In the future, it should be similar to real Trial offer - IDEA-6996.
*/
func createCcoSolution(units uint32, usageCycleStart string) Solution {
	return Solution{
		Name:        "cco",
		LicenseType: licenseType,
		Pools: []Pool{
			{
				Name:    "apm",
				Units:   uint64(units),
				Overage: 0,
				UsageCycle: UsageCycle{
					Period: "P1M",
					Start:  usageCycleStart,
				},
				Meters: []Meter{
					{
						MeterRef:  "cco:service",
						UnitRatio: 1,
					},
					{
						MeterRef:  "cco:database",
						UnitRatio: 1,
					},
				},
			},
			{
				Name:    "infra",
				Units:   uint64(units),
				Overage: 0,
				UsageCycle: UsageCycle{
					Period: "P1M",
					Start:  usageCycleStart,
				},
				Meters: []Meter{
					{
						MeterRef:  "cco:host",
						UnitRatio: 1,
					},
					{
						MeterRef:  "cco:function_invocation",
						UnitRatio: 1,
					},
				},
			},
		},
	}
}

/*
This license is based on the license which MELT QE test is provisioning for test tenant.
TODO: In the future, it should be similar to real Trial offer - IDEA-6996.
*/
func createDemSolution(units uint32, usageCycleStart string) Solution {
	return Solution{
		Name:        "dem",
		LicenseType: licenseType,
		Pools: []Pool{
			{
				Name:    "default",
				Units:   uint64(units),
				Overage: 0,
				UsageCycle: UsageCycle{
					Period: "P1M",
					Start:  usageCycleStart,
				},
			},
			{
				Name:    "session",
				Units:   1000,
				Overage: 10,
				UsageCycle: UsageCycle{
					Period: "P1M",
					Start:  usageCycleStart,
				},
				Meters: []Meter{
					{
						MeterRef:  "brum:session",
						UnitRatio: 0.001,
					},
					{
						MeterRef:  "mrum:session",
						UnitRatio: 0.001,
					},
				},
			},
			{
				Name:    "session-replay",
				Units:   1000,
				Overage: 5,
				UsageCycle: UsageCycle{
					Period: "P1M",
					Start:  usageCycleStart,
				},
				Meters: []Meter{
					{
						MeterRef:  "sessionrpl:session",
						UnitRatio: 0.001,
					},
					{
						MeterRef:  "mrumsessionrpl:session",
						UnitRatio: 0.001,
					},
				},
			},
		},
	}
}

/*
According to https://docs.appdynamics.com/observability/cisco-cloud-observability/en/cisco-cloud-observability-licensing-and-entitlements/licensing-for-cloud-native-application-observability#LicensingforCloudNativeApplicationObservability-IngestionExchangeRates
*/

func convert(fromTime time.Time) string {
	return fromTime.Format(time.RFC3339)
}

func calcStartOfMonth() time.Time {
	now := time.Now().UTC()
	year, month, _ := now.Date()
	return time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
}

/*
According to https://docs.appdynamics.com/observability/cisco-cloud-observability/en/cisco-cloud-observability-licensing-and-entitlements/license-tokens-tiers-and-rate-limits#LicenseTokens,Tiers,andRateLimits-TokensandTiers
*/
func calculateTier(units uint32) int8 {
	switch {
	case units <= 150:
		return 1
	case units <= 300:
		return 2
	case units <= 1200:
		return 3
	default:
		return 4
	}
}
