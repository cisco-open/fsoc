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
	fsoc knowledge update --type=<fully-qualified-typename>
	--object-id=<object id>
	--object-file=<fully-qualified-path>
	--layer-type=[SOLUTION|ACCOUNT|GLOBALUSER|TENANT|LOCALUSER]
	--layer-id=<respective-layer-id>

	Flags/Options:
	--type - Flag to indicate the fully qualified type name of the knowldge object that you would like to update
	--object-id - Flag to indicate the ID of the knowledge object that you want to update
	--object-file - Flag to indicate the fully qualified path (from your root directory) to the file containing the definition of the knowledge object that you want to update. Please note that update internally calls HTTP PUT so you will need to specify all fields in the knowledge object (even if you are updating just one field)
	--layer-type - Flag to indicate the layer at which the knowledge object you would like to update exists
	--layer-id - OPTIONAL Flag to specify a custom layer ID for the knowledge object that you would like to update.  This is calculated automatically for all layers currently supported but can be overridden with this flag`,

	Args:             cobra.ExactArgs(0),
	Run:              updateObject,
	TraverseChildren: true,
}

func getUpdateObjectCmd() *cobra.Command {
	objStoreUpdateCmd.Flags().
		String("type", "", "The fully qualified type name of the related knowledge object to update.  The fully qualified type name follows the format solutionName:typeName (e.g. extensibility:solution)")
	_ = objStoreUpdateCmd.MarkPersistentFlagRequired("type")
	_ = objStoreUpdateCmd.RegisterFlagCompletionFunc("type", typeCompletionFunc)

	objStoreUpdateCmd.Flags().
		String("object-id", "", "The id of the knowledge object to update")
	_ = objStoreUpdateCmd.MarkPersistentFlagRequired("type")
	_ = objStoreUpdateCmd.RegisterFlagCompletionFunc("object-id", objectCompletionFunc)

	objStoreUpdateCmd.Flags().
		String("object-file", "", "The fully qualified path to the json file containing the updated knowledge object data definition")
	_ = objStoreUpdateCmd.MarkPersistentFlagRequired("objectFile")

	objStoreUpdateCmd.Flags().
		String("layer-type", "", "The layer-type of the knowledge object to update")
	_ = objStoreUpdateCmd.MarkPersistentFlagRequired("layer-type")
	_ = objStoreUpdateCmd.RegisterFlagCompletionFunc("layer-type", layerTypeCompletionFunc)

	objStoreUpdateCmd.Flags().
		String("layer-id", "", "The layer-id of the knowledge object to update. Optional for TENANT and SOLUTION layers ")

	return objStoreUpdateCmd

}

func updateObject(cmd *cobra.Command, args []string) {
	objType, _ := cmd.Flags().GetString("type")

	objJsonFilePath, _ := cmd.Flags().GetString("object-file")
	objectFile, err := os.Open(objJsonFilePath)
	if err != nil {
		log.Fatalf("Can't find the knowledge object definition file named %s", objJsonFilePath)
	}
	defer objectFile.Close()

	objectBytes, _ := io.ReadAll(objectFile)
	var objectStruct map[string]interface{}
	err = json.Unmarshal(objectBytes, &objectStruct)
	if err != nil {
		log.Fatalf("Can't parse file %q. Make sure the knowledge object definition has all the required field and is valid according to the type definition.", objJsonFilePath)
	}

	layerType, _ := cmd.Flags().GetString("layer-type")
	layerID := getCorrectLayerID(layerType, objType)

	if layerID == "" {
		if !cmd.Flags().Changed("layer-id") {
			log.Fatal("Unable to set layer-id flag from given context. Please specify a unique layer-id value with the --layer-id flag")
		}
		layerID, err = cmd.Flags().GetString("layer-id")
		if err != nil {
			log.Fatalf("error trying to get %q flag value: %w", "layer-id", err)
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

	output.PrintCmdStatus(cmd, fmt.Sprintf("Replacing knowledge object %q with the new data from %q \n", objId, objJsonFilePath))
	err = api.JSONPut(objectUrl, objectStruct, &res, &api.Options{Headers: headers})
	if err != nil {
		log.Fatalf("Knowledge object update failed: %v", err)
	}
	output.PrintCmdStatus(cmd, "Knowledge object updated successfully.\n")
}
