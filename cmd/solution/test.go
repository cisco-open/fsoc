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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/output"
	"github.com/cisco-open/fsoc/platform/api"
)

var solutionTestCmd = &cobra.Command{
	Use:              "test",
	Args:             cobra.ExactArgs(0),
	Short:            "Test Solution",
	Long:             "This command allows the current tenant specified in the profile to run tests against an already deployed solution",
	Example:          `  fsoc solution test`,
	Run:              testSolution,
	TraverseChildren: true,
}

var solutionTestStatusCmd = &cobra.Command{
	Use:              "test-status",
	Args:             cobra.ExactArgs(0),
	Short:            "Status of Solution Test",
	Long:             "This command allows the current tenant specified in the profile to check the status of a test-run already initiated via the `fsoc solution test` command",
	Example:          ` fsoc solution test-status`,
	Run:              testSolutionStatus,
	TraverseChildren: true,
}

func getSolutionTestCmd() *cobra.Command {
	solutionTestCmd.Flags().String("test-bundle", "", "The fully qualified path name for the test bundle directory. If no value is provided, it will default to 'current' - meaning current directory, where this command is running.")
	solutionTestCmd.Flags().String("initial-delay", "", "Time duration (in seconds) that the Test Runner should wait before making first call to UQL")
	solutionTestCmd.Flags().String("max-retry-count", "", "Maximum Number of times the Test Runner should call UQL to get latest data. Depending on the error code returned by UQL, retry will be initiated.")
	solutionTestCmd.Flags().String("retry-delay", "", "Time duration (in seconds) that the Test Runner should wait between retries")
	return solutionTestCmd
}

func getSolutionTestStatusCmd() *cobra.Command {
	solutionTestStatusCmd.Flags().String("test-run-id", "", "The test-run-id returned by `fsoc solution test` command")
	return solutionTestStatusCmd
}

