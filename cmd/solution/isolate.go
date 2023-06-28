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

	"github.com/cisco-open/fsoc/cmd/config"
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
	
$dependency := function($name) {
	$name & $toSuffix($string($lookup(env.dependencyTags, $name)))
	};
`

var rgxp *regexp.Regexp // TODO: consider removing the global var

var solutionIsolateCmd = &cobra.Command{
	Use:    "isolate [--source-dir=<solution-dir>]  (--target-dir=<target-dir>|--target-file=<target-file>) [--env-file=<env-file>|--tag=<tag>]",
	Hidden: true,
	Args:   cobra.NoArgs,
	Short:  "Creates an isolated solution copy using values in an environment file",
	Long: `EXPERIMENTAL! 
This command creates an isolated solution using a tag specified in the env file (or --tag flag), starting from a solution directory with templated manifest and objects.
The command produces a solution whose name has the tag as a suffix (e.g., solution "spacefleet" with tag "dev" becomes "spacefleetdev"), allowing multiple copies of the same solution with different tags to coexist in the same environment. The env file may contain additional variables, including tags to use for dependent solutions.
Note that when the --tag flag is used, the env.json file is ignored and no dependency tags are defined (this is intended to work best with --tag=stable).

The command takes the solution from current directory (or --source-dir) and produces the isolated version either in a directory (if --target-dir is specified) or in a solution zip file (if --target-file is specified).

Note that this command is experimental and will likely be removed. Please use "push", "validate" or "package" directly.

