# fsoc - Cisco FSO Platform Developer's Control Tool

<!-- [![Release](https://img.shields.io/github/release/cisco-open/fsoc.svg?style=for-the-badge)](https://github.com/cisco-open/fsoc/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/cisco-open/fsoc?style=for-the-badge)](https://goreportcard.com/report/github.com/cisco-open/fsoc) -->
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg?style=for-the-badge)](LICENSE)
[![stability-alpha](https://img.shields.io/badge/stability-alpha-f4d03f.svg?style=for-the-badge)](https://github.com/mkenney/software-guides/blob/master/STABILITY-BADGES.md#alpha)
<!--
[![Github All Releases](https://img.shields.io/github/downloads/cisco-open/fsoc/total.svg?style=for-the-badge)](https://github.com/cisco-open/fsoc/releases/latest) -->


The Cisco Full Stack Observability (FSO) Platform provides a set of core capabilities for developers to build full-stack 
observability solutions that deliver visibility, insight and actions on top of MELT* telemetry to 
support their observability goals across the domains representing their technology and business 
stack.

The FSO control (fsoc) tool provides a command line interface to help developers manage their solutions 
lifecycle and interact with the core services and solutions currently available in the platform.

*MELT: Metrics, Events, Logs and Traces

## Documentation

The fsoc documentation is going to be published in Cisco's [DevNet](https://developer.cisco.com/docs/fso/). This link may not work until the first document publication is available. Until then, the `fsoc help` command is a good starting point. Most commands provide sample command lines you can try.

You can also run `fsoc gendocs` to generate a command reference. It provides the same information as `fsoc help` but in static Markdown pages.

## TL;DR Build

To build fsoc locally, after cloning this repository:

* Run `go build`
* Use the binary saved in the same directory, e.g., `./fsoc help`

For more information on setting up the development environment and building fsoc, please see [CONTRIBUTING](CONTRIBUTING.md).

## TL;DR Install Prebuilt Binaries

To download the prebuilt binaries for any of the supported environments, go to the [releases](https://github.com/cisco-open/fsoc/releases).

## Set Shell Autocompletion

This is an optional step. To add autocompletion in bash, run:

```
. <(./fsoc completion bash)
```

For other shells, check out the completion help with `fsoc help completion`.

## Configure

Configure the default profile to your tenant of choice (replace MYTENANT with your tenant's name):

```
fsoc config set --auth=oauth --server=MYTENANT.observe.appdynamics.com
fsoc login  # test access
```

NOTE: The login command will pop up a browser to perform the log in and then continue executing the command. Subsequent invocations of fsoc will use cached credentials. 

## Assistance and Suggestions

We are working to provide channels for help, suggestions, etc., for this project. In the meantime, if you have suggestions or want to report a problem, please use Github issues.
