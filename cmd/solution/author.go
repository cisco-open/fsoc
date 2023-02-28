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
	"github.com/cisco-open/fsoc/output"
)

const (
	callbackUrl       = "http://localhost:3101/solution"
	localUiUrl        = "http://localhost:3000/dev/authoring"
	authoringToolPath = "/ui/dev/authoring"
	templateType      = "dashui:template"
	manifestFname     = "manifest.json"
)

// templateServer provides the http server of the solution's templates
type templateServer struct {
	Fs            afero.Fs
	Name          string
	TemplatePath  string
	TerminateChan chan bool
	server        *http.Server
}

type apiItem struct {
	Name string `json:"name"`
}

type apiItemList struct {
	Items    []apiItem `json:"items"`
	NumItems int       `json:"total"`
}

var authorCmd = &cobra.Command{
	Use:    "author [flags]",
	Short:  "Open authoring tool for editing template files",
	Long:   "This command provides access to the FSO platform user interface authoring tool for editing a solution's templates interactively. It starts a background web server to provide access to the template files and opens the authoring tool in your default browser. This command currently works only with localhost-based development environments.",
	Run:    authorRunWrapper,
	Hidden: true,
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
	err := authorRun(cmd, cfg, dir, local)
	if err != nil {
		log.Fatal(err.Error())
	}
}

func authorRun(cmd *cobra.Command, cfg *config.Context, dir string, local bool) error {
	// create a template server object & start serving
	tServer := newTemplateServer(dir)

	// start the callback server
	err := tServer.Start()
	if err != nil {
		return fmt.Errorf("Could not start a local http server for auth: %v", err)
	}
	defer func() {
		err := tServer.Stop()
		if err != nil {
			log.Errorf("Failed to automatically close the callback server: %v", err)
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
			Path:   authoringToolPath,
		}
	}
	queryParams := url.Values{
		"url":      {callbackUrl},
		"solution": {tServer.Name},
	}
	browserUrl.RawQuery = queryParams.Encode()

	// inform about availability and required feature flag
	// TODO: remove when support changes
	log.Warnf("Note that this command currently does not work outside of development environments.")
	log.Warnf("Note that this command requires the following UI feature flags to be enabled:\n" +
		"\t- SHOW_TEMPLATE_AUTHORING\n" +
		"\t- ENABLE_FSOC_INTEGRATION_WITH_AUTHORING")

	// Open Browser at the authoring tool
	if err = openBrowser(browserUrl.String()); err != nil {
		log.Errorf("Failed to automatically launch browser: %v. Please try opening a browser at %q", err, browserUrl.String())
		// fall through
	}

	// wait for termination
	terminate := <-tServer.TerminateChan
	output.PrintCmdStatus(cmd, fmt.Sprintf("Terminating callback server: %s\n", strconv.FormatBool(terminate)))
	return nil
}

func newTemplateServer(dir string) *templateServer {
	// check direcory and create a confined file system
	solutionDirectory, err := filepath.Abs(dir)
	if err != nil {
		log.Fatalf("Invalid solution directory path: %v", err)
	}
	fileSystem := afero.NewBasePathFs(afero.NewOsFs(), solutionDirectory)

	// read solution manifest
	fileContents, err := afero.ReadFile(fileSystem, manifestFname)
	if err != nil {
		log.Fatalf("No manifest file in this directory: %v. Use `solution init` or `solution fork` to create.", err)
	}
	var manifest Manifest
	err = json.Unmarshal(fileContents, &manifest)
	if err != nil {
		log.Fatalf("Failed to parse solution manifest: %v", err)
	}
	if manifest.Name == "" {
		log.Fatal("Solution name is not missing (or empty) in the manifest; a valid name is required")
	}

	// obtain template path (for now, fsoc supports only a single object dir)
	templatePath, err := getTemplatePath(&manifest)
	if err != nil {
		log.Fatalf("Failed to obtain template path: %v", err)
	}
	exists, err := afero.DirExists(fileSystem, templatePath)
	if err != nil || !exists {
		log.Fatalf("Could not find %q directory specified in the manifest: %v", templatePath, err)
	}

	// create a termination signal channel
	terminateServer := make(chan bool)

	return &templateServer{
		Fs:            fileSystem,
		Name:          manifest.Name,
		TemplatePath:  templatePath + "/",
		TerminateChan: terminateServer,
	}
}

