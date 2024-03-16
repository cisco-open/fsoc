# Using Core FSOC Services

In addition to using the Go standard library and the spf13/cobra package, fsoc provides some common packages that are specific to FSO and help build domain commands.

You can see documentation for them by running `go doc <path>`, e.g., `go doc platform/api` .

You can see how they are used by looking at some of the simpler domain commands (e.g., `uql/query.go` and `optimize/report.go` )

## Config Data (Context)

Package `cmd/config`: config data access (context)

- `GetCurrentContext()` - get the current profile's configuration
- `UpdateCurrentContext()` - update the current context (should not be needed often but it's available)

## Platform API

Package `platform/api`: platform API call wrappers that provide transparent authentication and payload encoding/decoding

- `JSONGet()` - perform a GET REST request, parse result as a JSON object (freeform or structured)
- `JSONPost()` - perform a POST REST request, with request and response payloads as JSON (freeform or structured)
- `JSONPatch()` - perform a JSON Patch request
- `JSONRequest()` - perform any of the above requests, specifying the HTTP method as a parameter
- `JSONGetCollection()` - perform one or more GET REST requests to collect all items, paginating if necessary (following [RFC31](https://confluence.corp.appdynamics.com/display/ENG/RFC+31+%3A+API+Standards#RFC31:APIStandards-Pagination))
- `HTTPGet()` - perform a GET request, allowing for non-JSON response payload (e.g., downloading a solution archive)
- `HTTPPost()` - perform a POST request, allowing for non-JSON request payload (e.g., uploading a solution archive)

## Structured Logging

Package `github.com/apex/log`: [documentation](https://pkg.go.dev/github.com/apex/log)

- log.Infof(), log.Warnf(), log.Errorf(), log.Fatalf() - the obvious
- log.WithFields(log.Fields{"key1": "value1", "key2": "value2"}).Info("Operation")

## Command Line Parsing

Package `github.com/spf13/cobra`: [documentation](https://pkg.go.dev/github.com/spf13/cobra)

Used along with spf13/viper for env vars and spf13/pflag for command line flags

## Command Output

Package `output` 

- `PrintCmdOutput()` - print command's results, in format depending on the `--output` flag (human, json or yaml). JQ queries specify human output formats for table and detail in the command structure.
- `PrintCmdOutputCustom()` - print command's results with prepared table output (in case human output is needed)
- `PrintCmdStatus()` - print a simple string to the output, allowing for progress (e.g., `\r` ) and final output. Remember to add `\n` when printing messages.
- `PrintYAML()` - print data in YAML format
- `PrintJSON()` - print data in JSON format

## Status And Progress

Use the `PrintCmdStatus()` function to display progress for commands that don't output data to stdout.

## Command Kit (Utility)

Package `cmdkit`: a toolkit for building fsoc commands with minimum boilerplate

- `FetchAndPrint()` - calls a platform API and displays the output, handling errors. A one-line version of otherwise more verbose `JSONXxxx()` and `PrintCmdOutput()`. For an example on using this function, see `cmd/solution/list.go`. The `IsCollection` option specifies that a possibly-paginated collection is being retrieved (`JSONGetCollection()`+`PrintCmdOutput()`)