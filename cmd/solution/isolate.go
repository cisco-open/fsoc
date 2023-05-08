// Copyright 2023 Cisco Systems, Inc.
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

package solution

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/apex/log"
	jsonata "github.com/blues/jsonata-go"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/output"
)

const (
	regexPattern = `\${(.*?)\}`

	keyEnv           = "env"
	keyEnvTag        = "tag"
	keySys           = "sys"
	keySysSolutionID = "solutionId"
)

var jsonataFunctions = `
	$toSuffix := function($val) {
		$exists($val) and $val != "" and $val != 'null' and $val != "stable" ? $string($val) : ""
	};

	$isTagStable := function() {
		env.tag = "stable"
	};

    $dependencyTag := function($name) {
		$toSuffix($string($lookup(env.dependencyTags, $name)))
    };
`

var solutionIsolateCmd = &cobra.Command{
	Use:   "isolate --source-dir  [--target-dir  or --target-file] --env-file",
	Args:  cobra.MaximumNArgs(3),
	Short: "Creates a solution isolate the source dir to target dir by replacing expression in the artifacts with values in <env-file>",
	Long:  `This command creates a solution isolate from the <source-dir> folder to <target-dir> folder by replacing expressions in the solution artifacts with the values in the <env-file> file`,
	Example: `  
    fsoc solution --source-dir=mysolution --target-dir=mysolution-isolated --env-file=env.json
    fsoc solution --target-file=../mysolution-release.zip --tag=stable
    fsoc solution --target-dir=../mysolution-isolated --env-file=env.json 
	`,
	Run: solutionIsolateCommand,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) >= 1 {
			return nil, cobra.ShellCompDirectiveDefault
		} else {
			return getSolutionNames(toComplete), cobra.ShellCompDirectiveDefault
		}
	},
}

var rgxp *regexp.Regexp

func getsolutionIsolateCmd() *cobra.Command {
	log.SetLevel(log.DebugLevel)
	solutionIsolateCmd.Flags().String("source-dir", "", "path to the source folder")
	solutionIsolateCmd.Flags().String("target-dir", "", "path to the target folder")
	solutionIsolateCmd.Flags().String("target-file", "", "path to the target zip file")
	solutionIsolateCmd.Flags().String("tag", "", "tag for the solution")
	solutionIsolateCmd.Flags().String("env-file", "", "path to the env vars json file")
	return solutionIsolateCmd
}

func solutionIsolateCommand(cmd *cobra.Command, args []string) {
	srcFolder, _ := cmd.Flags().GetString("source-dir")
	targetFolder, _ := cmd.Flags().GetString("target-dir")
	targetFile, _ := cmd.Flags().GetString("target-file")
	tag, _ := cmd.Flags().GetString("tag")
	envVarsFile, _ := cmd.Flags().GetString("env-file")
	if srcFolder == "" {
		srcFolder = "./"
	}
	if tag == "" && envVarsFile == "" {
		envVarsFile = "./env.json"
	}
	if srcFolder == "" || (targetFolder == "" && targetFile == "") ||
		(envVarsFile == "" && tag == "") {
		log.Fatalf("<source-dir>, <target-dir>|<target-file> and <tag>|<env-file> cannot be empty")
	}
	if targetFolder != "" && targetFile != "" {
		log.Fatalf("cannot specify both <target-dir> and <target-file>")
	}
	if tag != "" && envVarsFile != "" {
		log.Fatalf("cannot specify both <tag> and <env-file>")
	}

	err := isolateSoluton(cmd, srcFolder, targetFolder, targetFile, tag, envVarsFile)
	if err != nil {
		log.Fatalf("Failed to isolate solution: %v", err)
	}

	message := fmt.Sprintf("Successfully created solution isolate from %s to %s .\r\n", srcFolder, targetFolder+targetFile)
	output.PrintCmdStatus(cmd, message)

}

var rgxPError error

