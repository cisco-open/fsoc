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
	"bytes"
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
	Short: "Create a new knowledge object of a given type",
	Long: `This command allows the creation of a new knowledge object of a given type in the Knowledge Store.

Example:
  fsoc knowledge create --type<fully-qualified-typename> --object-file=<fully-qualified-path> --layer-type=<valid-layer-type> [--layer-id=<valid-layer-id>]
`,

	Args:             cobra.ExactArgs(0),
	Run:              insertObject,
	TraverseChildren: true,
}

func getCreateObjectCmd() *cobra.Command {
	objStoreInsertCmd.Flags().
		String("type", "", "The fully qualified type name of the knowledge object to create.  The fully qualified type name follows the format solutionName:typeName (e.g. extensibility:solution)")
	_ = objStoreInsertCmd.MarkPersistentFlagRequired("type")
	_ = objStoreInsertCmd.RegisterFlagCompletionFunc("type", typeCompletionFunc)

	objStoreInsertCmd.Flags().
		Bool("include-tags", false, "Include knowledge object tags in the response from the Knowledge Store")

	objStoreInsertCmd.Flags().
		String("object-file", "", "The fully qualified path to the json file containing the knowledge object data")
	_ = objStoreInsertCmd.MarkPersistentFlagRequired("objectFile")

	objStoreInsertCmd.Flags().
		String("layer-type", "", "The layer-type that the created knowledge object will be added to")
	_ = objStoreInsertCmd.MarkPersistentFlagRequired("layer-type")
	_ = objStoreInsertCmd.RegisterFlagCompletionFunc("layer-type", layerTypeCompletionFunc)

	objStoreInsertCmd.Flags().
		String("layer-id", "", "The layer-id that the created knowledge object will be added to. Optional for TENANT and SOLUTION layers ")

	return objStoreInsertCmd

}

func insertObject(cmd *cobra.Command, args []string) {
	objType, _ := cmd.Flags().GetString("type")

	var includeTagsString string = "false"
	includeTagsFlag, _ := cmd.Flags().GetBool("include-tags")

	if includeTagsFlag {
		includeTagsString = "true"
	}

	objJsonFilePath, _ := cmd.Flags().GetString("object-file")
	objectFile, err := os.Open(objJsonFilePath)
	if err != nil {
		log.Fatalf("Can't find the knowledge object definition file named %q", objJsonFilePath)
	}
	defer objectFile.Close()

	objectBytes, _ := io.ReadAll(objectFile)
	var objectStruct map[string]interface{}
	err = json.Unmarshal(objectBytes, &objectStruct)
	if err != nil {
		log.Fatalf("Failed to parse knowledge object data from file %q: %v. Make sure the knowledge object definition has all the required field and is valid according to the type definition.", objJsonFilePath, err)
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
		"layer-type":  layerType,
		"layer-id":    layerID,
		"includeTags": includeTagsString,
	}

	var res any
	// objJsonStr, err := json.Marshal(objectStruct)
	err = api.JSONPost(getObjStoreObjectUrl()+"/"+objType, objectStruct, &res, &api.Options{Headers: headers})
	if err != nil {
		log.Fatalf("Failed to create knowledge object: %v", err)
	} else {
		log.Infof("Successfully created a knowledge object of type: %q", objType)
	}
}

func getObjStoreObjectUrl() string {
	return GetBaseUrl() + "/objects"
}

var objStoreInsertPatchedObjectCmd = &cobra.Command{
	Use:   "create-patch",
	Short: "Create a new patched knowledge object of a given type",
	Long: `This command allows the creation of a new patched knowledge object of a given type in the Knowledge Store.
A patched knowledge object inherits values from separate object that exists at a higher layer and can also override mutable fields when needed.

Example:
  fsoc knowledge create-patch --type<fully-qualified-typename> --object-file=<fully-qualified-path> --target-layer-type=<valid-layer-type> --target-object-id=<valid-object-id>`,

	Args:             cobra.ExactArgs(0),
	Run:              insertPatchObject,
	TraverseChildren: true,
}

