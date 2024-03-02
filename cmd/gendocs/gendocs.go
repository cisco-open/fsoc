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
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/apex/log"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"

	"github.com/cisco-open/fsoc/config"
	"github.com/cisco-open/fsoc/output"
)

const TOCFileName = "pages.json"

const DocLinkUrl = "https://developer.cisco.com/docs/cisco-observability-platform"

// DocLinkReplaceRegexp is a regular expression to match absolute links to the platform documentation, capturing the topic as $1
var gblLinkReplaceRegexp = regexp.MustCompile(regexp.QuoteMeta(DocLinkUrl+`/#!`) + `(.*?)(\s|$|\.\s|\.$)`)

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
	Annotations:      map[string]string{config.AnnotationForConfigBypass: ""},
}

func NewSubCmd() *cobra.Command {
	gendocsCmd.Flags().
		Bool("h1", true, "adjust generated headings to start from h1 rather than h2")
	gendocsCmd.Flags().
		Bool("rel-links", true, "adjust generated platform doc links to become relative")
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
	output.PrintCmdStatus(cmd, "Generating documentation\n")
	err = doc.GenMarkdownTree(cmd.Parent(), path)
	if err != nil {
		log.Fatalf("Error generating fsoc docs: %v", err)
	}

	// generate table of contents
	output.PrintCmdStatus(cmd, "Generating table of contents\n")
	err = genTableOfContents(cmd, path, fs)
	if err != nil {
		log.Fatalf("Error generating fsoc docs table of contents: %v", err)
	}

	flagH1, _ := cmd.Flags().GetBool("h1")
	flagRelLinks, _ := cmd.Flags().GetBool("rel-links")
	if flagH1 || flagRelLinks {
		log.Infof("Processing files: h1=%v, rel-links=%v", flagH1, flagRelLinks)

		files := getListOfFiles(path)
		log.Infof("There are %d files to process", len(files))

		for i := 0; i < len(files); i++ {
			file := files[i]
			log.Infof("Processing file %q", file.Name())

			err := processFile(file, flagH1, flagRelLinks)
			if err != nil {
				log.Fatalf(err.Error())
			}
		}
	}

	output.PrintCmdStatus(cmd, "Documentation generated successfully.\n")
}

type tocEntry struct {
	Title   string     `json:"title,omitempty"`
	Content string     `json:"content,omitempty"`
	Items   []tocEntry `json:"items,omitempty"`
}

func genTableOfContents(cmd *cobra.Command, path string, fs *afero.Afero) error {
	// determine cobra root command
	root := cmd.Parent() // gendocs is a top-level command, so its parent is the root

	// generate TOC in memory
	toc := tocEntry{Items: []tocEntry{*genTOCNode(root)}}

	// write TOC to file (rw permissions & umask)
	tocPath := filepath.Join(path, TOCFileName)
	tocFile, err := fs.OpenFile(tocPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return fmt.Errorf("failed to open TOC file %v: %v", path, err)
	}
	if err = output.WriteJson(toc, tocFile); err != nil {
		return fmt.Errorf("failed to write TOC file %v: %v", path, err)
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

func getFileFromArgs(fileLoc string) *os.File {
	file, err := os.Open(fileLoc)
	if err != nil {
		log.Fatal(err.Error())
	}
	return file
}

func getListOfFiles(dir string) []*os.File {
	var files []*os.File
	err := filepath.Walk(dir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				log.Warnf("Error walking files at %q: %v", path, err)
				return err
			}
			file := getFileFromArgs(path)
			isFileMarkdown := strings.Contains(file.Name(), ".md")
			if isFileMarkdown {
				log.Infof("Adding %q to the list of files to process", file.Name())
				files = append(files, file)
			}
			return nil
		})
	if err != nil {
		log.Infof(err.Error())
	}
	return files
}

func processFile(file *os.File, modifyHeaderLevels bool, makeDocLinksRelative bool) error {
	fileScanner := bufio.NewScanner(file)
	fileScanner.Split(bufio.ScanLines)
	var fileLines []string

	for fileScanner.Scan() {
		fileLines = append(fileLines, fileScanner.Text())
	}
	for i := 0; i < len(fileLines); i++ {
		line := fileLines[i]
		if modifyHeaderLevels {
			if len(line) > 2 {
				if line[0:2] == "##" {
					fileLines[i] = line[1:]
				}
			}

			// the following two changes are not header level changes but related to meeting the same doc requirements
			if fileLines[i] == "## SEE ALSO" {
				fileLines[i] = "## See Also"
			}
			if fileLines[i] == "## Options inherited from parent commands" {
				fileLines[i] = "## Options Inherited From Parent Commands"
			}
		}
		if makeDocLinksRelative {
			oldLine := fileLines[i]

			// replace topic links, changing from an absolute URL to a markdown doc root-relative link
			fileLines[i] = gblLinkReplaceRegexp.ReplaceAllString(fileLines[i], `[$1](/#!$1)$2`)

			// replace any remaining absolute links
			fileLines[i] = strings.ReplaceAll(fileLines[i], DocLinkUrl, "[platform documentation](./)")

			// log change
			if fileLines[i] != oldLine {
				log.Infof("Changed link %d: %q -> %q", i, oldLine, fileLines[i])
			}
		}
	}

	if err := os.Truncate(file.Name(), 0); err != nil {
		log.Infof("Failed to truncate: %v", err)
	}

	newFile, _ := os.OpenFile(file.Name(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	dataWriter := bufio.NewWriter(newFile)

	for _, data := range fileLines {
		_, _ = dataWriter.WriteString(data + "\n")
	}

	dataWriter.Flush()

	return nil
}
