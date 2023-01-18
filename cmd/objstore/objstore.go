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

package objstore

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewSubCmd() *cobra.Command {
	// objStoreCmd represents the objstore command
	objStoreCmd := &cobra.Command{
		Use:     "objstore",
		Short:   "Perform objectstore interactions.",
		Aliases: []string{"obj", "objs"},
		Long: `
---------------------------------------------------------------
        ___.         __           __                           
  ____  \_ |__      |__|  _______/  |_   ____  _______   ____  
 /  _ \  | __ \     |  | /  ___/\   __\ /  _ \ \_  __ \_/ __ \ 
(  <_> ) | \_\ \    |  | \___ \  |  |  (  <_> ) |  | \/\  ___/ 
 \____/  |___  //\__|  |/____  > |__|   \____/  |__|    \___  >
             \/ \______|     \/                             \/ 
----------------------------------------------------------------
Perform objectstore interactions.
See <docs url>`,
		Example: `# Get object type
  fsoc objstore get-type --type=<typeName> --solution=<solutionName>"
# Get object
  fsoc obj get --type=<typeName> --object=<objectId> --layer-id=<layerId> --layer-type=SOLUTION|ACCOUNT|GLOBALUSER|TENANT|LOCALUSER
# Get object
  fsoc obj create --type=<fully-qualified-typename> --object-file=<fully-qualified-path> --layer-type=SOLUTION|ACCOUNT|GLOBALUSER|TENANT|LOCALUSER [--layer-id=<respective-layer-id>] `,
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("incomplete command")
		},
		TraverseChildren: true,
	}

	objStoreCmd.AddCommand(newGetObjectCmd())
	objStoreCmd.AddCommand(newGetTypeCmd())
	objStoreCmd.AddCommand(getCreateObjectCmd())
	objStoreCmd.AddCommand(getUpdateObjectCmd())
	objStoreCmd.AddCommand(getDeleteObjectCmd())

	return objStoreCmd
}
