// Copyright 2023 Cisco Systems, Inc.
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

package solution

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/apex/log"
	"github.com/spf13/cobra"
	"github.com/xeipuuv/gojsonschema"

	"github.com/cisco-open/fsoc/config"
	"github.com/cisco-open/fsoc/output"
	"github.com/cisco-open/fsoc/platform/api"
)

var solutionCheckCmd = &cobra.Command{
	Use:              "check",
	Args:             cobra.ExactArgs(0),
	Short:            "Validate your solution component definitions",
	Long:             `This command allows the current tenant specified in the profile to check whether or not each FMM Knowledge Object inside their solution is valid.`,
	Example:          `  fsoc solution check --entities --metrics`,
	Run:              checkSolution,
	TraverseChildren: true,
}

func getSolutionCheckCmd() *cobra.Command {
	solutionCheckCmd.Flags().
		Bool("entities", false, "Validate all the entities and associations components defined in this solution")

	solutionCheckCmd.Flags().
		Bool("metrics", false, "Validate all the metrics, metricmappings and metricaggregations components defined in this solution")

	solutionCheckCmd.Flags().
		Bool("all", false, "Validate all the fmm components defined in this solution")

	return solutionCheckCmd
}

func checkSolution(cmd *cobra.Command, args []string) {
	var err error

	manifestFile := openFile("manifest.json")
	defer manifestFile.Close()

	manifestBytes, err := io.ReadAll(manifestFile)
	if err != nil {
		log.Fatalf("Failed to read manifest.json: %v", err)
	}

	var manifest *Manifest
	err = json.Unmarshal(manifestBytes, &manifest)
	if err != nil {
		log.Fatalf("Failed to parse manifest.json: %v", err)
	}

	cfg := config.GetCurrentContext()

	objDefs := manifest.Objects
	for _, objDef := range objDefs {
		typeSplit := strings.Split(objDef.Type, ":")
		namespace := typeSplit[0]
		typeName := typeSplit[1]
		if namespace == "fmm" {
			if cmd.Flags().Changed("all") {
				checkComponentDef(cmd, objDef, cfg)
			}
			if cmd.Flags().Changed("entities") {
				if typeName == "entity" || typeName == "resourceMapping" || typeName == "associationDeclaration" || typeName == "associationDerivation" {
					checkComponentDef(cmd, objDef, cfg)
				}
			}
			if cmd.Flags().Changed("metrics") {
				if typeName == "metric" || typeName == "metricMapping" || typeName == "metricAggregation" {
					checkComponentDef(cmd, objDef, cfg)
				}
			}
		}
	}
}

func getTypeUrl(fqtn string) string {
	return fmt.Sprintf("knowledge-store/v1/types/%s", fqtn)
}

func Fetch(path string, httpOptions *api.Options) map[string]interface{} {
	// finalize override fields
	var res map[string]interface{}
	// fetch data
	if err := api.JSONGet(path, &res, httpOptions, false); err != nil {
		log.Fatalf("Platform API call failed: %v", err)
	}
	return res
}

func checkComponentDef(cmd *cobra.Command, compDef ComponentDef, cfg *config.Context) {
	layerID := cfg.Tenant

	headers := map[string]string{
		"layer-type": "TENANT",
		"layer-id":   layerID,
	}

	objStoreUrl := getTypeUrl(compDef.Type)
	typeDef := Fetch(objStoreUrl, &api.Options{Headers: headers})

	w := new(bytes.Buffer)
	err := output.WriteJson(typeDef["jsonSchema"], w)
	if err != nil {
		log.Errorf("Couldn't marshal schema to json: %v", err)
	}
	schemaLoader := gojsonschema.NewStringLoader(w.String())

	compDefFile := openFile(compDef.ObjectsFile)
	defer compDefFile.Close()

	compDefBytes, _ := io.ReadAll(compDefFile)

	x := bytes.TrimLeft(compDefBytes, " \t\r\n")

	isArray := len(x) > 0 && x[0] == '['
	// isObject := len(x) > 0 && x[0] == '{'
	if isArray {
		var jsonArray []map[string]interface{}
		err = json.Unmarshal(compDefBytes, &jsonArray)
		if err != nil {
			log.Errorf("Failed to parse component: %v", err)
		}
		for _, object := range jsonArray {
			w := new(bytes.Buffer)
			err = output.WriteJson(object, w)
			if err != nil {
				log.Errorf("Couldn't marshal object to json: %v", err)
			}
			documentLoader := gojsonschema.NewStringLoader(w.String())
			validate(cmd, schemaLoader, documentLoader, compDef)
		}
	} else {
		documentLoader := gojsonschema.NewStringLoader(string(compDefBytes))
		validate(cmd, schemaLoader, documentLoader, compDef)
	}

}

func validate(cmd *cobra.Command, schemaLoader, documentLoader gojsonschema.JSONLoader, compDef ComponentDef) {
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		log.Fatalf("Schema validation failed: %v", err)
	}

	if result.Valid() {
		output.PrintCmdStatus(cmd, fmt.Sprintf("The components defined in the file %q are valid definitions of type %q \n", compDef.ObjectsFile, compDef.Type))
	} else {
		output.PrintCmdStatus(cmd, fmt.Sprintf("The components defined in the file %q are invalid definitions of type %q ! \n", compDef.ObjectsFile, compDef.Type))
		for _, desc := range result.Errors() {
			output.PrintCmdStatus(cmd, fmt.Sprintf("- %s\n", desc))
		}
	}
}
