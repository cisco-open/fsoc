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

package solution

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/apex/log"
	"github.com/pkg/browser"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/cisco-open/fsoc/cmd/config"
)

const (
	callbackUrl  = "http://localhost:3101/solution"
	localUiUrl   = "http://localhost:3000/dev/authoring"
	templatePath = "/objects/template/"
)

type item struct {
	Name string `json:"name"`
}

type itemList struct {
	Items    []item `json:"items"`
	NumItems int    `json:"total"`
}

var authorCmd = &cobra.Command{
	Use:   "author [flags]",
	Short: "Open authoring tool for editing template files",
	Long:  "This command provides access to the FSO platform user interface authoring tool for editing a solution's templates interactively. It starts a background web server to provide access to the template files and opens the authoring tool in your default browser.",
	Run:   authorRunWrapper,
}

func getAuthorCmd() *cobra.Command {
	authorCmd.Flags().BoolP("local", "l", false, "Use locally-running UI service (developers only)")
	authorCmd.Flags().StringP("directory", "d", ".", "Directory where the solution files reside")

	return authorCmd
}

func authorRunWrapper(cmd *cobra.Command, args []string) {
	cfg := config.GetCurrentContext()
	local, _ := cmd.Flags().GetBool("local")
	dir, _ := cmd.Flags().GetString("directory")
	err := authorRun(cfg, dir, local)
	if err != nil {
		log.Fatal(err.Error())
	}
}

func authorRun(cfg *config.Context, dir string, local bool) error {
	// Check to make sure that the current directory is valid
	solutionDirectory, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	fileSystem := afero.NewBasePathFs(afero.NewOsFs(), solutionDirectory)
	solutionName, err := getSolutionName(fileSystem)
	if err != nil {
		return err
	}
	terminateServer := make(chan bool)
	// If the directory is valid start the callback server
	callbackServer, err := startCallbackServerForUIAuthor(fileSystem, terminateServer)
	if err != nil {
		return fmt.Errorf("Could not start a local http server for auth: %v", err.Error())
	}

	defer func() {
		err := callbackServer.Close()
		if err != nil {
			log.Errorf("Failed to automatically close the callback callbackServer: %v", err.Error())
		}
	}()

	// Construct URL to authoring tool
	var browserUrl *url.URL
	if local {
		// point to a locally running authoring tool
		browserUrl, err = url.Parse(localUiUrl)
		if err != nil {
			return fmt.Errorf("likely bug: could not parse local URL %q: %v", localUiUrl, err)
		}
	} else {
		// point to authoring tool in tenant's UI
		browserUrl = &url.URL{
			Scheme: "https",
			Host:   cfg.Server,
			Path:   "/ui/dev/authoring",
		}
	}
	queryParams := url.Values{
		"url":      {callbackUrl},
		"solution": {solutionName},
	}
	browserUrl.RawQuery = queryParams.Encode()

	// inform about required feature flag
	// TODO: remove when the feature flag is no longer needed
	log.Warnf("Note that this function requires UI feature flag ENABLE_FSOC_INTEGRATION_WITH_AUTHORING to be enabled")

	// Open Browser at the authoring tool
	if err = openBrowser(browserUrl.String(), terminateServer); err != nil {
		log.Errorf("Failed to automatically launch browser: %v", err.Error())
	}
	terminate := <-terminateServer
	fmt.Printf("Terminate callbackServer: %s", strconv.FormatBool(terminate))
	return nil
}

func startCallbackServerForUIAuthor(fileSystem afero.Fs, terminateServer chan bool) (*http.Server, error) {
	urlStruct, err := url.Parse(callbackUrl)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse callback URL %q: %v", callbackUrl, err)
	}

	server := &http.Server{
		Addr: urlStruct.Host,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callbackHandlerAuthorUI(terminateServer, w, r, fileSystem)
		}),
	}
	go func() {
		log.Infof("Starting the auth http server on %v", server.Addr)
		err := server.ListenAndServe()
		if err != nil && err.Error() != "http: Server closed" {
			log.Errorf("Failed to start auth http server on %v: %v", server.Addr, err)
		}
	}()
	return server, nil
}