func isolateSoluton(cmd *cobra.Command, srcFolder, targetFolder, targetFile, tag, envVarsFile string) error {
	rgxp, rgxPError = regexp.Compile(regexPattern)
	if rgxPError != nil {
		return fmt.Errorf("Error compiling regex expression %v", rgxPError)
	}
	currentDirectory, err := filepath.Abs(".")
	if err != nil {
		return fmt.Errorf("Error getting current directory: %v", currentDirectory)
	}
	srcPath, err := filepath.Abs(srcFolder)
	if err != nil {
		return fmt.Errorf("Error getting src folder %s , error - %v", srcPath, err)
	}

	envVars, err := loadEnvVars(cmd, tag, envVarsFile)
	if err != nil {
		return err
	}
	// parse env vars
	log.Infof("Parsed env vars %v", envVars)

	var targetPath string
	// if a file is specified generate files in a temp directory
	if targetFile != "" {
		zipFileName := filepath.Base(targetFile)
		dirName := zipFileName[:len(zipFileName)-len(filepath.Ext(zipFileName))]
		tempDir, err := os.MkdirTemp("", "temp")
		if err != nil {
			return fmt.Errorf("Error creating temporary folder %v", err)
		}
		tempDir = filepath.Join(tempDir, dirName)
		defer os.RemoveAll(tempDir)
		if targetPath, err = filepath.Abs(tempDir); err != nil {
			return fmt.Errorf("Error getting absolute folder path %v", err)
		}
		output.PrintCmdStatus(cmd, fmt.Sprintf("target path is %s\n", targetPath))
	} else {
		targetPath, err = filepath.Abs(targetFolder)
		if err != nil {
			return fmt.Errorf("Error getting target folder %v", err)
		}
	}

	if err = prepareForIsolation(srcPath, targetPath, targetFile, envVars); err != nil {
		return err
	}

	mf, err := isolateManifest(cmd, srcPath, targetPath, envVars)
	if err != nil {
		return err
	}
	// merge some values from manifest into env vars
	envVars = addSysVariables(mf, envVars)

	// isolate file
	if err = isolateFiles(mf, srcPath, targetPath, targetFile, envVars); err != nil {
		return err
	}

	// create zip if requested
	if targetFile != "" {
		zipFile := generateZip(cmd, targetPath)
		err = os.Rename(zipFile.Name(), targetFile)
	}

	return err
}

func prepareForIsolation(srcPath, targetPath, targetFile string, envVars interface{}) error {
	srcFs := afero.NewBasePathFs(afero.NewOsFs(), srcPath)
	if exists, _ := afero.DirExists(srcFs, "."); !exists {
		return fmt.Errorf("Src folder %s does not exist", srcPath)
	}
	targetFs := afero.NewBasePathFs(afero.NewOsFs(), targetPath)
	if exists, _ := afero.DirExists(targetFs, "."); exists {
		if empty, _ := afero.IsEmpty(targetFs, "."); !empty {
			return fmt.Errorf("Target folder %s is not empty", targetPath)
		}
	}

	if targetFile != "" {
		targetFileDir := filepath.Dir(targetFile)
		targetFileFs := afero.NewBasePathFs(afero.NewOsFs(), targetFileDir)
		if exists, _ := afero.DirExists(targetFileFs, "."); !exists {
			return fmt.Errorf("Target file path %s does not exist", targetFileDir)
		}
		if !strings.HasSuffix(targetFile, ".zip") {
			return fmt.Errorf("Target file must be .zip")
		}
	}

	err := targetFs.MkdirAll("", 0777)
	if err != nil {
		return fmt.Errorf("Failed to create target folder %v", err)
	}

	return nil
}

func isolateManifest(cmd *cobra.Command, srcPath, targetPath string, envVars interface{}) (*Manifest, error) {
	srcFs := afero.NewBasePathFs(afero.NewOsFs(), srcPath)
	fileName := "./manifest.json"
	manifestFile, err := afero.ReadFile(srcFs, fileName)
	if err != nil {
		log.Fatalf("Error opening manifest file: %v", err)
	}
	// evaluate jsonata in manifest
	manifestFile, err = evaluateJSONata(manifestFile, envVars, fileName)
	if err != nil {
		log.Fatalf("Error evaluating expressions in manifest: %v", err)
	}

	var manifest Manifest
	err = json.Unmarshal(manifestFile, &manifest)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse solution manifest: %v", err)
	}

	targetFs := afero.NewBasePathFs(afero.NewOsFs(), targetPath)
	f, err := targetFs.OpenFile("./manifest.json", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return nil, fmt.Errorf("Can't open manifest file: %v", err)
	}
	defer f.Close()
	err = output.WriteJson(manifest, f)
	return &manifest, err
}

