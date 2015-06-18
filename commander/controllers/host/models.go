package dhcp

import (
	"fmt"
	"net"
	"strings"

	"github.com/jinzhu/gorm"
)

var (
	// Default options we request from the dhcp server
	DefaultRequestOptions = []string{
		"subnet-mask",
		"broadcast-address",
		"time-offset",
		"routers",
		"domain-name",
		"domain-name-servers",
		"domain-search",
		"host-name",
		"netbios-name-servers",
		"netbios-scope",
		"interface-mtu",
		"rfc3442-classless-static-routes",
		"ntp-servers",
		"dhcp6.domain-search",
		"dhcp6.fqdn",
		"dhcp6.name-servers",
		"dhcp6.sntp-servers",
	}
)

const (
	ModeDHCP   = "dhcp"
	ModeStatic = "static"
)

type Hostname struct {
	ID       int64 `json:"-"`
	Hostname string
}

type Domain struct {
	ID     int64 `json:"-"`
	Domain string
}

type DHCPProfile struct {
	ID               int64
	DefaultOptions   string
	AppendOptions    string
	PrependOptions   string
	SupersedeOptions string
	SendOptions      string
	DNSMode          string
	RequireOptions   string
	RequestOptions   string
}

type InterfaceConfig struct {
	ID      int64
	Name    string
	Enabled bool
	Mode    string

	Address string
	Gateway string
	Netmask string

	DHCPProfileID int64
}

//
// Callbacks
//

func (i *InterfaceConfig) BeforeSave(txn *gorm.DB) error {
	switch i.Mode {
	case ModeStatic:
		return i.validateIPs()
	case ModeDHCP:
		return i.validateDHCPProfile(txn)
	default:
		return fmt.Errorf("Invalid mode (%s) set for interface %s", i.Mode, i.Name)
	}
	return nil
}

func (d *DHCPProfile) BeforeCreate(txn *gorm.DB) error {
	if len(d.RequestOptions) <= 0 {
		txn.Model(d).Update(DHCPProfile{
			RequestOptions: strings.Join(DefaultRequestOptions, ","),
		})
	}
	return nil
}

//
// helpers
//

func (i *InterfaceConfig) validateDHCPProfile(txn *gorm.DB) error {
	if i.Mode == ModeDHCP {
		dp := DHCPProfile{}
		if err := txn.Find(&dp, i.DHCPProfileID).Error; err != nil {
			return fmt.Errorf("Cannot save interface %s (id:%d) with DHCP profile %d",
				i.Name, i.ID, i.DHCPProfileID)
		}
	}
	return nil
}

func (i *InterfaceConfig) validateIPs() error {
	addrs := []struct {
		ipstr string
		name  string
	}{
		{ipstr: i.Address, name: "IP"},
		{ipstr: i.Gateway, name: "Gateway"},
		{ipstr: i.Netmask, name: "Netmask"},
	}

	for _, addr := range addrs {
		a := net.ParseIP(addr.ipstr)
		if a == nil {
			return fmt.Errorf("Invalid %s address", addr.name)
		}
	}

	// validate netmask
	nm := net.IPMask(net.ParseIP(i.Netmask).To4())
	ones, bits := nm.Size()
	if ones == 0 && bits == 0 {
		return fmt.Errorf("Invalid netmask (%s)", i.Netmask)
	}

	// ensure gateway is within the network defined by addressr+netmask
	ipnet := net.IPNet{IP: net.ParseIP(i.Address), Mask: nm}
	if !ipnet.Contains(net.ParseIP(i.Gateway)) {
		return fmt.Errorf("Gateway %s is not on network (addr: %s mask %s)",
			i.Gateway, i.Address, i.Netmask)
	}

	return nil
}
