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

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/output"
	"github.com/cisco-open/fsoc/platform/api"
)

var objStoreDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete an existent knowledge object",
	Long: `This command allows an existent knowledge object to be deleted.

	Usage:
	fsoc objstore delete --type=<fully-qualified-typename> 
	--object-id=<object id>
	--layer-type=[SOLUTION|ACCOUNT|GLOBALUSER|TENANT|LOCALUSER]
	--layer-id=<respective-layer-id>
	
	Flags/Options:
	--type - Flag to indicate the fully qualified type name of the object that you would like to delete
	--object-id - Flag to indicate the ID of the object which you would like to delete
	--layer-type - Flag to indicate the layer at which the object you would like to delete currently exists
	--layer-id - OPTIONAL Flag to specify a custom layer ID for the object that you would like to delete.  This is calculated automatically for all layers currently supported but can be overridden with this flag`,

	`,

	Args:             cobra.ExactArgs(0),
	Run:              deleteObject,
	TraverseChildren: true,
}

func getDeleteObjectCmd() *cobra.Command {
	objStoreDeleteCmd.Flags().
		String("type", "", "The fully qualified type name of the object")
	_ = objStoreDeleteCmd.MarkPersistentFlagRequired("type")

	objStoreDeleteCmd.Flags().
		String("object-id", "", "The id of the knowledge object been updated")
	_ = objStoreDeleteCmd.MarkPersistentFlagRequired("type")

	objStoreDeleteCmd.Flags().
		String("layer-type", "", "The layer-type of the updated object")
	_ = objStoreDeleteCmd.MarkPersistentFlagRequired("layer-type")

	objStoreDeleteCmd.Flags().
		String("layer-id", "", "The layer-id of the updated object. Optional for TENANT and SOLUTION layers ")

	return objStoreDeleteCmd

}

func deleteObject(cmd *cobra.Command, args []string) {
	var err error

	objType, _ := cmd.Flags().GetString("type")

	layerType, _ := cmd.Flags().GetString("layer-type")
	layerID := getCorrectLayerID(layerType, objType)

	if layerID == "" {
		if !cmd.Flags().Changed("layer-id") {
			log.Error("Unable to set layer-id flag from given context. Please specify a unique layer-id value with the --layer-id flag")
			return
		}
		layerID, err = cmd.Flags().GetString("layer-id")
		if err != nil {
			log.Errorf("error trying to get %q flag value: %w", "layer-id", err)
			return
		}
	}

	headers := map[string]string{
		"layer-type": layerType,
		"layer-id":   layerID,
	}

	var res any
	objId, _ := cmd.Flags().GetString("object-id")
	urlStrf := getObjStoreObjectUrl() + "/%s/%s"
	objectUrl := fmt.Sprintf(urlStrf, objType, objId)

	output.PrintCmdStatus(cmd, (fmt.Sprintf("Deleting object %s of type  %s \n", objId, objType)))
	err = api.JSONDelete(objectUrl, &res, &api.Options{Headers: headers})
	if err != nil {
		log.Errorf("Solution command failed: %v", err.Error())
		return
	}
	output.PrintCmdStatus(cmd, "Object was successfully deleted!\n")
}
