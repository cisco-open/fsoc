# Adding A New Command Group

The intended way to extend fsoc is to add new command groups and commands. Aside from a few built-in command groups (like `help` and `config`), most fsoc top-level commands cover functions of a given FSO subsystem. Examples include `uql`, `optimize` and, coming soon, `obj` (Orion object store), `solution`, etc.

fsoc has specifically been designed to make it easy to add new command groups by new contributors, as we expect that each team will add and work on its own command group(s). There is high degree of decoupling between command groups; if everything goes according to plan, different teams will never have to modify the same file (so no merges and resolving conflicts on rebase).

To add a new command group (and its initial) command, take a look at the `cmd/iam-role.go` file and the `cmd/iam-role/` subdirectory. You can simply copy these, replace `iam-role` with your command group name and proceed.

Here is a summary of what you would need to do to add a fictitious command group `shed`, with two subcommands, `paint` and `report`. Note that command groups are usually nouns (and, for cooler project names, proper nouns) and commands are usually verbs. fsoc uses the `noun verb` order (instead of `verb noun`) in recognition of the fact that different groups may have vastly different commands.

To add the group, create a file `cmd/shed.go` with the following contents:

```go
// Copyright 2024 Cisco Systems, Inc.
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
 
package cmd
 
import (
         "github.com/cisco-open/fsoc/cmd/shed"
)
 
func init() {
        registerSubsystem(shed.NewSubCmd())
}
```

This file causes the shed package to be registered with the command-line processor.

Now, add the root command for this group, in `cmd/shed/shed.go`:

```go
// Copyright 2024 Cisco Systems, Inc.
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
 
package shed
 
import (
        "github.com/spf13/cobra"
)
 
// shedCmd represents the shed commands
var shedCmd = &cobra.Command{
        Use:   "shed",
        Short: "Bike shed control",
        Long: `Build and operate my bike shed in a particular city. `,
        Example: `  fsoc shed paint --color=blue --city=Berkeley`,
        // Add "Run:" field if you want to have the `fsoc shed` run something without a sub-command
        TraverseChildren: true,
}
 
func init() {
        // Here you will define your flags and configuration settings.
 
        // Cobra supports Persistent Flags which will work for this command
        // and all subcommands, e.g.:
        shedCmd.PersistentFlags().String("city", "", "Select the city where the shed resides")
 
        // Cobra supports local flags which will only run when this command
        // is called directly, e.g.:
        // shedCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
 
func NewSubCmd() *cobra.Command {
        return shedCmd
}
```

For more information on the command fields and options, see the [cobra docs](https://umarcor.github.io/cobra/) and the [cobra reference](https://pkg.go.dev/github.com/spf13/cobra). fsoc uses the excellent cobra package for command line parsing; however, it does not support cobra's command code generator, so commands need to be added manually.

# Adding Commands

Now, add the specific commands you want to support in the group. You can put these in one file or each command in a separate file (all in the same package/directory, `shed`):

```go
// Copyright 2024 Cisco Systems, Inc.
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
 
package shed
 
import (
        "fmt"
        "github.com/apex/log"
        "github.com/spf13/cobra"
 
        "github.com/cisco-open/fsoc/output"
        "github.com/cisco-open/fsoc/platform/api"
 )
 
// paintCmd represents the shed paint command
var paintCmd = &cobra.Command{
        Use:   "paint --color=COLOR --city=CITY",
        Short: "Paint the shed in a given color",
        Long: `Paint the shed in the specified city with the given color.`,
        Example:          `  fsoc shed paint --color=blue --city=Berkeley`,
        Args:             cobra.ExactArgs(0),
        Run:              paint,
        TraverseChildren: true,
}
 
// reportCmd represents the shed lock command
var reportCmd = &cobra.Command{
        Use:   "report --city=CITY",
        Short: "Report shed info",
        Long: `Report shed details, such as color, in a specified city`,
        Example:          `  fsoc shed report --city=Berkeley`,
        Args:             cobra.ExactArgs(0),
        Run :             report,
        TraverseChildren: true,
}
 
 
func init() {
        shedCmd.AddCommand(paintCmd)
        shedCmd.AddCommand(reportCmd)
 
        // Here you will define your flags and configuration settings.
 
        // Cobra supports Persistent Flags which will work for this command
        // and all subcommands, e.g.:
        // reportCmd.PersistentFlags().String("foo", "", "A help for foo")
 
        // Cobra supports local flags which will only run when this command
        // is called directly, e.g.:
        paintCmd.Flags().String("color", "", "paint color to use")
}
 
type requestStruct struct {
        City string `json:"city"`
        Color string `json:"color"`
}
 
func paint(cmd *cobra.Command, args []string) {
        city, _ := shedCmd.Flags().GetString("city")
        color, _ := paintCmd.Flags().GetString("color")
 
        log.WithFields(log.Fields{"command": cmd.Name(), "color": color, "city": city}).Info("Shed group command")
 
        // create JSON query payload
        request := requestStruct{City: city, Color: color}
 
        // call API to perform request
        err := api.JSONPost("/shed/v1/paint", &request, &response, nil)
        if err != nil {
                log.Fatalf("Request failed %v", err.Error())
        }
 
        // display output
        output.PrintCmdOutput(cmd, response)
}
 
func report(cmd *cobra.Command, args []string) {
        city, _ := shedCmd.Flags().GetString("city")
        // note: this command does not have a flag --color
 
        log.WithFields(log.Fields{"command": cmd.Name(), "city": city}).Info("Shed group command")
 
        // call API to perform request
        err := api.JSONGet(fmt.Sprintf("/shed/v1/city/%v", city), &response, nil)
        if err != nil {
                log.Fatalf("Request failed %v", err.Error())
        }
 
        // display output
        output.PrintCmdOutput(cmd, response)
}
```

# Modifying Global Flag Options

This is rarely needed but certain commands may want to impose limitations or extend a global flag (such as `--output` ).

To do this, there are 3 things that need to be done:

1. Modify the help text for the command that uses the flag differently
2. Modify the usage text for the command that uses the flag differently
3. Add parsing as needed (e.g., additional checks to reject unsupported output values)

For an example of how to do this, check out `cmd/uql/uql.go`'s init() function.

In short:

```go
func init() {
    uqlCmd.Flags().StringVarP(&outputFlag, "output", "o", "table", "overridden")
    uqlCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
        changeFlagUsage(cmd.Parent())
        cmd.Parent().HelpFunc()(cmd, args)
    })
    uqlCmd.SetUsageFunc(func(cmd *cobra.Command) error {
        changeFlagUsage(cmd.Parent())
        return cmd.Parent().UsageFunc()(cmd)
    })
}
 
func changeFlagUsage(cmd *cobra.Command) {
    cmd.Flags().VisitAll(func(flag *pflag.Flag) {
        if flag.Name == "output" {
            flag.Usage = "output format (auto, table)"
        }
    })
}
```

Note that the flag is changed only when the command (and its help/usage is being invoked).