func (t *templateServer) Start() error {
	urlStruct, err := url.Parse(callbackUrl)
	if err != nil {
		return fmt.Errorf("Failed to parse callback URL %q: %v", callbackUrl, err)
	}

	t.server = &http.Server{
		Addr: urlStruct.Host,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.callbackHandler(w, r)
		}),
	}
	go func() {
		log.Infof("Starting the auth http server on %v", t.server.Addr)
		err := t.server.ListenAndServe()
		if err != nil && err.Error() != "http: Server closed" {
			log.Errorf("Failed to start auth http server on %v: %v", t.server.Addr, err)
		}
	}()
	return nil
}

func (t *templateServer) Stop() error {
	if t.server != nil {
		err := t.server.Close()
		if err != nil {
			err = fmt.Errorf("failed to stop http server: %v", err)
			return err
		}
	}
	return nil
}

func (t *templateServer) callbackHandler(w http.ResponseWriter, r *http.Request) {
	// add CORS headers
	enableCors(&w)

	// if terminate is set to true then the terminateServer channel will close and the server will terminate
	terminate := false

	// log the incoming request
	log.WithFields(log.Fields{
		"Type": r.Method,
		"URI":  r.RequestURI,
	}).Info("Request Received")

	// check URI for malformations
	uri, err := checkURI(w, r)
	if err != nil {
		log.Errorf("Invalid request URI: %v", err)
		// nb: error written to w by checkURI
		return
	}

	// process request by method
	err = nil
	switch r.Method {
	case "PUT":
		if uri.RequestURI() == "/solution/close" {
			terminate = true
			log.Info("Closing callback server")
			fmt.Fprint(w, "Successfully closed the server")
		} else if strings.HasPrefix(uri.RequestURI(), "/solution/templates") && len(uri.RequestURI()) > len("/solution/templates") {
			err = t.updateTemplate(w, r, uri)
		} else {
			log.Errorf("This URI is not recognized, please check the path")
			fmt.Fprint(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		}
	case "GET":
		w.Header().Set("Content-Type", "application/json")
		if uri.RequestURI() == "/solution/templates" {
			err = t.returnTemplateList(w, r)
		} else if uri.RequestURI() == "/ping" {
			fmt.Fprintf(w, "Server is up at %q", callbackUrl)
		} else if strings.HasPrefix(uri.RequestURI(), "/solution/templates") && len(uri.RequestURI()) > len("/solution/templates") {
			err = t.readTemplate(w, r, uri)
		} else {
			log.Errorf("This URI is not recognized, please check the path")
			fmt.Fprint(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		}
	case "DELETE":
		if strings.HasPrefix(uri.RequestURI(), "/solution/templates") && len(uri.RequestURI()) > len("/solution/templates") {
			err = t.deleteTemplateFile(w, r, uri)
		} else {
			log.Errorf("This URI is not recognized, please check the Method")
			fmt.Fprint(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		}
	case "OPTION":
		log.Error("Method OPTION is not supported")
		fmt.Fprint(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
	default:
		log.Errorf("Method %v is not recognized, please check the path", r.Method)
		fmt.Fprint(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
	}
	if err != nil {
		log.Errorf("Error while parsing request: %v", err)
		fmt.Fprint(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
	}
	if terminate {
		t.TerminateChan <- terminate
	}
}

func (t *templateServer) readTemplate(w http.ResponseWriter, r *http.Request, uri *url.URL) error {
	URIParts := strings.Split(uri.RequestURI(), "/")
	file, err := afero.ReadFile(t.Fs, t.TemplatePath+URIParts[len(URIParts)-1])
	log.Info("Getting file at " + t.TemplatePath + URIParts[len(URIParts)-1])
	if err != nil {
		log.Errorf("No such template file in this directory: %v", err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return err
	} else {
		// temporary support for content-type
		if strings.HasSuffix(URIParts[len(URIParts)-1], ".html") {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
		}

		// send file contents
		fmt.Fprint(w, string(file))
		return nil
	}
}

func (t *templateServer) returnTemplateList(w http.ResponseWriter, r *http.Request) error {
	files, err := afero.ReadDir(t.Fs, t.TemplatePath)
	if err != nil {
		log.Errorf("Cannot read the files: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return err
	}

	var itemsList apiItemList
	itemArray := make([]apiItem, 0)
	for i := 0; i < len(files); i++ {
		var item apiItem
		item.Name = files[i].Name()
		itemArray = append(itemArray, item)
	}

	itemsList.Items = itemArray
	itemsList.NumItems = len(files)
	jsonOutput, err := json.MarshalIndent(&itemsList, "", "  ")
	if err != nil {
		log.Error(err.Error())
		return err
	}
	fmt.Fprint(w, string(jsonOutput))

	return nil
}

func (t *templateServer) updateTemplate(w http.ResponseWriter, r *http.Request, uri *url.URL) error {
	//var body string
	b, err := io.ReadAll(r.Body)
	if err != nil {
		log.Errorf("Cannot read request body: %v", err)
	}
	URIParts := strings.Split(uri.RequestURI(), "/")
	err = afero.WriteFile(t.Fs, t.TemplatePath+URIParts[len(URIParts)-1], b, 0644)
	if err != nil {
		log.Errorf("Error writing to this file: %v", err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return err
	} else {
		fmt.Fprint(w, "Successfully wrote to "+t.TemplatePath+URIParts[len(URIParts)-1])
		return nil
	}
}

func (t *templateServer) deleteTemplateFile(w http.ResponseWriter, r *http.Request, uri *url.URL) error {
	URIParts := strings.Split(uri.RequestURI(), "/")
	err := t.Fs.Remove(t.TemplatePath + URIParts[len(URIParts)-1])
	if err != nil {
		log.Errorf("Error in deleting file: %v", err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return err
	}
	fmt.Fprint(w, "File Successfully Deleted")
	return nil
}

func checkURI(w http.ResponseWriter, r *http.Request) (*url.URL, error) {
	_, err := url.Parse(callbackUrl)
	if err != nil {
		log.Errorf("Unexpected failure to obtain expected callback path (likely a bug): %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return nil, err
	}
	uri, err := url.Parse(r.RequestURI)
	if err != nil {
		log.Errorf("Unexpected failure to parse callback path received (malformed request?): %v", err)
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

func openBrowser(url string) error {
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

func getTemplatePath(manifest *Manifest) (string, error) {
	// locate the one and only one template directory
	var templatePath string
	numTemplateObjects := 0
	for _, obj := range manifest.Objects {
		if obj.Type == templateType {
			numTemplateObjects += 1
			if obj.ObjectsDir != "" && templatePath == "" {
				templatePath = obj.ObjectsDir
			}
		}
	}

	// report error
	switch numTemplateObjects {
	case 0:
		return "", fmt.Errorf("No template object (type %q) found in the manifest. Use `solution extend --add-dash-ui` to add.", templateType)
	case 1:
		break
	default:
		return "", fmt.Errorf("Multiple template objects not supported by fsoc; use exactly one object of type objectDir")
	}
	if templatePath == "" {
		return "", fmt.Errorf("No template object directory found; fsoc supports only a template object with objectsDir")
	}

	// check format
	if strings.HasPrefix(templatePath, "/") {
		return "", fmt.Errorf("Template path %q cannot start with \"/\"", templatePath)
	}
	if strings.HasSuffix(templatePath, "/") {
		return "", fmt.Errorf("Template path %q cannot end with \"/\"", templatePath)
	}

	return templatePath, nil
}
