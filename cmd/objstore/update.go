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
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/output"
	"github.com/cisco-open/fsoc/platform/api"
)

var objStoreUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update an existent knowledge object",
	Long: `This command allows the an existent knowledge object to be updated according to the fields and values provided in a .json file.

	Usage:
	fsoc objstore update --type=<fully-qualified-typename> 
	--object-id=<object id>
	--object-file=<fully-qualified-path> 
	--layer-type=[SOLUTION|ACCOUNT|GLOBALUSER|TENANT|LOCALUSER]
	--layer-id=<respective-layer-id>
	
	Flags/Options:
	--type - Flag to indicate the fully qualified type name of the object that you would like to update
	--object-id - Flag to indicate the ID of the object that you want to update
	--object-file - Flag to indicate the fully qualified path (from your root directory) to the file containing the definition of the object that you want to update. Please note that update internally calls HTTP PUT so you will need to specify all fields in the object (even if you are updating just one field)
	--layer-type - Flag to indicate the layer at which the object you would like to update exists
	--layer-id - OPTIONAL Flag to specify a custom layer ID for the object that you would like to update.  This is calculated automatically for all layers currently supported but can be overridden with this flag`,

	Args:             cobra.ExactArgs(0),
	Run:              updateObject,
	TraverseChildren: true,
}

func getUpdateObjectCmd() *cobra.Command {
	objStoreUpdateCmd.Flags().
		String("type", "", "The fully qualified type name of the object")
	_ = objStoreUpdateCmd.MarkPersistentFlagRequired("type")

	objStoreUpdateCmd.Flags().
		String("object-id", "", "The id of the knowledge object been updated")
	_ = objStoreUpdateCmd.MarkPersistentFlagRequired("type")

	objStoreUpdateCmd.Flags().
		String("object-file", "", "The fully qualified path to the json file containing the knowledge object data definition")
	_ = objStoreUpdateCmd.MarkPersistentFlagRequired("objectFile")

	objStoreUpdateCmd.Flags().
		String("layer-type", "", "The layer-type of the updated object")
	_ = objStoreUpdateCmd.MarkPersistentFlagRequired("layer-type")

	objStoreUpdateCmd.Flags().
		String("layer-id", "", "The layer-id of the updated object. Optional for TENANT and SOLUTION layers ")

	return objStoreUpdateCmd

}

func updateObject(cmd *cobra.Command, args []string) {
	objType, _ := cmd.Flags().GetString("type")

	objJsonFilePath, _ := cmd.Flags().GetString("object-file")
	objectFile, err := os.Open(objJsonFilePath)
	if err != nil {
		log.Errorf("Can't find the object definition file named %s", objJsonFilePath)
		return
	}
	defer objectFile.Close()

	objectBytes, _ := io.ReadAll(objectFile)
	var objectStruct map[string]interface{}
	err = json.Unmarshal(objectBytes, &objectStruct)
	if err != nil {
		log.Errorf("Can't generate a %s object from the %s file. Make sure the object definition has all the required field and is valid according to the type definition.")
		return
	}

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

	output.PrintCmdStatus(cmd, fmt.Sprintf("Replacing object %s with the new definition from %s \n", objId, objJsonFilePath))
	err = api.JSONPut(objectUrl, objectStruct, &res, &api.Options{Headers: headers})
	if err != nil {
		log.Errorf("Solution command failed: %v", err.Error())
		return
	}
	output.PrintCmdStatus(cmd, "Object replacement was done successfully!\n")
}