func isolateFiles(mf *Manifest, srcPath, targetPath, targetFile string, envVars interface{}) error {
	var err error
	// traverse objects
	for _, objDef := range mf.Objects {
		if objDef.ObjectsFile != "" {
			err = evalAndCopyFile(objDef.ObjectsFile, srcPath, targetPath, envVars)
		} else {
			dirPath := filepath.Join(srcPath, objDef.ObjectsDir)
			err = traverseSolutionFolder(dirPath, mf, srcPath, targetPath, envVars)
		}
		if err != nil {
			return err
		}
	}
	// traverse types
	for _, typeFile := range mf.Types {
		if err = evalAndCopyFile(typeFile, srcPath, targetPath, envVars); err != nil {
			return err
		}
	}
	return nil
}

func traverseSolutionFolder(dirPath string, mf *Manifest, srcPath, targetPath string, envVars interface{}) error {
	log.Debugf("Traversing folder %s !", dirPath)
	err := filepath.Walk(dirPath,
		func(path string, info os.FileInfo, err error) error {
			// log.Infof("subfolder %v, err: %v", info, err)
			if !info.IsDir() {
				filePath := strings.Replace(path, srcPath, "", 1)
				return evalAndCopyFile(filePath, srcPath, targetPath, envVars)
			}
			return nil
		},
	)
	return err
}

func loadEnvVars(cmd *cobra.Command, tag, envVarsFile string) (interface{}, error) {
	if tag != "" {
		envVars := map[string]interface{}{
			"env": map[string]interface{}{
				"tag": tag,
			},
		}
		return envVars, nil
	}
	output.PrintCmdStatus(cmd, fmt.Sprintf("Loading env vars from %s \n", envVarsFile))
	absPath, err := filepath.Abs(envVarsFile)
	if err != nil {
		return nil, fmt.Errorf("Error getting envvars absolute path %s , error - %v", absPath, err)
	}
	fs := afero.NewBasePathFs(afero.NewOsFs(), absPath)
	envVarsContent, err := afero.ReadFile(fs, ".")
	if err != nil {
		return nil, fmt.Errorf("error reading env vars file %v", err)
	}
	var envVars interface{}
	err = json.Unmarshal(envVarsContent, &envVars)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse env vars file: %v", err)
	}
	return envVars, nil
}

func evalAndCopyFile(fileName, srcPath, targetPath string, envVars interface{}) error {
	log.Debugf("copying file %s from %v to %v", fileName, srcPath, targetPath)
	srcFs := afero.NewBasePathFs(afero.NewOsFs(), srcPath)
	targetFs := afero.NewBasePathFs(afero.NewOsFs(), targetPath)
	targetDirPath := filepath.Dir(fileName)
	if err := targetFs.MkdirAll(targetDirPath, 0777); err != nil {
		return err
	}
	in, err := afero.ReadFile(srcFs, fileName)
	if err != nil {
		return err
	}
	out, err := evaluateJSONata(in, envVars, fileName)
	if err != nil {
		return err
	}
	err = afero.WriteFile(targetFs, fileName, out, 0777)
	log.Debugf("writing file %s, %v", fileName, err)
	return err
}

func evaluateJSONata(in []byte, envVars interface{}, fileName string) ([]byte, error) {
	matches := rgxp.FindAllStringSubmatch(string(in), -1)
	out := in
	for _, m := range matches {
		e := m[1]
		expr := createJSONata(e)
		if !validExpression(e) {
			continue
		}
		// compile jsonata expr
		jexpr, err := jsonata.Compile(expr)
		if err != nil {
			return nil, fmt.Errorf("Error compiling expression %s in %s , error: %s", e, fileName, err)
		}
		// evalute expr
		res, err := jexpr.Eval(envVars)
		if err != nil {
			return nil, fmt.Errorf("Error evaluating expr %s in %s, error: %v", m[0], fileName, err)
		}
		out = bytes.ReplaceAll(out, []byte(m[0]), []byte(fmt.Sprintf("%s", res)))
	}
	return out, nil
}

func createJSONata(expr string) string {
	return fmt.Sprintf("( %s \n %s)", jsonataFunctions, expr)
}

func addSysVariables(mf *Manifest, envVars interface{}) interface{} {
	// convert to map
	m := envVars.(map[string]interface{})
	solutionID := mf.Name
	m[keySys] = map[string]interface{}{
		keySysSolutionID: solutionID,
	}
	return envVars
}

func validExpression(expr string) bool {
	return !strings.HasPrefix(strings.TrimSpace(expr), ".")
}