func jsonType(in io.Reader) string {
	dec := json.NewDecoder(in)
	// Get just the first valid JSON token from input
	t, err := dec.Token()
	if err != nil {
		panic("Failed to read the first token of provided json")
	}
	if delim, ok := t.(json.Delim); ok {
		// The first token is a delimiter, so this is an array or an object
		switch delim {
		case '[':
			return "array"
		case '{':
			return "object"
		default: // ] or }, shouldn't be possible
			panic("Unexpected delimiter")
		}
	}
	panic("Input does not represent a JSON object or array")
}

func getCreatePatchObjectCmd() *cobra.Command {
	objStoreInsertPatchedObjectCmd.Flags().
		String("type", "", "The fully qualified type name of the knowledge object")
	_ = objStoreInsertPatchedObjectCmd.MarkPersistentFlagRequired("type")

	objStoreInsertPatchedObjectCmd.Flags().
		String("target-object-id", "", "The id of the object for which you want to create a patched knowledge object at a lower layer")
	_ = objStoreInsertPatchedObjectCmd.MarkPersistentFlagRequired("target-object-id")

	objStoreInsertPatchedObjectCmd.Flags().
		String("object-file", "", "The fully qualified path to the json file containing the knowledge object definition")
	_ = objStoreInsertPatchedObjectCmd.MarkPersistentFlagRequired("object-file")

	objStoreInsertPatchedObjectCmd.Flags().
		String("target-layer-type", "", "The layer-type at which the patch knowledge object will be created. For inheritance purposes, this should always be a `lower` layer than the target object's layer")
	_ = objStoreInsertPatchedObjectCmd.MarkPersistentFlagRequired("target-layer-type")

	objStoreInsertPatchedObjectCmd.Flags().Bool(
		"json-patch", false, "Specify this flag if you want to use the JSON Patch (rfc6902). If neither --json-patch nor --json-merge-patch specified, fsoc will interpret it from the provided file content. If the file contains one array of object, the json-patch will be used, otherwise json-merge-patch will be used",
	)

	objStoreInsertPatchedObjectCmd.Flags().Bool(
		"json-merge-patch", false, "Specify this flag if you want to use the JSON Merge Patch (rfc7386). If neither --json-patch nor --json-merge-patch specified, fsoc will interpret it from the provided file content. If the file contains one array of object, the json-patch will be used, otherwise json-merge-patch will be used",
	)

	return objStoreInsertPatchedObjectCmd
}

func insertPatchObject(cmd *cobra.Command, args []string) {
	objType, _ := cmd.Flags().GetString("type")
	parentObjId, _ := cmd.Flags().GetString("target-object-id")

	useJsonPatch, _ := cmd.Flags().GetBool("json-patch")
	useJsonMergePatch, _ := cmd.Flags().GetBool("json-merge-patch")

	if useJsonPatch && useJsonMergePatch {
		log.Fatalf("Both --json-patch and --json-merge-patch specified, please only specify one of them")
		return
	}

	objJsonFilePath, _ := cmd.Flags().GetString("object-file")
	objectFile, err := os.Open(objJsonFilePath)
	if err != nil {
		log.Fatalf("Can't find the knowledge object definition file %q", objJsonFilePath)
		return
	}
	defer objectFile.Close()

	objectBytes, _ := io.ReadAll(objectFile)

	if !useJsonPatch && !useJsonMergePatch {
		t := jsonType(bytes.NewReader(objectBytes))
		if t == "object" {
			useJsonMergePatch = true //nolint:ineffassign
		} else {
			useJsonPatch = true
		}
	}

	layerType, _ := cmd.Flags().GetString("target-layer-type")
	layerID := getCorrectLayerID(layerType, objType)

	headers := map[string]string{
		"layer-type": layerType,
		"layer-id":   layerID,
	}
	if useJsonPatch {
		headers["Content-Type"] = "application/json-patch+json"
	} else {
		headers["Content-Type"] = "application/merge-patch+json"
	}

	var res any
	err = api.JSONPatch(getObjStoreObjectUrl()+"/"+objType+"/"+parentObjId, objectBytes, &res, &api.Options{Headers: headers})
	if err != nil {
		log.Fatalf("Failed to create knowledge object: %v", err)
		return
	} else {
		output.PrintCmdOutput(cmd, fmt.Sprintf("Successfully created a patched knowledge object of type: %q the %s layer.\n", objType, layerType))
	}
}
