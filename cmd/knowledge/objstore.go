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

package knowledge

import (
	"github.com/spf13/cobra"
)

func NewSubCmd() *cobra.Command {
	// objStoreCmd represents the knowledge command
	knowledgeStoreCmd := &cobra.Command{
		Use:     "knowledge",
		Aliases: []string{"obj", "objs", "objstore", "ks"},
		Short:   "Perform Knowledge Store interactions.",
		Long: `

Perform Knowledge Store interactions.
See <docs url>`,
		Example: `# Get knowledge object type
  fsoc knowledge get-type --type=<fully-qualified-type-name>
# Get object
  fsoc knowledge get --type=<fully-qualified-type-name> --object=<objectId> --layer-id=<layerId> --layer-type=SOLUTION|ACCOUNT|GLOBALUSER|TENANT|LOCALUSER
# Get object
  fsoc knowledge create --type=<fully-qualified-typename> --object-file=<fully-qualified-path> --layer-type=SOLUTION|ACCOUNT|GLOBALUSER|TENANT|LOCALUSER [--layer-id=<respective-layer-id>] `,
		TraverseChildren: true,
	}

	knowledgeStoreCmd.AddCommand(newGetObjectCmd())
	knowledgeStoreCmd.AddCommand(newGetTypeCmd())
	knowledgeStoreCmd.AddCommand(getCreateObjectCmd())
	knowledgeStoreCmd.AddCommand(getUpdateObjectCmd())
	knowledgeStoreCmd.AddCommand(getDeleteObjectCmd())
	knowledgeStoreCmd.AddCommand(getCreatePatchObjectCmd())

	return knowledgeStoreCmd
}