// Implementation for `fsoc solution test` command.
// This command takes 1 mandatory argument, called `test-bundle`, which is a path to a directory where the files necessary to run the solution test are present.
// If no `test-bundle` path is provided, the command will use current directoy as `test-bundle` path.
// The command looks for a file called `test-objects.json` in the `test-bundle` directory. This file should contain payload that will be sent to the test-runner server-side component to run the solution test.
// All the file references present in `test-objects.json` should be relative paths to other files inside `test-bundle` path.
// The command will try to read those files and replace file refereces in `test-objects.json` with actual file contents.
// Once all this parsing is done, the command will prepare the payload for test-runner; Make http call to it and print the `test-run-id“ string that it gets from the test-runner.
// The test-run-id returned by this command should be used to check status of the test using `fsoc solution test-status` command.
func testSolution(cmd *cobra.Command, args []string) {
	var testBundleDir string
	testBundlePath, _ := cmd.Flags().GetString("test-bundle")
	initialDelay, _ := cmd.Flags().GetString("initial-delay")
	maxRetryCount, _ := cmd.Flags().GetString("max-retry-count")
	retryDelay, _ := cmd.Flags().GetString("retry-delay")

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

	// Read Test Objects JSON
	if !isTestPackageRoot(testBundleDir) {
		log.Fatalf("No test-objects file found in %q; please run this command in a directory with a test-objects file or use the --test-bundle flag", testBundleDir)
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
			inputBytes, err := readFileLocation(fmt.Sprintf("%s/%s", testBundleDir, setup.Location))
			if err != nil {
				log.Fatalf("Failed to read file ref %q: %v", setup.Location, err)
			}
			inputCompactBytes := new(bytes.Buffer)
			err = json.Compact(inputCompactBytes, inputBytes)
			if err != nil {
				log.Fatalf("JSON compact operation failed: %v", err)
			}
			var inputData interface{}
			err = json.Unmarshal(inputCompactBytes.Bytes(), &inputData)
			if err != nil {
				log.Fatalf("JSON Unmarshal failed: %v", err)
			}
			setup.Input = inputData
			testObj.Setup = setup
		}
		for k := range testObj.Assertions {
			assertion := testObj.Assertions[k]
			for l := range assertion.Transforms {
				transform := assertion.Transforms[l]
				if transform.Location != "" {
					transformBytes, err := readFileLocation(fmt.Sprintf("%s/%s", testBundleDir, transform.Location))
					if err != nil {
						log.Fatalf("Failed to load JSON in place of file ref %q: %v", transform.Location, err)
					}
					transformStr := string(transformBytes)
					transformStr = sanitizeString(transformStr)
					transform.Expression = transformStr
					assertion.Transforms[l] = transform
				}
			}
			testObj.Assertions[k] = assertion
		}
		testObjects.Tests[i] = testObj
	}

	// Set initial-delay, max-retry-count, retry-delay
	if initialDelay != "" {
		testObjectsInt, err := strconv.Atoi(initialDelay)
		if err != nil {
			log.Fatalf("Error while reading integer value from string %s: %v", initialDelay, err)
		}
		testObjects.InitialDelay = testObjectsInt
	}
	if maxRetryCount != "" {
		maxRetryCountInt, err := strconv.Atoi(maxRetryCount)
		if err != nil {
			log.Fatalf("Error while reading integer value from string %s: %v", maxRetryCount, err)
		}
		testObjects.MaxRetryCount = maxRetryCountInt
	}
	if retryDelay != "" {
		retryDelayInt, err := strconv.Atoi(retryDelay)
		if err != nil {
			log.Fatalf("Error while reading integer value from string %s: %v", retryDelay, err)
		}
		testObjects.RetryDelay = retryDelayInt
	}

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
// This command takes 1 mandatory argument, called `test-run-id` which is a string that represents a solution test-run already initiated via `fsoc solution test` command.
// It is therefore recommended that `fsoc solution test` command is run before this command, and the `test-run-id` returned from it is used here.
// The command will read the supplied test-run-id; Call test-runner server-side component, that runs the solution tests; Get the latest status of the test-run and print it in a user-friendly notation.
func testSolutionStatus(cmd *cobra.Command, args []string) {
	// Read the test-run-id
	suppliedTestId, _ := cmd.Flags().GetString("test-run-id")
	if suppliedTestId == "" {
		log.Fatal("Supplied test-run-id is null or empty.")
	}

	// Send the test-run-id to the test-runner and print the response
	var res SolutionTestStatusResult
	err := api.JSONGet(getSolutionTestStatusUrl(suppliedTestId), &res, nil)
	if err != nil {
		log.Fatalf("Solution Test Status request failed: %v", err)
	}

	// Print the result in JSON format
	for i := range res.StatusMessages {
		statusMessage := res.StatusMessages[i].Message
		if strings.Contains(statusMessage, "\n") {
			res.StatusMessages[i].Statuses = strings.Split(statusMessage, "\n")
			res.StatusMessages[i].Message = ""
		}
	}
	resJsonBytes, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		log.Fatalf("JSON marshal failed: %v", err)
	}
	output.PrintCmdStatus(cmd, fmt.Sprintf("Solution Test Status received for test-run-id (%s): \n%v", suppliedTestId, string(resJsonBytes)))
}

func isTestPackageRoot(path string) bool {
	testObjectsPath := fmt.Sprintf("%s/test-objects.json", path)
	testObjectsFile, err := os.Open(testObjectsPath)
	if err != nil {
		log.Errorf("The directory %s is not a solution test root directory", path)
		return false
	}
	testObjectsFile.Close()
	return true
}

func getTestObjects(path string) (*SolutionTestObjects, error) {
	testObjectsPath := fmt.Sprintf("%s/test-objects.json", path)
	testObjectsFile, err := os.Open(testObjectsPath)
	if err != nil {
		return nil, fmt.Errorf("%q is not a solution test root directory", path)
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

func readFileLocation(path string) ([]byte, error) {
	jsonBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return jsonBytes, nil
}

func getSolutionTestUrl() string {
	return fmt.Sprintf("%s/solution/v1/test", getBaseUrl())
}

func getSolutionTestStatusUrl(testId string) string {
	return fmt.Sprintf("%s/solution/v1/status/%s", getBaseUrl(), testId)
}

func getBaseUrl() string {
	return "rest/fsomon-test-runner-solution/test-runner"
}

func sanitizeString(input string) string {
	input = strings.ReplaceAll(input, "\n", "")
	input = strings.ReplaceAll(input, " ", "")
	return input
}
