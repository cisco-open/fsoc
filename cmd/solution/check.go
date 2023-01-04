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

	"github.com/cisco-open/fsoc/cmd/config"
	"github.com/cisco-open/fsoc/output"
	"github.com/cisco-open/fsoc/platform/api"
)

var solutionCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Validate your solution component definitions",
	Long: `This command allows the current tenant specified in the profile to build and package a solution bundle to be deployed into the FSO Platform.

Usage:
	fsoc solution check`,
	Args:             cobra.ExactArgs(0),
	Run:              checkSolution,
	TraverseChildren: true,
}

func getSolutionCheckCmd() *cobra.Command {
	solutionCheckCmd.Flags().
		Bool("entities", false, "Validate all the entities and associations components defined in this solution package")

	solutionCheckCmd.Flags().
		Bool("metrics", false, "Validate all the metrics, metricmappings and metricaggregations components defined in this solution package")

	return solutionCheckCmd
}

func checkSolution(cmd *cobra.Command, args []string) {
	var err error

	manifestFile := openFile("manifest.json")
	defer manifestFile.Close()

	manifestBytes, _ := io.ReadAll(manifestFile)
	var manifest *Manifest

	err = json.Unmarshal(manifestBytes, &manifest)
	if err != nil {
		output.PrintCmdStatus("Can't generate a manifest objects from the manifest.json, make sure your manifest.json file is correct.")
		return
	}

	cfg := config.GetCurrentContext()

	objDefs := manifest.Objects
	for _, objDef := range objDefs {
		typeSplit := strings.Split(objDef.Type, ":")
		namespace := typeSplit[0]
		typeName := typeSplit[1]
		if namespace == "fmm" {
			if cmd.Flags().Changed("entities") {
				if typeName == "entity" || typeName == "resourceMapping" || typeName == "associationDeclaration" || typeName == "associationDerivation" {
					checkComponentDef(objDef, cfg)
				}
			}
			if cmd.Flags().Changed("metrics") {
				if typeName == "metric" || typeName == "metricMapping" || typeName == "metricAggregation" {
					checkComponentDef(objDef, cfg)
				}
			}
		}
	}
}

func getTypeUrl(fqtn string) string {
	return fmt.Sprintf("objstore/v1beta/types/%s", fqtn)
}

func Fetch(path string, httpOptions *api.Options) map[string]interface{} {
	// finalize override fields
	var res map[string]interface{}
	// fetch data
	if err := api.JSONGet(path, &res, httpOptions); err != nil {
		log.Fatalf("Platform API call failed: %v", err)
	}
	return res
}

func checkComponentDef(compDef ComponentDef, cfg *config.Context) {
	layerID := cfg.Tenant

	headers := map[string]string{
		"layer-type": "TENANT",
		"layer-id":   layerID,
	}

	objStoreUrl := getTypeUrl(compDef.Type)
	typeDef := Fetch(objStoreUrl, &api.Options{Headers: headers})

	jsonSchema, err := json.Marshal(typeDef["jsonSchema"])
	if err != nil {
		log.Errorf("Couldn't marshal json object to []byte: %v", err)
	}

	schemaLoader := gojsonschema.NewStringLoader(string(jsonSchema))

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
			log.Errorf("Couldn't unmarshal json file content to object array: %v", err)
		}
		for _, object := range jsonArray {
			jsonObject, err := json.Marshal(object)
			if err != nil {
				log.Errorf("Couldn't marshal json object to []byte: %v", err)
			}
			documentLoader := gojsonschema.NewStringLoader(string(jsonObject))
			validate(schemaLoader, documentLoader, compDef)
		}
	} else {
		documentLoader := gojsonschema.NewStringLoader(string(compDefBytes))
		validate(schemaLoader, documentLoader, compDef)
	}

}

func validate(schemaLoader, documentLoader gojsonschema.JSONLoader, compDef ComponentDef) {
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		panic(err.Error())
	}

	if result.Valid() {
		output.PrintCmdStatus(fmt.Sprintf("The components defined in the file %s are valid definitions of type %s \n", compDef.ObjectsFile, compDef.Type))
	} else {
		output.PrintCmdStatus(fmt.Sprintf("The components defined in the file %s are invalid definitions of type %s ! \n", compDef.ObjectsFile, compDef.Type))
		for _, desc := range result.Errors() {
			output.PrintCmdStatus(fmt.Sprintf("- %s\n", desc))
		}
	}
}
