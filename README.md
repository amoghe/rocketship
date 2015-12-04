# Rocketship

Rocketship helps you launch your software on a linux appliance (physical or virtual).

## What is the motivation behind this project?

Yours truly has seen various flavors of the same software at different companies
who were shipping virtual and physical appliances to their customers. Each time
they spent an inordinate amount of time and resources building software that was
not directly related to their core IP. `rocketship` is an attempt to provide the
common core functionality needed in order to ship linux based appliances.

Shipping your software on an appliance involves developing all the pieces of software
that are required to keep the appliance operational and usable. Some of these include:

* System metrics
* Crash reporting
* Easy (atomic) upgrades
* User login account management
* Command line interface (CLI)

`rocketship` intends to provide the platform on which you can ship your software
without needing to develop all these peripheral pieces. You place your binary (or
binaries, and accompanying scripts) in the workspace , and let `rocketship` build
you an appliance image that you can use to build physical and virtual appliances.

## How does `rocketship` do everything it claims to do?

See the wiki for detailed notes on the architecture of the system as well as
the processes that

## What does `rocketship` not handle for me?

Almost all appliances can be configured using a UI that is accessibly from a web
browser. `rocketship` intentionally does NOT ship with any UI because that is a
non-reusable component because every user of project will want to present their
own interface and user experience depending on their business domain. Instead
rocketship exposes a configuration managenemt API which can be invoked from an
UI code.

## How do I completely integrate with `rocketship` beyond just launching my process?

In order to provide a consistent UX, you'll want to integrate further by letting
`rocketship` handle the configuration for your binary so that users can interact with
all system confiration the same way. This involves adding APIs to handle configuration
storage and adding command line tools so users can modify the configuration from the
CLI.

## How do I modify `rocketship` and/or contribute to it?

`rocketship` software is written using go(lang). In order to build the project, you
need `go` to be installed (and `$GOPATH` to be set).

1. Clone this repo into your `$GOPATH`
2. Run `./bootstrap-dev.sh`, to detect whether the prerequisite tools are present
3. Install any missing tools as advised by the output from `bootstrap-dev.sh`
4. Run `rake -T` to list all the build targets
5. Invoke the necessary build target
