package solution

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/apex/log"
	"github.com/spf13/afero"
	"github.com/spf13/afero/zipfs"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/output"
	"github.com/cisco-open/fsoc/platform/api"
)

var solutionForkCmd = &cobra.Command{
	Use:   "fork --name=<solutionName> --source-name=<sourceSolutionName>",
	Short: "Fork a solution in the specified folder",
	Long:  `This command will download the solution into this folder and change the name of the manifest to the specified name`,
	Run:   solutionForkCommand,
}

func GetSolutionForkCommand() *cobra.Command {
	solutionForkCmd.Flags().String("source-name", "", "name of the solution that needs to be downloaded")
	_ = solutionForkCmd.MarkFlagRequired("source-name")
	solutionForkCmd.Flags().String("name", "", "name of the solution to copy it to")
	_ = solutionForkCmd.MarkFlagRequired("name")
	return solutionForkCmd
}

func solutionForkCommand(cmd *cobra.Command, args []string) {
	solutionName, _ := cmd.Flags().GetString("source-name")
	forkName, _ := cmd.Flags().GetString("name")
	if solutionName == "" {
		log.Fatalf("name cannot be empty, use --source-name=<solution-name>")
	}

	currentDirectory, err := filepath.Abs(".")
	if err != nil {
		log.Fatalf("Error getting current directory: %v", currentDirectory)
	}

	fileSystemRoot := afero.NewBasePathFs(afero.NewOsFs(), currentDirectory)

	if solutionNameFolderInvalid(fileSystemRoot, forkName) {
		log.Fatalf(fmt.Sprintf("A non empty folder with the name %s already exists", forkName))
	}

	fileSystem := afero.NewBasePathFs(afero.NewOsFs(), currentDirectory+"/"+forkName)

	if manifestExists(fileSystem, forkName) {
		log.Fatalf("There is already a manifest file in this folder")
	}

	downloadSolutionZip(cmd, solutionName, forkName)
	err = extractZip(fileSystemRoot, fileSystem, solutionName)
	if err != nil {
		log.Fatalf("Failed to copy files from the zip file to current directory: %v", err)
	}

	editManifest(fileSystem, forkName)

	err = fileSystemRoot.Remove("./" + solutionName + ".zip")
	if err != nil {
		log.Fatalf("Failed to remove zip file in current directory: %v", err)
	}

	message := fmt.Sprintf("Successfully forked %s to current directory.\r\n", solutionName)
	output.PrintCmdStatus(cmd, message)

}

func solutionNameFolderInvalid(fileSystem afero.Fs, forkName string) bool {
	exists, _ := afero.DirExists(fileSystem, forkName)
	if exists {
		empty, _ := afero.IsEmpty(fileSystem, forkName)
		return !empty
	} else {
		err := fileSystem.Mkdir(forkName, os.ModeDir)
		if err != nil {
			log.Fatalf("Failed to create folder in this directory")
		}
		err = os.Chmod(forkName, 0700)
		if err != nil {
			log.Fatalf("Failed to set permission on folder")
		}
	}
	return false
}

func manifestExists(fileSystem afero.Fs, forkName string) bool {
	exists, err := afero.Exists(fileSystem, forkName+"/manifest.json")
	if err != nil {
		log.Fatalf("Failed to read filesystem for manifest: %v", err)
	}
	return exists
}

func extractZip(rootFileSystem afero.Fs, fileSystem afero.Fs, solutionName string) error {
	solutionZip := "./" + solutionName + ".zip"
	zipFile, err := rootFileSystem.OpenFile(solutionZip, os.O_RDONLY, os.FileMode(0644))
	if err != nil {
		log.Fatalf("Error opening zip file: %v", err)
	}
	fileInfo, _ := rootFileSystem.Stat(solutionZip)
	reader, _ := zip.NewReader(zipFile, fileInfo.Size())
	zipFileSystem := zipfs.New(reader)
	err = copyFolderToLocal(zipFileSystem, fileSystem, "./"+solutionName)
	return err
}

func editManifest(fileSystem afero.Fs, forkName string) {
	manifestFile, err := afero.ReadFile(fileSystem, "./manifest.json")
	if err != nil {
		log.Fatalf("Error opening manifest file")
	}

	var manifest Manifest
	err = json.Unmarshal(manifestFile, &manifest)
	if err != nil {
		log.Errorf("Failed to parse solution manifest: %v", err)
	}

	err = refactorSolution(fileSystem, &manifest, forkName)
	if err != nil {
		log.Errorf("Failed to refactor component definition files within the solution: %v", err)
	}
	manifest.Name = forkName

	b, _ := json.Marshal(manifest)

	err = afero.WriteFile(fileSystem, "./manifest.json", b, 0644)
	if err != nil {
		log.Errorf("Failed to to write to solution manifest: %v", err)
	}
}

