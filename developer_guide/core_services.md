# Using Core FSOC Services

In addition to using the Go standard library and the spf13/cobra package, fsoc provides some common packages that are specific to FSO and help build domain commands.

You can see documentation for them by running `go doc <path>`, e.g., `go doc platform/api` .

You can see how they are used by looking at some of the simpler domain commands (e.g., `iamrole/permissions.go` and `optimize/report.go`).

* [Config Data (Context)](#config-data-context)
* [Platform API](#platform-api)
* [Structured Logging](#structured-logging)
* [Command Line Parsing](#command-line-parsing)
* [Command Output](#command-output)
* [Status And Progress](#status-and-progress)
* [Command Kit (Utility)](#command-kit-utility)

## Config Data (Context)

Package `config`: config data access (context)

- `GetCurrentContext()` - get the current profile's configuration
- `UpdateCurrentContext()` - update the current context (should not be needed often but it's available)

## Platform API

Package `platform/api`: platform API call wrappers that provide transparent authentication and payload encoding/decoding

- `JSONGet()` - perform a GET REST request, parse result as a JSON object (freeform or structured)
- `JSONPost()` - perform a POST REST request, with request and response payloads as JSON (freeform or structured)
- `JSONPatch()` - perform a JSON Patch request
- `JSONRequest()` - perform any of the above requests, specifying the HTTP method as a parameter
- `JSONGetCollection()` - perform one or more GET REST requests to collect all items, paginating if necessary (following [Cisco REST API conventions](https://developer.cisco.com/api-guidelines/#rest-conventions/API.REST.CONVENTIONS.07))
- `HTTPGet()` - perform a GET request, allowing for non-JSON response payload (e.g., downloading a solution archive)
- `HTTPPost()` - perform a POST request, allowing for non-JSON request payload (e.g., uploading a solution archive)

## Structured Logging

Package `github.com/apex/log`: [documentation](https://pkg.go.dev/github.com/apex/log)

- `log.Infof()`, `log.Warnf()`, `log.Errorf()`, `log.Fatalf()` - the obvious
- `log.WithFields(log.Fields{"key1": "value1", "key2": "value2"}).Info("Operation")`

## Command Line Parsing

Package `github.com/spf13/cobra`: [documentation](https://pkg.go.dev/github.com/spf13/cobra)

Used along with spf13/viper for env vars and spf13/pflag for command line flags

## Command Output

Package `output` 

- `PrintCmdOutput()` - print command's results, in format depending on the `--output` flag (human, json or yaml). JQ queries specify human output formats for table and detail in the command structure.
- `PrintCmdOutputCustom()` - print command's results with prepared table output (in case human output is needed)
- `PrintCmdStatus()` - print a simple string to the output, allowing for progress (e.g., `\r` ) and final output. Remember to add `\n` when printing messages.
- `PrintYAML()` - print data in YAML format (deprecated in favor of `PrintCmdOutput()`)
- `PrintJSON()` - print data in JSON format (deprecated in favor of `PrintCmdOutput()`)

## Status And Progress

Use the `PrintCmdStatus()` function from the `output` package to display progress and final status for commands that don't otherwise output data to stdout. For example, the `fsoc iam-role-binding remove` command displays `"Roles removed successfully.\n"` using this function to indicate its success.

## Command Kit (Utility)

Package `cmdkit`: a toolkit for building fsoc commands with minimum boilerplate

- `FetchAndPrint()` - calls a platform API and displays the output, handling errors. A one-line version of otherwise more verbose `JSONGet()` and `PrintCmdOutput()`. 

For an example on using this function, see:

* for a single object, [`cmd/iamrole/list.go`](../cmd/iamrole/permissions.go).  
* for a collection, [`cmd/iamrole/list.go`](../cmd/iamrole/list.go). The `IsCollection` option specifies that a possibly-paginated collection is being retrieved (`JSONGetCollection()`+`PrintCmdOutput()`).

With this function, it is possible to have one-liner handlers, for example:

```
...
var iamRoleListCmd = &cobra.Command{
	Use:   "list",
	Short: "List roles",
	Example: `
  fsoc iam-role list
  fsoc iam-role list -o json
  fsoc iam-role list -o detail`,
	Args: cobra.NoArgs,
	Run:  listRoles,
	Annotations: map[string]string{
		output.TableFieldsAnnotation:  "id:.id, name:.data.displayName, description:.data.description",
		output.DetailFieldsAnnotation: "id:.id, name:.data.displayName, description:.data.description, permissions:(reduce .data.permissions[].id as $o ([]; . + [$o])), scopes:.data.scopes",
	},
}
...

func listRoles(cmd *cobra.Command, args []string) {
	cmdkit.FetchAndPrint(cmd, "/iam/policy-admin/v1beta2/roles", &cmdkit.FetchAndPrintOptions{IsCollection: true})
}

```

Note the annotations that define the fields to be displayed from the retrieved objects.