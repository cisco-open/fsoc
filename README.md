# fsoc - Cisco FSO Platform Developer's Control Tool

[![Release](https://img.shields.io/github/release/cisco-open/fsoc.svg?style=for-the-badge)](https://github.com/cisco-open/fsoc/releases/latest)
[![stability-alpha](https://img.shields.io/badge/stability-alpha-f4d03f.svg?style=for-the-badge)](https://github.com/mkenney/software-guides/blob/master/STABILITY-BADGES.md#alpha)
[![Go Report Card](https://goreportcard.com/badge/github.com/cisco-open/fsoc?style=for-the-badge)](https://goreportcard.com/report/github.com/cisco-open/fsoc) 
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg?style=for-the-badge)](LICENSE)
[![Github All Releases](https://img.shields.io/github/downloads/cisco-open/fsoc/total.svg?style=for-the-badge)](https://github.com/cisco-open/fsoc/releases/latest) 


The Cisco Full Stack Observability (FSO) Platform  provides core capabilities for developers to build observability solutions to gain visibility and actionable insights across their technology and business stack. The platform leverages OpenTelemetry collections to collect MELT* telemetry and then transforms the raw data into flexible and scalable objects that can be correlated and queried.

*MELT: Metrics, Events, Logs and Traces

The FSO control tool, `fsoc`, provides a command line interface to help developers manage their solutions 
lifecycle and interact with the core services and solutions currently available in the platform.

## Documentation

The `fsoc` documentation is going to be published in Cisco's [DevNet](https://developer.cisco.com/docs/fso/). This link may not work until the first document publication is available. Until then, the `fsoc help` command is a good starting point. Most commands provide sample command lines you can try.

You can also run `fsoc gendocs` to generate a command reference. It provides the same information as `fsoc help` but in static Markdown pages.

## TL;DR Build

To build `fsoc` locally, after cloning this repository:

* Run `go build`
* Use the binary saved in the same directory, e.g., `./fsoc help`

For more information on setting up the development environment and building fsoc, please see [CONTRIBUTING](CONTRIBUTING.md).

## Install using Homebrew

1. Install homebrew if not already installed from https://brew.sh
2. Install `fsoc` using homebrew
    ```
    brew tap cisco-open/tap
    brew install fsoc
    ```

## TL;DR Install Prebuilt Binaries

Prebuilt binaries are published for each [`fsoc` release](https://github.com/cisco-open/fsoc/releases) for the following platforms:

| Platform | Binary file name |
| --- | --- |
| Mac OS, Intel | `fsoc-darwin-amd64` |
| Mac OS, M1/M2 | `fsoc-darwin-arm64` |
| Linux, Intel/AMD | `fsoc-linux-amd64` |
| Linux, ARM | `fsoc-linux-arm64` |
| Windows 10/11 | `fsoc-windows-amd64.exe` |

### Installing on Linux or Windows with WSL

```
FSOCOS=linux-amd64 \
bash -c 'curl -fSL -o fsoc "https://github.com/cisco-open/fsoc/releases/latest/download/fsoc-${FSOCOS}"'
chmod +x fsoc
sudo mv fsoc /usr/local/bin
```
Change the `FSOCOS` platform name above to `linux-arm64` for installing on Linux/ARM.

### Installing on Mac OS, Intel

```
curl -fSL -o fsoc "https://github.com/cisco-open/fsoc/releases/latest/download/fsoc-darwin-amd64"
chmod +x fsoc
sudo mv fsoc /usr/local/bin
```

### Installing on Mac OS, M1/M2

```
curl -fSL -o fsoc "https://github.com/cisco-open/fsoc/releases/latest/download/fsoc-darwin-arm64"
chmod +x fsoc
sudo mv fsoc /usr/local/bin
```

### Installing On Windows

If you will run `fsoc` on the Windows Subsystem for Linux (WSL), please use the Linux and WSL instructions above.

For installing `fsoc` as a native application on Windows, follow these steps:

1. Download the [latest release](https://github.com/cisco-open/fsoc/releases/latest/download/fsoc-windows-amd64.exe). If you have curl installed, you can run the following command in cmd.exe or Powershell:
```
curl -fSL -o fsoc "https://github.com/cisco-open/fsoc/releases/latest/download/fsoc-windows-amd64.exe"
```

2. Append or prepend the `fsoc` binary folder to your PATH environment variable.

3. Test to ensure the version of `fsoc` is the same as the [latest](https://github.com/cisco-open/fsoc/releases/latest):

```
fsoc version
```

## Set Shell Autocompletion

This is an optional step. To add autocompletion in bash, run:

```
. <(./fsoc completion bash)
```

For other shells, check out the completion help with `fsoc help completion`.

## Configure

Configure the default profile to your tenant of choice (replace MYTENANT with your tenant's name):

```
fsoc config set auth=oauth url=https://MYTENANT.observe.appdynamics.com
fsoc login  # test access
```

NOTE: The login command will pop up a browser to perform the log in and then continue executing the command. Subsequent invocations of fsoc will use cached credentials. 

## Assistance and Suggestions

We are working to provide channels for help, suggestions, etc., for this project. In the meantime, if you have suggestions or want to report a problem, please use Github issues.
