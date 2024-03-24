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

package proxy

import (
	"os"

	"github.com/apex/log"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/output"
	"github.com/cisco-open/fsoc/platform/api"
)

// proxyCmd represents the login command
var proxyCmd = &cobra.Command{
	Use:   "proxy",
	Short: "Proxy local http requests to platform",
	Long: `This command runs a proxy server to forward http requests
to the platform API. It will automatically login and provide the necessary authentication.

The command can be used in two modes:
1. Run the proxy server in foreground until terminated with Ctrl-C.
2. Start the proxy server, execute a command (e.g., curl or shell script) and terminate.

When running a command, fsoc will exit with the exit code of the command.
`,
	Example: `  fsoc proxy -p 8000
  fsoc proxy -c "curl -fsSL http://localhost:8080/knowledge-store/v1/objects/extensibility:solution/k8sprofiler" -q
  fsoc proxy -p 8000 -c "mytest.sh 8000"`,
	Run: proxy,
}

func NewSubCmd() *cobra.Command {
	proxyCmd.Flags().IntP("port", "p", 8080, "Port to listen on")
	proxyCmd.Flags().StringP("command", "c", "", "Command to run after starting the proxy server")
	proxyCmd.Flags().BoolP("quiet", "q", false, "Suppress all fsoc status output to stdout")
	return proxyCmd
}

func proxy(cmd *cobra.Command, args []string) {
	// setup status printer, suppressing output if quiet flag is set
	quiet, _ := cmd.Flags().GetBool("quiet")
	statusPrinter := func(s string) {
		log.Info(s)
		if !quiet {
			output.PrintCmdStatus(cmd, s+"\n")
		}
	}

	// ensure profile is logged in before we start the proxy
	if err := api.Login(); err != nil {
		log.Fatalf("Login failed: %v", err)
	}
	statusPrinter("Login completed successfully")

	// run the proxy server
	port, _ := cmd.Flags().GetInt("port")
	command, _ := cmd.Flags().GetString("command")
	var exitCode int
	if err := api.RunProxyServer(port, command, statusPrinter, &exitCode); err != nil {
		log.Fatalf("Proxy server failed: %v", err)
	}

	// pass exit code back to caller (if we executed a command)
	if command != "" && exitCode != 0 {
		os.Exit(exitCode)
	}
}
