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

	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/platform/api"
)

// Config defines the subsystem configuration under fsoc
type Config struct {
	ApiVersion api.Version `mapstructure:"apiver,omitempty" fsoc-help:"API version to use for knowledge store commands. The default is \"v1\"."`
}

var GlobalConfig Config

func NewSubCmd() *cobra.Command {
	// objStoreCmd represents the knowledge command
	knowledgeStoreCmd := &cobra.Command{
		Use:     "knowledge",
		Aliases: []string{"ks", "obj", "objs", "objstore"},
		Short:   "Perform Knowledge Store operations",
		Long: `
Perform Knowledge Store operations. "ks" is a convenient alias to the "knowledge" command.

See https://developer.cisco.com/docs/cisco-observability-platform/#!knowledge-store-introduction for more information on the Knowledge Store.

All operations require the type to be specified as a fully-qualified type name (FQTN). FQTN follows the format solutionName:typeName (e.g., extensibility:solution).

All data operations also require specifying the layer type and layer ID. fsoc attempts to provide a default layer ID whenever possible (e.g., the tenant ID for TENANT and user ID for LOCALUSER).

Object IDs often need to be quoted in order to avoid special character interpretation in your shell. They do not need to be URL-escaped (see examples in the "knowledge get" command).

To see the exact API call that fsoc makes, use the --curl flag with the desired command.
`,
		Example: `  # Get knowledge object type
  fsoc knowledge get-type --type=<fully-qualified-type-name>

  # Get object
  fsoc knowledge get --type=<fully-qualified-type-name> --object-id <objectId> --layer-type=SOLUTION|ACCOUNT|GLOBALUSER|TENANT|LOCALUSER [--layer-id=<layerId>]

  # Create object
  fsoc knowledge create --type=<fully-qualified-typename> --object-file=<fully-qualified-path> --layer-type=SOLUTION|ACCOUNT|GLOBALUSER|TENANT|LOCALUSER [--layer-id=<layer-id>]

  # Delete object
  fsoc knowledge delete --type=<fully-qualified-typename> --object-id=<object-id> --layer-type=SOLUTION|ACCOUNT|GLOBALUSER|TENANT|LOCALUSER [--layer-id=<layer-id>]`,
		TraverseChildren: true,
	}

	knowledgeStoreCmd.AddCommand(newGetObjectCmd())
	knowledgeStoreCmd.AddCommand(newGetTypeCmd())
	knowledgeStoreCmd.AddCommand(getCreateObjectCmd())
	knowledgeStoreCmd.AddCommand(getUpdateObjectCmd())
	knowledgeStoreCmd.AddCommand(getDeleteObjectCmd())
	knowledgeStoreCmd.AddCommand(getCreatePatchObjectCmd())
	knowledgeStoreCmd.AddCommand(editObjectCmd())

	return knowledgeStoreCmd
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

	var result api.CollectionResult[KSObject]
	err := api.JSONGetCollection[KSObject](getObjectListUrl(typeName), &result, httpOptions)
	if err != nil {
		return objects
	}

	for _, s := range result.Items {
		if strings.HasPrefix(s.ID, prefix) {
			objects = append(objects, s.ID)
		}
	}

	return objects
}

func getTypes(prefix string) (types []string) {

	var result api.CollectionResult[KSType]
	err := api.JSONGetCollection[KSType](getTypeUrl(""), &result, nil)
	if err != nil {
		return types
	}

	for _, s := range result.Items {
		t := fmt.Sprintf("%s:%s", s.Solution, s.Name)
		if strings.HasPrefix(t, prefix) {
			types = append(types, t)
		}
	}
	return types
}

func parseObjectInfo(cmd *cobra.Command) (typeName string, objectID string, layerID string, layerType string, err error) {
	typeName, err = cmd.Flags().GetString("type")
	if err != nil {
		return "", "", "", "", fmt.Errorf("error trying to get %q flag value: %w", "type", err)
	}

	objectID, err = cmd.Flags().GetString("object-id")
	if err != nil {
		return "", "", "", "", fmt.Errorf("error trying to get %q flag value: %w", "object-id", err)
	}

	layerTypeFlag := cmd.Flags().Lookup("layer-type") // works with string and enum flags
	if layerTypeFlag == nil {
		return "", "", "", "", fmt.Errorf("error trying to get %q flag value: %w", "layer-type", err)
	}
	layerType = layerTypeFlag.Value.String()

	layerID, _ = cmd.Flags().GetString("layer-id")
	if layerID == "" {
		if layerType == "SOLUTION" {
			err = fmt.Errorf("requests made to the SOLUTION layer require the --layer-id flag")
			return "", "", "", "", err
		} else {
			layerID = getCorrectLayerID(layerType, typeName)
		}
	}

	return typeName, objectID, layerID, layerType, nil
}

func GetBaseUrl() string {
	ver := GlobalConfig.ApiVersion.String()
	if ver == "" {
		ver = "v2beta"
	}
	return "knowledge-store/" + ver
}