/*
 * Reads the manifest file and extracts the solution name
 * fileContent should be the bytestream generated from the manifest file
 */
func getSolutionFromManifest(fileContent []byte) (string, error) {
	var manifest Manifest
	err := json.Unmarshal(fileContent, &manifest)
	if err != nil {
		log.Errorf("Failed to parse solution manifest: %v", err.Error())
		return "", err
	}
	return manifest.Name, nil
}

func callbackHandlerAuthorUI(terminateServer chan bool, w http.ResponseWriter, r *http.Request, fs afero.Fs) {
	// If terminate is set to true then the terminateServer channel will close and the server will terminate
	enableCors(&w)
	terminate := false
	//Check URI for malformations
	uri, err := checkURI(w, r)
	if err != nil {
		terminate = true
		log.Errorf("Error occurred processing request: %v", err)
	}

	// Log the incoming request
	log.WithFields(log.Fields{
		"Type": r.Method,
		"URI":  uri.RequestURI(),
	}).Info("Request Received")
	log.Info(uri.RequestURI())
	log.Info(r.Method)

	// Check the method of the incoming request
	switch r.Method {
	case "PUT":
		if uri.RequestURI() == "/solution/close" {
			terminate = true
			log.Info("Closing callback server")
			fmt.Fprint(w, "Successfully closed the server")
		} else if strings.HasPrefix(uri.RequestURI(), "/solution/templates") && len(uri.RequestURI()) > len("/solution/templates") {
			err = updateTemplate(w, r, fs, uri)
		} else {
			log.Errorf("This URI is not recognized, please check the path")
			fmt.Fprint(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		}
	case "GET":
		w.Header().Set("Content-Type", "application/json")
		if uri.RequestURI() == "/solution/templates" {
			err = returnTemplateList(w, r, fs)
		} else if uri.RequestURI() == "/ping" {
			fmt.Fprint(w, "Server is up at "+callbackUrl)
		} else if strings.HasPrefix(uri.RequestURI(), "/solution/templates") && len(uri.RequestURI()) > len("/solution/templates") {
			err = returnTemplate(w, r, fs, uri)
		} else {
			log.Errorf("This URI is not recognized, please check the path")
			fmt.Fprint(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		}
	case "DELETE":
		if strings.HasPrefix(uri.RequestURI(), "/solution/templates") && len(uri.RequestURI()) > len("/solution/templates") {
			err = deleteTemplateFile(w, r, fs, uri)
		} else {
			log.Errorf("This URI is not recognized, please check the Method")
			fmt.Fprint(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		}
	case "OPTION":
	default:
		log.Errorf("This method is not recognized, please check the path")
		fmt.Fprint(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
	}
	if err != nil {
		log.Errorf("Error while parsing request: %v", err.Error())
		fmt.Fprint(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
	}
	if terminate {
		terminateServer <- terminate
	}
}

func returnTemplate(w http.ResponseWriter, r *http.Request, fs afero.Fs, uri *url.URL) error {
	URIParts := strings.Split(uri.RequestURI(), "/")
	file, err := afero.ReadFile(fs, templatePath+URIParts[len(URIParts)-1])
	log.Info("Getting file at " + templatePath + URIParts[len(URIParts)-1])
	if err != nil {
		log.Errorf("No such template file in this directory: %v", err.Error())
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return err
	} else {
		fmt.Fprint(w, string(file))
		return nil
	}
}

func returnTemplateList(w http.ResponseWriter, r *http.Request, fs afero.Fs) error {
	files, err := afero.ReadDir(fs, templatePath)
	if err != nil {
		log.Errorf("Cannot read the files: %v", err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return err
	}
	var itemsList itemList
	itemArray := make([]item, 0)
	for i := 0; i < len(files); i++ {
		var item item
		item.Name = files[i].Name()
		itemArray = append(itemArray, item)
	}
	itemsList.Items = itemArray
	itemsList.NumItems = len(files)
	jsonOutput, err := json.Marshal(&itemsList)
	if err != nil {
		log.Error(err.Error())
		return err
	}
	fmt.Fprint(w, string(jsonOutput))
	return nil
}

func updateTemplate(w http.ResponseWriter, r *http.Request, fs afero.Fs, uri *url.URL) error {
	//var body string
	b, err := io.ReadAll(r.Body)
	if err != nil {
		log.Errorf("Cannot read request body: %v", err.Error())
	}
	URIParts := strings.Split(uri.RequestURI(), "/")
	err = afero.WriteFile(fs, templatePath+URIParts[len(URIParts)-1], b, 0644)
	if err != nil {
		log.Errorf("Error writing to this file: %v", err.Error())
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return err
	} else {
		fmt.Fprint(w, "Successfully wrote to "+templatePath+URIParts[len(URIParts)-1])
		return nil
	}
}

func deleteTemplateFile(w http.ResponseWriter, r *http.Request, fs afero.Fs, uri *url.URL) error {
	URIParts := strings.Split(uri.RequestURI(), "/")
	err := fs.Remove(templatePath + URIParts[len(URIParts)-1])
	if err != nil {
		log.Errorf("Error in deleting file: %v", err.Error())
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return err
	}
	fmt.Fprint(w, "File Successfully Deleted")
	return nil
}

func checkURI(w http.ResponseWriter, r *http.Request) (*url.URL, error) {
	_, err := url.Parse(callbackUrl)
	if err != nil {
		log.Errorf("Unexpected failure to obtain expected callback path (likely a bug): %v", err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return nil, err
	}
	uri, err := url.Parse(r.RequestURI)
	if err != nil {
		log.Errorf("Unexpected failure to parse callback path received (malformed request?): %v", err.Error())
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return nil, err
	}
	return uri, nil
}

func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Headers", "Content-Type, Access-Control-Allow-Headers, Authorization, X-Requested-With, appd-next-tenant-id")
	(*w).Header().Set("Access-Control-Allow-Methods", "GET, PUT, POST, OPTIONS, DELETE")
}

func openBrowser(url string, terminateServer chan bool) error {
	// redirect browser's package stdout to a pipe, saving the original stdout
	orig := browser.Stdout
	r, w, _ := os.Pipe()
	browser.Stdout = w
	defer func() {
		browser.Stdout = orig
	}()

	// start browser
	browserErr := browser.OpenURL(url) // check error later
	w.Close()                          // no more writing

	// copy the output in a separate goroutine so printing can't block indefinitely
	outChan := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, err := io.Copy(&buf, r)
		if err != nil {
			log.Warnf("Error capturing browser launch output: %v; ignoring it", err)
			// fall through
		}
		outChan <- buf.String()
	}()

	// collect and log any message displayed
	outMsg := strings.TrimSpace(<-outChan)
	if outMsg != "" {
		log.Infof("Browser launch: %v", outMsg)
	}

	return browserErr
}

func getSolutionName(fileSystem afero.Fs) (string, error) {
	fileContents, err := afero.ReadFile(fileSystem, "./manifest.json")
	if err != nil {
		log.Errorf("No manifest file in this directory: %v", err.Error())
		return "", err
	}
	exists, err := afero.DirExists(fileSystem, templatePath)
	if err != nil || !exists {
		log.Errorf("No "+templatePath+" directory: %v", err.Error())
		return "", err
	}
	solutionName, err := getSolutionFromManifest(fileContents)
	if err != nil {
		log.Errorf("Cannot get solution name for callback_url, check manifest file: %v", err.Error())
		return "", err
	}
	return solutionName, nil
}
