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

// Package gendocs generates a command line reference for the fsoc utility
// using markdown. In addition, it generates a table of contents JSON file
// which matches the format expected by Cisco DevNet publishing hub (but can
// easily be used on its own)
package gendocs

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/apex/log"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"

	"github.com/cisco-open/fsoc/output"
)

const TOCFileName = "pages.json"

// gendocsCmd represents the gendocs command
var gendocsCmd = &cobra.Command{
	Use:   "gendocs PATH",
	Short: "Generate docs for fsoc",
	Long: `Generates markdown documentation for the fsoc command and their usage.
It will generate a series of files, one for each command; root and sub-root
commands include relative hyperlinks to individual commands. 
The directory should either be empty or not exist.`,
	Example:          `  fsoc gendocs /tmp/docs`,
	Args:             cobra.ExactArgs(1),
	Run:              genDocs,
	TraverseChildren: true,
}

func NewSubCmd() *cobra.Command {
	return gendocsCmd
}

func genDocs(cmd *cobra.Command, args []string) {
	// check path validity
	path := args[0]
	if path == "" {
		log.Fatal(`The path to target directory cannot be empty. Try "fsoc gendocs ./docs".`)
	}

	// ensure directory is empty (create if needed)
	fs := &afero.Afero{Fs: afero.NewOsFs()}
	isExisting, err := fs.Exists(path)
	if err != nil {
		log.Fatalf(`Invalid target path %q: %v. Correct it or try "fsoc gendocs ./docs".`, path, err)
	}
	if isExisting {
		// fail if the existing path is not a directory or it is not empty
		isDir, err := fs.IsDir(path)
		if err != nil {
			log.Fatalf(`Invalid target path %q: %v. Correct it or try "fsoc gendocs ./docs".`, path, err)
		}
		if !isDir {
			log.Fatalf(`Target path %q is not a directory. Try a different name, e.g., "./docs".`, path)
		}
		isEmpty, err := fs.IsEmpty(path)
		if err != nil {
			log.Fatalf(`Invalid target path %q: %v. Correct it or try "fsoc gendocs ./docs".`, path, err)
		}
		if !isEmpty {
			log.Fatalf(`Target directory %q is not empty. Either delete the files or use try a different name.`, path)
		}
	} else {
		// create directory, including intermediate paths
		err := fs.MkdirAll(path, 0755) // u=rwx,go=rx
		if err != nil {
			log.Fatalf("Failed to create target directory %q: %v", path, err)
		}
	}

	// generate full docs, assumes gendocs is a direct child of the root cmd
	output.PrintCmdStatus("Generating documentation\n")
	err = doc.GenMarkdownTree(cmd.Parent(), path)
	if err != nil {
		log.Fatalf("Error generating fsoc docs: %v", err)
	}

	// generate table of contents
	output.PrintCmdStatus("Generating table of contents\n")
	err = genTableOfContents(cmd.Parent(), path, fs)
	if err != nil {
		log.Fatalf("Error generating fsoc docs table of contents: %v", err)
	}

	output.PrintCmdStatus("Documentation generated successfully.\n")
}

type tocEntry struct {
	Title   string     `json:"title,omitempty"`
	Content string     `json:"content,omitempty"`
	Items   []tocEntry `json:"items,omitempty"`
}

func genTableOfContents(root *cobra.Command, path string, fs *afero.Afero) error {
	// generate TOC in memory
	toc := tocEntry{Items: []tocEntry{*genTOCNode(root)}}

	// marshal to JSON
	jsToc, err := json.MarshalIndent(toc, "", "  ")
	if err != nil {
		return fmt.Errorf("Failed to marshal TOC to JSON: %v", err)
	}

	// display TOC if verbose
	if verbose, _ := root.Flags().GetBool("verbose"); verbose {
		output.PrintCmdStatus(string(jsToc) + "\n")
	}

	// write TOC to file (rw permissions & umask)
	tocPath := filepath.Join(path, TOCFileName)
	if err = fs.WriteFile(tocPath, jsToc, 0666); err != nil {
		return fmt.Errorf("Failed to write TOC file %v: %v", path, err)
	}

	return nil
}

func genTOCNode(root *cobra.Command) *tocEntry {
	var entry tocEntry

	// form entry for the command
	entry.Title = root.Name()
	entry.Content = strings.ReplaceAll(root.CommandPath(), " ", "_") + ".md"
	entry.Items = make([]tocEntry, 0)

	// recursively add subcommands, if any
	for _, cmd := range root.Commands() {
		// skip deprecated, hidden and non-commands
		if !cmd.IsAvailableCommand() || cmd.IsAdditionalHelpTopicCommand() {
			continue
		}

		// generate sub-entry(ies)
		entry.Items = append(entry.Items, *genTOCNode(cmd))
	}

	return &entry
}
