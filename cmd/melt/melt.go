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

package melt

import (
	"github.com/spf13/cobra"
)

// meltCmd represents the login command
var meltCmd = &cobra.Command{
	Use:              "melt",
	Short:            "Generates fsoc telemetry data models and sends OTLP payloads to the platform ingestion services",
	Long:             "This command generate fsoc telemetry data models and sends the data to the platform ingestion services. \nIt helps developers to generate mock telemetry data to test their solution's domain models.",
	Args:             cobra.ExactArgs(0),
	TraverseChildren: true,
}

func NewSubCmd() *cobra.Command {
	return meltCmd
}
