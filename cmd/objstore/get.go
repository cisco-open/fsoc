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

package objstore

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
		Short:   "Fetch an object or a list of objects from the object store.",
		Aliases: []string{"g"},
		Long:    `Fetch an object from object store using set of properties which can uniquely identify it.`,
		Example: `  # Get object [SERVICE principal]
  fsoc obj get --type=extensibility:solution --object=extensibility --layer-id=extensibility --layer-type=SOLUTION
  # Get object [USER principal]
  fsoc obj get --type extensibility:solution --object extensibility --layer-id udayfso@yopmail.com --layer-type LOCALUSER
  # Get list of solution objects that are system solutions
  fsoc obj get --type=extensibility:solution --layer-type=TENANT --filter=(data.isSystem eq true) 
  `,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return getObject(cmd, args, ltFlag)
		},
		TraverseChildren: true,
	}

	// get object

	getCmd.PersistentFlags().
		String("type", "", "Fully qualified type name. It will be formed by combining the solution which defined the type and the type name.")

	getCmd.PersistentFlags().String("object", "", "Object ID to fetch.")
	getCmd.PersistentFlags().String("layer-id", "", "Layer ID object belongs to.")

	getCmd.Flags().
		Var(&ltFlag, "layer-type", fmt.Sprintf("Valid value: %q, %q, %q, %q, %q", solution, account, globalUser, tenant, localUser))

	getCmd.PersistentFlags().String("filter", "", "Filter condition in SCIM filter format for getting objects")
	_ = getCmd.MarkPersistentFlagRequired("type")
	// _ = getCmd.MarkPersistentFlagRequired("object")
	//_ = getCmd.MarkPersistentFlagRequired("layer-id")
	_ = getCmd.MarkPersistentFlagRequired("layer-type")

	_ = getCmd.MarkPersistentFlagRequired("type")

	return getCmd
}

func newGetTypeCmd() *cobra.Command {
	// getTypeCmd represents the get type command
	getTypeCmd := &cobra.Command{
		Use:     "get-type",
		Short:   "Fetch type from object store.",
		Aliases: []string{"gt"},
		Long:    `Fetch type from object store using type name`,
		Example: `# Get type by using fully qualified type name
  fsoc obj get-type --type extensibility:solution`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return getType(cmd, args)
		},
		TraverseChildren: true,
	}

	// get type
	getTypeCmd.PersistentFlags().
		String("type", "", "Fully qualified type name. It will be formed by combining the solution which defined the type and the type name.")

	getTypeCmd.PersistentFlags().String("solution", "", "Name of solution type belongs to.")

	return getTypeCmd
}

func getType(cmd *cobra.Command, args []string) error {
	log.Info("Fetching type...")

	fqtn, err := cmd.Flags().GetString("type")
	if err != nil {
		return fmt.Errorf("error trying to get %q flag value: %w", "type", err)
	}

	/* 	soln, err := cmd.Flags().GetString("solution")
		if err != nil {
			return err
		}

	 	headers := map[string]string{
			"layer-type": "SOLUTION",
			"layer-id":   soln,
		} */

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

	objID, err := cmd.Flags().GetString("object")
	if err != nil {
		return fmt.Errorf("error trying to get %q flag value: %w", "object", err)
	}

	var layerType string = string(ltFlag)
	layerID, _ := cmd.Flags().GetString("layer-id")
	if layerID == "" {
		if layerType == "SOLUTION" {
			return fmt.Errorf("Error: for GET requests made to the SOLUTION layer, please manually supply the layerId flag")
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
			query := fmt.Sprintf("filter=%s", url.QueryEscape(filterCriteria))
			if err != nil {
				log.Errorf("error trying to get %q flag value: %w", "filter", err)
				return nil
			}
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
