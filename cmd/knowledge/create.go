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

var objStoreInsertCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new Knowledge Object of a given type",
	Long: `This command allows the creation of a new Knowledge Object of a given type in the Knowledge Store.

Example:
  fsoc knowledge create --type<fully-qualified-typename> --object-file=<fully-qualified-path> --layer-type=<valid-layer-type> [--layer-id=<valid-layer-id>]
`,

	Args:             cobra.ExactArgs(0),
	Run:              insertObject,
	TraverseChildren: true,
}

func getCreateObjectCmd() *cobra.Command {
	objStoreInsertCmd.Flags().
		String("type", "", "The fully qualified type name of the Knowledge Object to create")
	_ = objStoreInsertCmd.MarkPersistentFlagRequired("type")

	objStoreInsertCmd.Flags().
		String("object-file", "", "The fully qualified path to the json file containing the Knowledge Object data")
	_ = objStoreInsertCmd.MarkPersistentFlagRequired("objectFile")

	objStoreInsertCmd.Flags().
		String("layer-type", "", "The layer-type that the created Knowledge Object will be added to")
	_ = objStoreInsertCmd.MarkPersistentFlagRequired("layer-type")

	objStoreInsertCmd.Flags().
		String("layer-id", "", "The layer-id that the created Knowledge Object will be added to. Optional for TENANT and SOLUTION layers ")

	return objStoreInsertCmd

}

func insertObject(cmd *cobra.Command, args []string) {
	objType, _ := cmd.Flags().GetString("type")

	objJsonFilePath, _ := cmd.Flags().GetString("object-file")
	objectFile, err := os.Open(objJsonFilePath)
	if err != nil {
		log.Fatalf("Can't find the Knowledge Object definition file named %q", objJsonFilePath)
	}
	defer objectFile.Close()

	objectBytes, _ := io.ReadAll(objectFile)
	var objectStruct map[string]interface{}
	err = json.Unmarshal(objectBytes, &objectStruct)
	if err != nil {
		log.Fatalf("Failed to parse Knowledge Object data from file %q: %v. Make sure the Knowledge Object definition has all the required field and is valid according to the type definition.", objJsonFilePath, err)
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
	// objJsonStr, err := json.Marshal(objectStruct)
	err = api.JSONPost(getObjStoreObjectUrl()+"/"+objType, objectStruct, &res, &api.Options{Headers: headers})
	if err != nil {
		log.Fatalf("Failed to create Knowledge Object: %v", err)
	} else {
		log.Infof("Successfully created a Knowledge Object of type: %q", objType)
	}
}

func getObjStoreObjectUrl() string {
	return "objstore/v1beta/objects"
}

var objStoreInsertPatchedObjectCmd = &cobra.Command{
	Use:   "create-patch",
	Short: "Create a new patched Knowledge Object of a given type",
	Long: `This command allows the creation of a new patched Knowledge Object of a given type in the Knowledge Store.
A patched Knowledge Object inherits values from separate object that exists at a higher layer and can also override mutable fields when needed.

Example:
  fsoc knowledge create-patch --type<fully-qualified-typename> --object-file=<fully-qualified-path> --target-layer-type=<valid-layer-type> --target-object-id=<valid-object-id>`,

	Args:             cobra.ExactArgs(0),
	Run:              insertPatchObject,
	TraverseChildren: true,
}

func getCreatePatchObjectCmd() *cobra.Command {
	objStoreInsertPatchedObjectCmd.Flags().
		String("type", "", "The fully qualified type name of the Knowledge Object")
	_ = objStoreInsertPatchedObjectCmd.MarkPersistentFlagRequired("type")

	objStoreInsertPatchedObjectCmd.Flags().
		String("target-object-id", "", "The id of the object for which you want to create a patched Knowledge Object at a lower layer")
	_ = objStoreInsertPatchedObjectCmd.MarkPersistentFlagRequired("target-object-id")

	objStoreInsertPatchedObjectCmd.Flags().
		String("object-file", "", "The fully qualified path to the json file containing the Knowledge Object definition")
	_ = objStoreInsertPatchedObjectCmd.MarkPersistentFlagRequired("objectFile")

	objStoreInsertPatchedObjectCmd.Flags().
		String("target-layer-type", "", "The layer-type at which the patch Knowledge Object will be created. For inheritance purposes, this should always be a `lower` layer than the target object's layer")
	_ = objStoreInsertPatchedObjectCmd.MarkPersistentFlagRequired("target-layer-type")

	return objStoreInsertPatchedObjectCmd
}

func insertPatchObject(cmd *cobra.Command, args []string) {
	objType, _ := cmd.Flags().GetString("type")
	parentObjId, _ := cmd.Flags().GetString("target-object-id")

	objJsonFilePath, _ := cmd.Flags().GetString("object-file")
	objectFile, err := os.Open(objJsonFilePath)
	if err != nil {
		log.Fatalf("Can't find the Knowledge Object definition file %q", objJsonFilePath)
		return
	}
	defer objectFile.Close()

	objectBytes, _ := io.ReadAll(objectFile)
	var objectStruct map[string]interface{}
	err = json.Unmarshal(objectBytes, &objectStruct)
	if err != nil {
		log.Fatalf("Failed to parse Knowledge Object data from file %q: %v. Make sure the Knowledge Object definition has all the required fields and is valid according to the type definition.", objJsonFilePath, err)
	}

	layerType, _ := cmd.Flags().GetString("target-layer-type")
	layerID := getCorrectLayerID(layerType, objType)

	headers := map[string]string{
		"layer-type": layerType,
		"layer-id":   layerID,
	}

	var res any
	err = api.JSONPatch(getObjStoreObjectUrl()+"/"+objType+"/"+parentObjId, objectStruct, &res, &api.Options{Headers: headers})
	if err != nil {
		log.Fatalf("Failed to create Knowledge Object: %v", err)
		return
	} else {
		output.PrintCmdOutput(cmd, fmt.Sprintf("Successfully created a patched Knowledge Object of type: %q the %s layer.\n", objType, layerType))
	}
}
