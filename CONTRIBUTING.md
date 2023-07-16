# Contributing to fsoc

We are working to set up the contribution guidelines and policies for this project. 

While at this time we are not accepting contributions, if there is a bugfix or a suggestion or you just need help, please open an issue in this repository.

## Building fsoc

### Set up a Go development environment

fsoc is written in Go (1.20+); both development and usage are intentionally multiplatform. Supported development environments include: Linux (e.g., Ubuntu), Mac OS (Intel, M1) and Windows 10/11 with WSL (non-WSL environment may be supported if there is interest).

To develop fsoc, you will need the following tools:

1. git
1. Go 1.20.1+ (follow the instructions at https://go.dev/doc/install)
1. GNU Make (install with `sudo apt install` make on Ubuntu/Debian)
1. goimports (install with `go install golang.org/x/tools/cmd/goimports@latest`)
1. godoc (only if you want to see fsoc packages docs in a browser on your laptop, install with `go install golang.org/x/tools/cmd/godoc@latest`)

### Clone the fsoc repository

Grab the latest fsoc from Github:

```
git clone https://github.com/cisco-open/fsoc.git
```

### Quick local build

To build fsoc locally, after cloning (and possibly modifying) this repository:

1. Run `go build` (or `make dev-build`, which fills in the version/git/build info better but may be a bit slower)
1. Use the binary saved in the same directory, e.g., `./fsoc help`

### Multiplatform build

To build fsoc binaries for all supported environments, run:

```
make build
```

This command will build the utility for the following supported target environments:

* Mac OS (Darwin) Intel - amd64
* Mac OS (Darwin) M1 - arm64
* Linux Intel - amd64
* Linux ARM - arm64
* Windows 10/11 - amd64

The binaries will be placed in the `builds/` directory.

### Linting, formatting, etc.

The fsoc project is set up with several tools that help maintain uniformity even as it being developed by multiple teams.

To run all the tools (e.g., when preparing for a commit):

```
make pre-commit
```

Some of the individual tools are also available, as `make lint`, `make vet`, `make go-impi`, `make tidy`.

### Running fsoc unit tests

To run all existing fsoc unit tests:

```
make dev-test
```
