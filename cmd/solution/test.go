// Copyright 2023 Cisco Systems, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
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

var solutionTestStatusCmd = &cobra.Command{
	Use:              "test-status",
	Args:             cobra.ExactArgs(0),
	Short:            "Status of Solution Test",
	Long:             "This command allows the current tenant specified in the profile to check status of the solution-test already run by `fsoc solution test` command",
	Example:          ` fsoc solution test-status`,
	Run:              testSolutionStatus,
	TraverseChildren: true,
}

var testId string

func getSolutionTestCmd() *cobra.Command {
	solutionTestCmd.Flags().String("test-bundle", "", "The fully qualified path name for the test bundle directory. If no value is provided, it will default to 'current' - meaning current directory, where this command is running.")
	return solutionTestCmd
}

func getSolutionTestStatusCmd() *cobra.Command {
	solutionTestStatusCmd.Flags().String("test-id", "", "The test-id provided by `fsoc solution test` command. If no value is provided, it will default to 'current' test-id saved locally - it may or may not be present. So it is advised that test-id is always supplied.")
	return solutionTestStatusCmd
}

// Implementation for `fsoc solution test` command.
// This command takes 1 argument, called `test-bundle`, which is a path to a directory where the files necessary to run the solution test are present.
// If no `test-bundle` path is provided, the command will use current directoy as `test-bundle` path.
// The command looks for a file called `test-objects.json` in the `test-bundle` directory. This file should contain payload that will be sent to the test-runner server-side component to run the solution test.
// All the file references present in `test-objects.json` should be relative paths to other files inside `test-bundle` path.
// The command will try to read those files and replace file refereces in `test-objects.json` with actual file contents.
// Once all this parsing is done, the command will prepare the payload for test-runner; Make http call to it and print the `test-id“ string that it gets from the test-runner.
// The test-id returned by this command should be used to check status of the test using `fsoc solution test-status` command.
func testSolution(cmd *cobra.Command, args []string) {
	var testBundleDir string
	testBundlePath, _ := cmd.Flags().GetString("test-bundle")

	// Get Test Bundle Directory
	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatal(err.Error())
	}
	if testBundlePath == "" {
		fmt.Println("Supplied test-bundle path is empty. Using current directoty to look for test-bundle.")
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
	testId := res.ID
	output.PrintCmdStatus(cmd, fmt.Sprintf("Solution Test data sent to test-runner successfully. Test ID - %s", testId))
}

// Implementation for `fsoc solution test-status` command.
// This command takes 1 argument, called `test-id` which is a string that represents a solution test run by `fsoc solution test` command.
// If no `test-id` string is provided, then the command will try to use locally stored test-id from previously run test.
// It is therefore recommended that `fsoc solution test` command is run before this command, and the `test-id` returned from it is used here.
// The command will read the supplied test-id; Call test-runner server-side component, that runs the solution tests; Get the latest status of the test and print it in a user-readable notation.
func testSolutionStatus(cmd *cobra.Command, args []string) {
	// Read the test-id
	suppliedTestId, _ := cmd.Flags().GetString("test-id")
	if suppliedTestId == "" {
		fmt.Println("Supplied test-id is null or empty.")
		suppliedTestId = testId
		if suppliedTestId == "" {
			log.Fatalf("No local test-id saved as well. Exiting...")
		}
		fmt.Printf("Using locally saved test-id (%s) to check test status", suppliedTestId)
	} else {
		fmt.Printf("Using supplied test-id (%s) to check test status", suppliedTestId)
	}

	// Send the test-id to the test-runner and print the response
	var res SolutionTestStatusResult
	err := api.JSONGet(getSolutionTestStatusUrl(testId), &res, nil)
	if err != nil {
		log.Fatalf("Solution Test Status request failed: %v", err)
	}
	output.PrintCmdStatus(cmd, fmt.Sprintf("Solution Test Status received for test-id (%s): %s", testId, res.Status))
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
	return fmt.Sprintf("%s/solution/v1/test", getBaseUrl())
}

func getSolutionTestStatusUrl(testId string) string {
	return fmt.Sprintf("%s/solution/v1/status/%s", getBaseUrl(), testId)
}

func getBaseUrl() string {
	return "rest/kirby-solution-testing-poc/kirby-solution-testing-poc-function"
}
