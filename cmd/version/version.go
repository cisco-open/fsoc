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
	"fmt"
	"github.com/Masterminds/semver/v3"
	"github.com/apex/log"
	"net/http"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/cmd/config"
	"github.com/cisco-open/fsoc/output"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print fsoc version",
	Long:  `Print fsoc version`,
	Run: func(cmd *cobra.Command, args []string) {
		displayVersion(cmd)
	},
	Annotations: map[string]string{config.AnnotationForConfigBypass: ""},
}

func init() {
	versionCmd.PersistentFlags().StringP("output", "o", "human", "Output format (human*, json, yaml)")
	versionCmd.PersistentFlags().BoolP("detail", "d", false, "Show full version detail (incl. git info)")
}

func NewSubCmd() *cobra.Command {
	return versionCmd
}

func displayVersion(cmd *cobra.Command) {
	// determine whether we need short output
	outfmt, _ := cmd.Flags().GetString("output")
	detail, _ := cmd.Flags().GetBool("detail")
	if !detail && (outfmt == "" || outfmt == "human") {
		output.PrintCmdStatus(cmd, fmt.Sprintf("fsoc version %v\n", GetVersionShort()))
		return
	}

	// prepare human output (in case needed)
	titles := []string{}
	values := []string{}
	for _, fieldTuple := range GetVersionDetailsHuman() {
		titles = append(titles, fieldTuple[0])
		values = append(values, fieldTuple[1])
	}
	output.PrintCmdOutputCustom(cmd, version, &output.Table{
		Headers: titles,
		Lines:   [][]string{values},
		Detail:  true,
	})
}

func GetLatestVersion() (string, error) {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error { // no redirect
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Get("https://github.com/cisco-open/fsoc/releases/latest")
	if err != nil {
		return "", err
	}
	split := strings.Split(resp.Header.Get("Location"), "/")
	if len(split) < 1 {
		return "", fmt.Errorf("version request did not return a version")
	}
	return split[len(split)-1], nil
}

func CheckForUpdate() {
	log.Infof("Checking for newer version of FSOC")
	newestVersion, err := GetLatestVersion()
	log.Infof("Latest version available: %s", newestVersion)
	if err != nil {
		log.Fatalf(err.Error())
	}
	currentVersion := GetVersion()
	currentVersionSemVer := semver.New(
		uint64(currentVersion.VersionMajor),
		uint64(currentVersion.VersionMajor),
		uint64(currentVersion.VersionMajor),
		currentVersion.VersionMeta, "")
	newestVersionSemVar := semver.MustParse(newestVersion)
	newerVersionAvailable := currentVersionSemVer.Compare(newestVersionSemVar) == -1
	var debugFields = log.Fields{"newerVersionAvailable": newerVersionAvailable, "oldVersion": currentVersionSemVer.String(), "newVersion": newestVersionSemVar.String()}
	if newerVersionAvailable {
		log.WithFields(debugFields).Warnf("There is a newer version of FSOC available, please upgrade from version %s to version %s", currentVersionSemVer.String(), newestVersionSemVar.String())
	} else {
		log.WithFields(debugFields)
	}
}
