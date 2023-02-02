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

package version

import (
	//"errors"

	"github.com/apex/log"
	"github.com/cisco-open/fsoc/cmd/config"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:         "update",
	Short:       "Update fsoc",
	Long:        `Update fsoc if a new version is available.`,
	Run:         update,
	Annotations: map[string]string{config.AnnotationForConfigBypass: ""},
}

func init() {
	versionCmd.AddCommand(updateCmd)
}

func update(cmd *cobra.Command, args []string) {
	log.Fatalf("Update command is not implemented yet. Please check version manually and download an update if available.")
}

// func update(core *Core, pkg dependency.Installable) {
// 	core.cfg.Log.Info("")
// 	core.cfg.Log.ProgressReporter().SetProgress("checking updates")

// 	err := pkg.Check()
// 	if err == nil {
// 		core.cfg.Log.ProgressReporter().Stop()
// 		core.cfg.Log.Infof("cli is already up to date: %s", pkg.Version())

// 		return
// 	}

// 	if !errors.As(err, &dependency.OldVersionError{}) {
// 		core.cfg.Log.ProgressReporter().Stop()
// 		core.cfg.Log.Errorf("error occurred during check: %s", err.Error())

// 		return
// 	}

// 	core.cfg.Log.ProgressReporter().SetProgress("installing update")

// 	if err = pkg.Install(core.fs); err != nil {
// 		core.cfg.Log.ProgressReporter().Stop()
// 		core.cfg.Log.Errorf("error occurred during update: %s", err.Error())

// 		return
// 	}

// 	core.cfg.Log.ProgressReporter().Stop()
// 	core.cfg.Log.Infof("cli updated successfully to version: %s\n", pkg.Version())
// }
