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
	"fmt"

	"github.com/apex/log"
	"github.com/spf13/cobra"
)

// getSolutionNameFromArgs gets the solution name from the command line, either from
// the first positional argument or from a flag (deprecated but kepts for backward compatibility).
// The flagName is optional (use "" to omit).
// Prints error message and terminates if the name is missing/empty
func getSolutionNameFromArgs(cmd *cobra.Command, args []string, flagName string, tag string) string {
	// get solution name from a flag, if provided (deprecated but kept for backward compatibility)
	var nameFromFlag string
	if flagName != "" {
		var err error
		nameFromFlag, err = cmd.Flags().GetString(flagName)
		if err != nil {
			log.Fatalf("Error parsing flag %q: %v", flagName, err)
		}
	}

	// get solution name from the first positional argument and
	// return it (or fail if flag was provided as well)
	var name string
	if len(args) > 0 {
		name = args[0]
	}
	if name != "" {
		if nameFromFlag != "" {
			log.Fatal("Solution name must be specified either as a positional argument or with a flag but not both")
		}
		if tag != "" {
			name = fmt.Sprintf("%s.%s", name, tag)
		}
		return name
	}

	// return the solution name from flag, if provided
	if nameFromFlag != "" {
		return nameFromFlag
	}

	// fail
	log.Fatal("A non-empty <solution-name> argument is required.")
	return "" // unreachable
}
