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
	"net/url"

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/cmdkit"
)

func newGetObjectCmd() *cobra.Command {
	ltFlag := unknown

	// getCmd represents the get object command
	getCmd := &cobra.Command{
		Use:     "get",
		Short:   "Fetch a knowledge object or a list of knowledge objects from the Knowledge Store.",
		Aliases: []string{"g"},
		Long:    `Fetch a knowledge object from the Knowledge Store using set of properties which can uniquely identify it.`,
		Example: `  
  # Get knowledge object [SERVICE principal]
  fsoc knowledge get --type=extensibility:solution --object-id=extensibility --layer-id=extensibility --layer-type=SOLUTION
  
  # Get object [USER principal]
  fsoc knowledge get --type extensibility:solution --object-id extensibility --layer-type LOCALUSER
  
  # Get list of solution layer knowledge objects that are system solutions
  fsoc knowledge get --type=extensibility:solution --layer-type=TENANT --filter="data.isSystem eq true"

  # Get list of objects filtering by a data field
  fsoc knowledge get --type preferences:theme --layer-type TENANT --filter "data.backgroundColor eq \"green\""
  `,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return getObject(cmd, args, ltFlag)
		},
		TraverseChildren: true,
	}

	// get object

	getCmd.PersistentFlags().
		String("type", "", "Fully qualified type name of knowledge object.  The fully qualified type name follows the format solutionName:typeName (e.g. extensibility:solution)")

	getCmd.PersistentFlags().String("object-id", "", "Object ID of the knowledge object to fetch")
	getCmd.PersistentFlags().String("layer-id", "", "Layer ID of the related knowledge object to fetch")

	getCmd.Flags().
		Var(&ltFlag, "layer-type", fmt.Sprintf("Layer type at which the knowledge object exists.  Valid values: %q, %q, %q, %q, %q", solution, account, globalUser, tenant, localUser))

	getCmd.PersistentFlags().String("filter", "", "Filter condition in SCIM filter format for getting knowledge objects")
	_ = getCmd.MarkPersistentFlagRequired("type")
	_ = getCmd.MarkPersistentFlagRequired("layer-type")

	return getCmd
}

func newGetTypeCmd() *cobra.Command {
	// getTypeCmd represents the get type command
	getTypeCmd := &cobra.Command{
		Use:     "get-type",
		Short:   "Fetch type from Knowledge Store.",
		Aliases: []string{"gt"},
		Long:    `Fetch type from Knowledge Store using type name`,
		Example: `# Get type by using fully qualified type name
  fsoc knowledge get-type --type extensibility:solution`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return getType(cmd, args)
		},
		TraverseChildren: true,
	}

	// get type
	getTypeCmd.PersistentFlags().
		String("type", "", "Fully qualified type name of of the type to fetch. It will be formed by combining the solution which defined the type and the type name.")

	// only get type by fqtn is supported.
	_ = getTypeCmd.MarkPersistentFlagRequired("type")

	return getTypeCmd
}

func getType(cmd *cobra.Command, args []string) error {
	log.Info("Fetching type...")

	fqtn, err := cmd.Flags().GetString("type")
	if err != nil {
		return fmt.Errorf("error trying to get %q flag value: %w", "type", err)
	}

	// execute command and print result
	cmdkit.FetchAndPrint(cmd, getTypeUrl(fqtn), nil)
	return nil
}

func getObject(cmd *cobra.Command, args []string, ltFlag layerType) error {
	log.Info("Fetching object...")

	fqtn, err := cmd.Flags().GetString("type")
	if err != nil {
		return fmt.Errorf("error trying to get %q flag value: %w", "type", err)
	}

	objID, err := cmd.Flags().GetString("object-id")
	if err != nil {
		return fmt.Errorf("error trying to get %q flag value: %w", "object-id", err)
	}

	var layerType string = string(ltFlag)
	layerID, _ := cmd.Flags().GetString("layer-id")
	if layerID == "" {
		if layerType == "SOLUTION" {
			return fmt.Errorf("Requests made to the SOLUTION layer require the --layer-id flag")
		} else {
			layerID = getCorrectLayerID(layerType, fqtn)
		}
	}

	headers := map[string]string{
		"layer-type": layerType,
		"layer-id":   layerID,
	}

	// execute command and print output
	var objStoreUrl string
	if objID != "" {
		objStoreUrl = getObjectUrl(fqtn, objID)
	} else {
		if cmd.Flags().Changed("filter") {
			filterCriteria, err := cmd.Flags().GetString("filter")
			if err != nil {
				return fmt.Errorf("error trying to get %q flag value: %w", "filter", err)
			}
			query := fmt.Sprintf("filter=%s", url.QueryEscape(filterCriteria))
			fqtn = fqtn + "?" + query
		}
		objStoreUrl = getObjectListUrl(fqtn)
	}

	cmdkit.FetchAndPrint(cmd, objStoreUrl, &cmdkit.FetchAndPrintOptions{Headers: headers})
	return nil
}

func getTypeUrl(fqtn string) string {
	return fmt.Sprintf("objstore/v1beta/types/%s", fqtn)
}

func getObjectUrl(fqtn, objId string) string {
	return fmt.Sprintf("objstore/v1beta/objects/%s/%s", fqtn, objId)
}

func getObjectListUrl(fqtn string) string {
	return fmt.Sprintf("objstore/v1beta/objects/%s", fqtn)
}

type layerType string

const (
	unknown    layerType = ""
	solution   layerType = "SOLUTION"
	account    layerType = "ACCOUNT"
	globalUser layerType = "GLOBALUSER"
	tenant     layerType = "TENANT"
	localUser  layerType = "LOCALUSER"
)

func (e *layerType) String() string {
	return string(*e)
}

func (e *layerType) Set(v string) error {
	switch v {
	case string(solution), string(account), string(globalUser), string(tenant), string(localUser):
		*e = layerType(v)
		return nil
	default:
		return fmt.Errorf(
			"valid values are: %q, %q, %q, %q, %q",
			solution,
			account,
			globalUser,
			tenant,
			localUser,
		)
	}
}

func (e *layerType) Type() string {
	return "layerType"
}