See documentation for manifest syntax and examples (link to be added here).
`,
	Example: `  
  fsoc solution isolate --target-dir=../mysolution-joe  # tags come from the current directory's private copy of ./env.json
  fsoc solution isolate --target-file=../mysolution-release.zip --tag=stable
  fsoc solution isolate --source-dir=mysolution --target-dir=mysolution-staging --env-file=staging-env.json
	`,
	Run:         solutionIsolateCommand,
	Annotations: map[string]string{config.AnnotationForConfigBypass: ""},
	Deprecated:  "please use `push`, `validate` or `package` instead",
}

func getsolutionIsolateCmd() *cobra.Command {
	c := solutionIsolateCmd
	c.Flags().String("source-dir", ".", "path to the source directory")
	c.Flags().String("target-dir", "", "path to the target directory")
	c.Flags().String("target-file", "", "path to the target zip file")
	c.Flags().String("tag", "", "tag for the solution")
	c.Flags().String("env-file", "./env.json", "path to the env vars json file")

	c.MarkFlagsMutuallyExclusive("tag", "env-file")
	c.MarkFlagsMutuallyExclusive("target-file", "target-dir")
	return c
}

func solutionIsolateCommand(cmd *cobra.Command, args []string) {
	srcFolder, _ := cmd.Flags().GetString("source-dir")
	targetFolder, _ := cmd.Flags().GetString("target-dir")
	targetFile, _ := cmd.Flags().GetString("target-file")
	tag, _ := cmd.Flags().GetString("tag")
	envVarsFile, _ := cmd.Flags().GetString("env-file")
	if tag != "" {
		envVarsFile = "" // remove default when tag is specified
	}
	if targetFolder == "" && targetFile == "" {
		_ = cmd.Usage()
		log.Fatal("Either <target-dir> or <target-file> must be specified.")
	}

	solutionName, _, err := isolateSolution(cmd, srcFolder, targetFolder, targetFile, tag, envVarsFile)
	if err != nil {
		log.Fatalf("Failed to isolate solution: %v", err)
	}

	message := fmt.Sprintf("Successfully created isolated solution %s from %s to %s\n", solutionName, srcFolder, targetFolder+targetFile)
	output.PrintCmdStatus(cmd, message)
}

// isolateSolution returns path to directory with isolated artifacts, the tag used and error
func isolateSolution(cmd *cobra.Command, srcFolder, targetFolder, targetFile, tag, envVarsFile string) (string, string, error) {
	var err error
	rgxp, err = regexp.Compile(regexPattern)
	if err != nil {
		return "", "", fmt.Errorf("(likely bug) Error compiling regex /%v/: %w", regexPattern, err)
	}
	srcPath, err := filepath.Abs(srcFolder)
	if err != nil {
		return "", "", fmt.Errorf("Error getting source directory %q: %w", srcFolder, err)
	}

	// parse env vars
	envVars, err := LoadEnvVars(cmd, tag, envVarsFile)
	if err != nil {
		return "", "", err
	}
	log.WithField("env_vars", envVars).Info("Parsed env vars")

	var targetPath string
	// use temp directory if a target is specified as a file
	if targetFile != "" {
		// create root directory for archiving, using the base name of the archive file
		// e.g., for /home/joe/spacefleet4.zip create /tmp/fsocXXXXX/spacefleet4/
		tempDirRoot, err := os.MkdirTemp("", "fsoc")
		if err != nil {
			return "", "", fmt.Errorf("Failed to create a temporary directory: %v", err)
		}
		defer os.RemoveAll(tempDirRoot)
		zipFileName := filepath.Base(targetFile)
		dirName := zipFileName[:len(zipFileName)-len(filepath.Ext(zipFileName))]
		tempDir := filepath.Join(tempDirRoot, dirName)
		if targetPath, err = filepath.Abs(tempDir); err != nil {
			return "", "", fmt.Errorf("Error getting absolute directory path %v", err)
		}
		output.PrintCmdStatus(cmd, fmt.Sprintf("Assembling solution in temp target directory %q\n", targetPath))
	} else {
		targetPath, err = filepath.Abs(targetFolder)
		if err != nil {
			return "", "", fmt.Errorf("Error getting target directory %v", err)
		}
	}

	if err = prepareForIsolation(srcPath, targetPath, targetFile, envVars); err != nil {
		return "", "", err
	}

	mf, err := isolateManifest(cmd, srcPath, targetPath, envVars)
	if err != nil {
		return "", "", err
	}

	// merge some values from manifest into env vars
	envVars = addSysVariables(mf, envVars)

	// isolate file
	if err = isolateFiles(mf, srcPath, targetPath, targetFile, envVars); err != nil {
		return "", "", err
	}

	// create zip if requested
	if targetFile != "" {
		zipFile := generateZip(cmd, targetPath, "")
		err = os.Rename(zipFile.Name(), targetFile)
		if err != nil {
			return "", "", fmt.Errorf("Failed to rename temp file %q to final file %q: %w", zipFile.Name(), targetFile, err)
		}
	}

	log.Info("Pseudo-isolation successfully completed")

	return mf.Name, GetTag(envVars), nil
}

func prepareForIsolation(srcPath, targetPath, targetFile string, envVars interface{}) error {
	srcFs := afero.NewBasePathFs(afero.NewOsFs(), srcPath)
	if exists, _ := afero.DirExists(srcFs, "."); !exists {
		return fmt.Errorf("Source directory %q does not exist", srcPath)
	}
	targetFs := afero.NewBasePathFs(afero.NewOsFs(), targetPath)
	if exists, _ := afero.DirExists(targetFs, "."); exists {
		if empty, _ := afero.IsEmpty(targetFs, "."); !empty {
			return fmt.Errorf("Target directory %q is not empty", targetPath)
		}
	}

	if targetFile != "" {
		targetFileDir := filepath.Dir(targetFile)
		targetFileFs := afero.NewBasePathFs(afero.NewOsFs(), targetFileDir)
		if exists, _ := afero.DirExists(targetFileFs, "."); !exists {
			return fmt.Errorf("Target file path %q does not exist", targetFileDir)
		}
		if !strings.HasSuffix(targetFile, ".zip") {
			return fmt.Errorf(`Target file %q must have a ".zip" extension`, targetFile)
		}
	}

	err := targetFs.MkdirAll("", 0777)
	if err != nil {
		return fmt.Errorf("Failed to create target directory %q: %v", targetPath, err)
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
	log.WithField("path", dirPath).Debug("Traversing directory")
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

func LoadEnvVars(cmd *cobra.Command, tag, envVarsFile string) (interface{}, error) {
	// create ad-hoc env vars if the tag flag is specified (instead of an env json file)
	if tag != "" {
		envVars := map[string]interface{}{
			"env": map[string]interface{}{
				"tag": tag,
			},
		}
		return envVars, nil
	}

	// parse the env vars file
	output.PrintCmdStatus(cmd, fmt.Sprintf("Reading env vars from %q\n", envVarsFile))
	absPath, err := filepath.Abs(envVarsFile)
	if err != nil {
		return nil, fmt.Errorf("Failed to get the absolute path for the env vars file %q: %w", envVarsFile, err)
	}
	fs := afero.NewBasePathFs(afero.NewOsFs(), absPath)
	envVarsContent, err := afero.ReadFile(fs, ".")
	if err != nil {
		return nil, fmt.Errorf("Failed to read the env vars file: %w", err)
	}
	var envVars interface{}
	err = json.Unmarshal(envVarsContent, &envVars)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse env vars file: %w", err)
	}
	return envVars, nil
}

func GetTag(envVars interface{}) string {
	if root, ok := envVars.(map[string]any); ok {
		if env, ok := root["env"].(map[string]any); ok {
			if tag, ok := env["tag"].(string); ok && tag != "" {
				return tag
			}
		}
	}
	log.Fatalf(`Failed to extract tag from env vars. Minmum required is --tag flag or {"env":{"tag":"<tagvalue>""}} env.json file`)
	return "" // can never get here, just keep compiler happy
}

func evalAndCopyFile(fileName, srcPath, targetPath string, envVars interface{}) error {
	log.Debugf("copying file %v from %v to %v", fileName, srcPath, targetPath)
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
