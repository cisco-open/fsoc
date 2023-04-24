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
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/apex/log"
	"github.com/cisco-open/fsoc/output"
	"github.com/cisco-open/fsoc/platform/api"
	"github.com/spf13/cobra"
)

var solutionTestCmd = &cobra.Command{
	Use:              "test",
	Args:             cobra.ExactArgs(0),
	Short:            "Test Solution",
	Long:             "This command allows the current tenant specified in the profile to run tests against an already-deployed solution",
	Example:          `  fsoc solution test`,
	Run:              testSolution,
	TraverseChildren: true,
}

func getSolutionTestCmd() *cobra.Command {
	solutionTestCmd.Flags().String("test-bundle", "", "The fully qualified path name for the test bundle directory. If no value is provided, it will default to 'current' - meaning current directory, where this command is running.")
	return solutionTestCmd
}

func testSolution(cmd *cobra.Command, args []string) {
	var testBundleDir string
	testBundlePath, _ := cmd.Flags().GetString("test-bundle")

	// Get Test Bundle Directory
	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatal(err.Error())
	}
	if testBundlePath == "" {
		testBundleDir = currentDir
	} else {
		testBundleDir = testBundlePath
	}
	// fmt.Printf("\nTest Bundle Directory - %s", testBundleDir)

	// Read Test Objects JSON
	if !isTestPackageRoot(testBundleDir) {
		log.Fatalf("No test-objects file found in %q; please run this command in a folder with a test-objects file or use the --test-bundle flag", testBundleDir)
	}
	testObjects, err := getTestObjects(testBundleDir)
	if err != nil {
		log.Fatalf("Failed to read the test objects file in %q: %v", testBundleDir, err)
	}

	// Replace any file references in input, assertions with contents of those files
	for i := range testObjects.Tests {
		testObj := testObjects.Tests[i]
		setup := testObj.Setup
		if setup.Location != "" {
			setup.Input, err = readFileLocation(fmt.Sprintf("%s/%s", testBundleDir, setup.Location))
			if err != nil {
				log.Fatalf("Failed to load JSON in place of file ref %q: %v", setup.Location, err)
			}
			setup.Location = ""
			setup.Type = ""
			testObj.Setup = setup
		}
		for k := range testObj.Assertions {
			assertion := testObj.Assertions[k]
			for l := range assertion.Transforms {
				transform := assertion.Transforms[l]
				if transform.Location != "" {
					transform.Expression, err = readFileLocation(fmt.Sprintf("%s/%s", testBundleDir, transform.Location))
					if err != nil {
						log.Fatalf("Failed to load JSON in place of file ref %q: %v", transform.Location, err)
					}
					transform.Location = ""
					assertion.Transforms[l] = transform
				}
			}
			testObj.Assertions[k] = assertion
		}
		testObjects.Tests[i] = testObj
	}

	// testObjectsStr, err := json.MarshalIndent(testObjects, "", "  ")
	// if err != nil {
	// 	log.Fatalf("Failed to marshal testObjects into a JSON string: %v", err)
	// }
	// fmt.Printf("\nTest Objects file:- %s", testObjectsStr)

	// Send this payload to Test Runner and print the id returned by it
	var res SolutionTestResult
	err = api.JSONPut(getSolutionTestUrl(), testObjects, &res, nil)
	if err != nil {
		log.Fatalf("Solution Test request failed: %v", err)
	}
	output.PrintCmdStatus(cmd, fmt.Sprintf("Solution Test data sent to test-runner successfully. Test ID - %s", res.ID))
}

func isTestPackageRoot(path string) bool {
	testObjectsPath := fmt.Sprintf("%s/test-objects.json", path)
	testObjectsFile, err := os.Open(testObjectsPath)
	if err != nil {
		log.Errorf("The folder %s is not a solution test root folder", path)
		return false
	}
	testObjectsFile.Close()
	return true
}

func getTestObjects(path string) (*SolutionTestObjects, error) {
	testObjectsPath := fmt.Sprintf("%s/test-objects.json", path)
	testObjectsFile, err := os.Open(testObjectsPath)
	if err != nil {
		return nil, fmt.Errorf("%q is not a solution test root folder", path)
	}
	defer testObjectsFile.Close()

	testObjectBytes, err := io.ReadAll(testObjectsFile)
	if err != nil {
		return nil, err
	}

	var testObjects *SolutionTestObjects
	err = json.Unmarshal(testObjectBytes, &testObjects)
	if err != nil {
		return nil, err
	}

	return testObjects, nil
}

func readFileLocation(path string) (string, error) {
	jsonStr, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(jsonStr), nil
}

func getSolutionTestUrl() string {
	return "rest/kirby-solution-testing-poc/kirby-solution-testing-poc-function/solution/v1/test"
}