func downloadSolutionZip(cmd *cobra.Command, solutionName string, forkName string) {
	var solutionNameWithZipExtension = getSolutionNameWithZip(solutionName)
	var message string

	headers := map[string]string{
		"stage":            "STABLE",
		"solutionFileName": solutionNameWithZipExtension,
	}
	httpOptions := api.Options{Headers: headers}
	bufRes := make([]byte, 0)
	if err := api.HTTPGet(getSolutionDownloadUrl(solutionName), &bufRes, &httpOptions); err != nil {
		log.Fatalf("Solution download failed: %v", err)
	}

	message = fmt.Sprintf("Solution bundle %s was successfully downloaded in the this directory.\r\n", solutionName)
	output.PrintCmdStatus(cmd, message)

	message = fmt.Sprintf("Changing solution name in manifest to %s.\r\n", forkName)
	output.PrintCmdStatus(cmd, message)
}

func copyFolderToLocal(zipFileSystem afero.Fs, localFileSystem afero.Fs, subDirectory string) error {
	dirInfo, err := afero.ReadDir(zipFileSystem, subDirectory)
	if err != nil {
		return err
	}
	for i := range dirInfo {
		zipLoc := subDirectory + "/" + dirInfo[i].Name()
		localLoc := convertZipLocToLocalLoc(subDirectory + "/" + dirInfo[i].Name())
		if !dirInfo[i].IsDir() {
			err = copyFile(zipFileSystem, localFileSystem, zipLoc, localLoc)
			if err != nil {
				return err
			}
		} else {
			err = localFileSystem.Mkdir(localLoc, os.ModeDir)
			if err != nil {
				return err
			}
			println(localLoc)
			err = localFileSystem.Chmod(localLoc, 0700)
			if err != nil {
				return err
			}
			err = copyFolderToLocal(zipFileSystem, localFileSystem, zipLoc)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func copyFile(zipFileSystem afero.Fs, localFileSystem afero.Fs, zipLoc string, localLoc string) error {
	data, err := afero.ReadFile(zipFileSystem, zipLoc)
	if err != nil {
		return err
	}
	_, err = localFileSystem.Create(localLoc)
	if err != nil {
		return err
	}
	err = afero.WriteFile(localFileSystem, localLoc, data, os.FileMode(os.O_RDWR))
	if err != nil {
		return err
	}
	return nil
}

func convertZipLocToLocalLoc(zipLoc string) string {
	secondSlashIndex := strings.Index(zipLoc[2:], "/")
	return zipLoc[secondSlashIndex+3:]
}

func refactorSolution(fileSystem afero.Fs, manifest *Manifest, forkName string) error {
	objDefs := manifest.Objects
	var err error
	for _, objDef := range objDefs {
		if objDef.ObjectsFile != "" {
			err = replaceStringInFile(fileSystem, objDef.ObjectsFile, manifest.Name, forkName)
		} else {
			wkDir, _ := os.Getwd()
			dirPath := fmt.Sprintf("%s/%s/%s", wkDir, forkName, objDef.ObjectsDir)
			err = filepath.Walk(dirPath,
				func(path string, info os.FileInfo, err error) error {
					// if err != nil {
					// 	return err
					// }
					if !info.IsDir() {
						removeStr := fmt.Sprintf("%s/%s/", wkDir, forkName)
						filePath := strings.ReplaceAll(path, removeStr, "")
						err = replaceStringInFile(fileSystem, filePath, manifest.Name, forkName)
					}
					return err
				})
		}
	}
	return err
}

func replaceStringInFile(fileSystem afero.Fs, filePath string, searchValue string, replaceValue string) error {
	data, err := afero.ReadFile(fileSystem, filePath)
	if err != nil {
		return err
	}
	newFileContent := string(data)
	newFileContent = strings.ReplaceAll(newFileContent, searchValue, replaceValue)
	err = afero.WriteFile(fileSystem, filePath, []byte(newFileContent), os.FileMode(os.O_RDWR))
	if err != nil {
		return err
	}
	return nil
}
