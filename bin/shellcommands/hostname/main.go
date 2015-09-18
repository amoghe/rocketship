package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"rocketship/commander/modules/host"

	"github.com/parnurzeal/gorequest"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	getCmdStr = "show"
	putCmdStr = "configure"

	getCmd = kingpin.Command(getCmdStr, "Display the currently configured hostname")
	putCmd = kingpin.Command(putCmdStr, "Configure the system hostname")

	name = putCmd.Flag("hostname", "Hostname to be configured").String()
)

func main() {
	kingpin.Version("1.0")
	mode := kingpin.Parse()

	switch mode {
	case getCmdStr:
		doGetHostname()
	case putCmdStr:
		doPutHostname()
	default:
		fmt.Println("Unknown subcommand:", mode)
		os.Exit(1)
	}
}

func doGetHostname() {

	req := gorequest.New()
	_, body, errs := req.Get("http://localhost:8888" + host.EHostname).End()
	if errs != nil {
		fmt.Println(errs)
		os.Exit(1)
	}

	h := host.HostnameResource{}
	if err := json.Unmarshal([]byte(body), &h); err != nil {
		fmt.Println("Failed to parse JSON response from server")
		os.Exit(1)
	}

	fmt.Println("Configured hostname:", h.Hostname)
}

func doPutHostname() {
	var (
		h   = host.HostnameResource{Hostname: *name}
		req = gorequest.New()
	)

	if len(*name) <= 0 {
		fmt.Println("Cannot set empty hostname")
		os.Exit(1)
	}

	res, body, errs := req.
		Put("http://localhost:8888" + host.EHostname).
		Send(h).
		End()
	if errs != nil {
		fmt.Println(errs)
		os.Exit(1)
	}
	if res.StatusCode != http.StatusOK {
		fmt.Println("Error response from server:")
		fmt.Println("\tCode:\t", res.StatusCode)
		fmt.Println("\tBody:\t", body)
		os.Exit(1)
	}

	fmt.Println("Hostname updated succesfully")
}
