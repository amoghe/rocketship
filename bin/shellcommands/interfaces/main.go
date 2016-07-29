package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"rocketship/commander/modules/host"
	"strings"

	"github.com/alecthomas/kingpin"
	"github.com/parnurzeal/gorequest"
)

var (
	// The 'gorequest' SuperAgent we'll use for all our requests.
	req = gorequest.New()

	listCmdStr = "list"
	showCmdStr = "show"
	editCmdStr = "edit"

	listCmd = kingpin.Command(listCmdStr, "List interfaces")
	editCmd = kingpin.Command(editCmdStr, "Edit an interface")
	showCmd = kingpin.Command(showCmdStr, "Show details for an interface")

	// show opts
	showname = showCmd.Flag("name", "Name of interface to display").String()

	// edit opts
	name    = editCmd.Flag("name", "Name of interface to edit").String()
	mode    = editCmd.Flag("mode", "Interface mode").Enum(host.ModeDHCP, host.ModeStatic)
	address = editCmd.Flag("address", "Interface address. Only for "+host.ModeStatic).IP()
	gateway = editCmd.Flag("address", "Default gateway. Only for "+host.ModeStatic).IP()
	netmask = editCmd.Flag("address", "Netmask. Only for "+host.ModeStatic).Default("255.255.255.0").IP()
)

func main() {
	kingpin.Version("1.0")
	mode := kingpin.Parse()

	switch mode {
	case listCmdStr:
		doListInterfaces()
	case showCmdStr:
		doShowInterface()
	case editCmdStr:
		doEditInterface()
	default:
		fmt.Println("Unknown subcommand:", mode)
		os.Exit(1)
	}
}

func doListInterfaces() {

	_, body, errs := req.Get("http://localhost:8888" + host.EInterfaces).End()
	if errs != nil {
		fmt.Println(errs)
		os.Exit(1)
	}

	interfaceNames := []string{}
	if err := json.Unmarshal([]byte(body), &interfaceNames); err != nil {
		fmt.Println("Failed to parse json response from server:", err)
		os.Exit(1)
	}

	fmt.Println("Configured interfaces:")
	for _, name := range interfaceNames {
		fmt.Printf("\t%s\n", name)
	}
}

func doShowInterface() {

	if len(*showname) <= 0 {
		fmt.Println("No interface name specified")
		os.Exit(1)
	}

	endpoint := strings.Replace(host.EInterfacesID, ":id", *showname, 1)
	fmt.Println("endpoint", endpoint)

	res, body, errs := req.Get("http://localhost:8888" + endpoint).End()
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

	iface := host.InterfaceConfigResource{}
	if err := json.Unmarshal([]byte(body), &iface); err != nil {
		fmt.Println("Failed to parse JSON response from server")
		os.Exit(1)
	}

	fmt.Println("Interface details:")
	fmt.Println("\tName\t\t:", iface.Name)
	fmt.Println("\tMode\t\t:", iface.Mode)
	if *mode == host.ModeStatic {
		fmt.Println("\tAddress\t\t:", iface.Address)
		fmt.Println("\tGateway\t\t:", iface.Gateway)
		fmt.Println("\tNetmask\t\t:", iface.Netmask)
	} else {
		fmt.Println("\tDHCP Profile ID\t:", iface.DHCPProfileID)
	}

	fmt.Printf("\tInterface status:\n\n")
	for _, line := range strings.Split(iface.InterfaceStatus, "\n") {
		fmt.Println("\t\t", line)
	}

}

func doEditInterface() {

	if len(*name) <= 0 {
		fmt.Println("Interface name not specified")
		os.Exit(1)
	}

	iface := host.InterfaceConfigResource{
		InterfaceConfig: host.InterfaceConfig{
			Name:    *name,
			Mode:    *mode,
			Enabled: true, // TODO: server doesn't support this yet?
		},
	}

	if *mode == host.ModeStatic {
		iface.InterfaceConfig.Address = address.String()
		iface.InterfaceConfig.Gateway = gateway.String()
		iface.InterfaceConfig.Netmask = netmask.String()
	} else {
		//iface.DHCPProfileID = *dhcpProfileID
	}

	endpoint := strings.Replace(host.EInterfacesID, ":id", *name, 1)

	res, body, errs := req.
		Put("http://localhost:8888" + endpoint).
		Send(iface).
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
