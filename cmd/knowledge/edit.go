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

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/cmdkit/editor"
	"github.com/cisco-open/fsoc/platform/api"
)

func editObjectCmd() *cobra.Command {

	editCmd := &cobra.Command{
		Use:     "edit",
		Short:   "Edit an knowledge object from the default editor.",
		Aliases: []string{"e"},
		Long: `The edit command allows you to edit an existent knowledge object in the the editor defined by EDITOR
	environment variable, or fall back to 'vi' for Linux/MacOS or 'notepad' for Windows.`,

		Args:             cobra.NoArgs,
		Run:              editObject,
		TraverseChildren: true,
	}

	editCmd.Flags().
		String("type", "", "The fully qualified type name of the related knowledge object to edit. The fully qualified type name follows the format solutionName:typeName (e.g. extensibility:solution)")
	_ = editCmd.MarkFlagRequired("type")
	_ = editCmd.RegisterFlagCompletionFunc("type", typeCompletionFunc)

	editCmd.Flags().
		String("object-id", "", "The id of the knowledge object to edit")
	_ = editCmd.MarkFlagRequired("object-id")
	_ = editCmd.RegisterFlagCompletionFunc("object-id", objectCompletionFunc)

	editCmd.Flags().
		String("layer-type", fmt.Sprintf("%v", tenant), fmt.Sprintf("Layer type at which the knowledge object exists. Valid values: %q, %q, %q, %q, %q", solution, account, globalUser, tenant, localUser))
	_ = editCmd.MarkPersistentFlagRequired("layer-type")
	_ = editCmd.RegisterFlagCompletionFunc("layer-type", layerTypeCompletionFunc)

	editCmd.Flags().
		Bool("include-tags", false, "Include knowledge object tags in the response from the Knowledge Store")

	editCmd.Flags().
		String("layer-id", "", "The layer-id of the knowledge object to update. Optional for TENANT and SOLUTION layers ")

	return editCmd

}

func editObject(cmd *cobra.Command, args []string) {
	log.Info("Fetching object...")

	fqtn, objID, layerID, layerType, err := parseObjectInfo(cmd)
	if err != nil {
		log.Fatal(err.Error())
	}

	var includeTagsString string = "false"
	includeTagsFlag, _ := cmd.Flags().GetBool("include-tags")

	if includeTagsFlag {
		includeTagsString = "true"
	}

	headers := map[string]string{
		"layer-type":  layerType,
		"layer-id":    layerID,
		"includeTags": includeTagsString,
	}
	httpOptions := &api.Options{Headers: headers}
	url := getObjectUrl(fqtn, objID)

	var res KSObject
	err = api.JSONGet(url, &res, httpOptions)
	if err != nil {
		log.Fatalf("Failed to fetch object: %v", err)
	}

	log.Infof("Object data %vn", res.Data)

	etagHeader := httpOptions.ResponseHeaders["Etag"]
	if len(etagHeader) != 1 || etagHeader[0] == "" {
		log.Fatalf("etag not found in response headers")
	}
	etag := etagHeader[0]
	log.Infof("Object Etag: %s", etag)

	buf := new(bytes.Buffer)
	encoder := json.NewEncoder(buf)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(res.Data)
	if err != nil {
		log.Fatalf("Failed to JSON encode object data before editting: %v", err)
	}

	edited, err := editor.Run(buf)
	if err != nil {
		log.Fatalf("Failed to run editor: %v", err)
	}

	// Parse edited to make sure it is valid json
	var editedData map[string]interface{}
	err = json.Unmarshal(edited, &editedData)
	if err != nil {
		log.Fatalf("Edited data is not valid json: %v", err)
	}

	// Send update to server, with etag
	headersPut := map[string]string{
		"layer-type":  layerType,
		"layer-id":    layerID,
		"If-Match":    etag,
		"includeTags": includeTagsString,
	}
	var resPut any
	err = api.JSONPut(url, editedData, &resPut, &api.Options{Headers: headersPut})
	if err != nil {
		log.Fatalf("Knowledge object update failed: %v", err)
	}

	// TODO: If there is an error, open the editor again with the error message

	log.Infof("Successfully updated object, got output %v\n", resPut)

}
