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
	"fmt"
	"strings"

	"github.com/cisco-open/fsoc/platform/api"
	"github.com/spf13/cobra"
)

func NewSubCmd() *cobra.Command {
	// objStoreCmd represents the knowledge command
	knowledgeStoreCmd := &cobra.Command{
		Use:     "knowledge",
		Aliases: []string{"obj", "objs", "objstore", "ks"},
		Short:   "Perform Knowledge Store interactions.",
		Long: `

Perform Knowledge Store interactions. See https://developer.cisco.com/docs/fso/#!use-the-knowledge-store-introduction
for more information on the Knowledge Store. `,
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

type Type struct {
	Name     string `json:"name"`
	Solution string `json:"solution"`
}

type TypeList struct {
	Items []Type `json:"items"`
}

type Object struct {
	ID string `json:"id"`
}

type ObjectList struct {
	Items []Object `json:"items"`
}

// completion functions
var typeCompletionFunc = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return getTypes(toComplete), cobra.ShellCompDirectiveNoFileComp
}
var objectCompletionFunc = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	typeName, err := cmd.Flags().GetString("type")
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	layerID, _ := cmd.Flags().GetString("layer-id")
	layerTypeFlag := cmd.Flags().Lookup("layer-type") // works with string and enum flags
	if layerTypeFlag == nil {
		return nil, cobra.ShellCompDirectiveError
	}
	layerType := layerTypeFlag.Value.String()

	return getObjectsForType(typeName, layerType, layerID, toComplete), cobra.ShellCompDirectiveNoFileComp
}

var layerTypeCompletionFunc = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{string(solution), string(account), string(globalUser), string(tenant), string(localUser)},
		cobra.ShellCompDirectiveNoFileComp
}

func getObjectsForType(typeName string, lType string, layerID string, prefix string) (objects []string) {

	if lType == "" {
		lType = "TENANT" // might as well default to something
	}

	if layerID == "" {
		if lType == "SOLUTION" {
			return objects // Not suppored
		} else {
			layerID = getCorrectLayerID(lType, typeName)
		}
	}

	headers := map[string]string{
		"layer-type": lType,
		"layer-id":   layerID,
	}

	httpOptions := &api.Options{Headers: headers}

	var res ObjectList
	url := fmt.Sprintf("%s?max=%d", getObjectListUrl(typeName), api.MAX_COMPLETION_RESULTS)
	err := api.JSONGet(url, &res, httpOptions)
	if err != nil {
		return objects
	}

	for _, s := range res.Items {
		if strings.HasPrefix(s.ID, prefix) {
			objects = append(objects, s.ID)
		}
	}

	return objects
}

func getTypes(prefix string) (types []string) {

	var res TypeList
	url := fmt.Sprintf("%s?max=%d", getTypeUrl(""), api.MAX_COMPLETION_RESULTS)
	err := api.JSONGet(url, &res, nil)
	if err != nil {
		return types
	}

	for _, s := range res.Items {
		t := fmt.Sprintf("%s:%s", s.Solution, s.Name)
		if strings.HasPrefix(t, prefix) {
			types = append(types, t)
		}
	}
	return types
}
