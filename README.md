# Rocketship

Rocketship helps you launch your software on a linux appliance (physical or virtual).

## Motivation

`rocketship` provides all the generic functionality that your customers may expect
from an appliance. These are features that you must develop and maintain in order to
provide your customers a usable appliance. These include:
* System metrics
* Crash reporting
* Easy (atomic) upgrades
* Command line interface (CLI)

## Development

`rocketship` software is written using go(lang). In order to build the project, you
need `go` to be installed and a `$GOPATH` to be setup.

1. Check out this repo into your `$GOPATH`
2. Run `./bootstrap-dev.sh`, to detect whether the prerequisite tools are present
3. Install any missing tools as advised by the output from `bootstrap-dev.sh`
4. Run `rake -T` to list all the build targets
5. Invoke the necessary build target
