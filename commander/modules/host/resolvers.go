package host

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"

	"github.com/zenazn/goji/web"
)

const (
	etcResolvConfPath = "/etc/resolv.conf"

	runResolvConfPath = "/run/resolvconf/resolv.conf"
)

//
// Handlers
//

func (c *Controller) SetResolvers(ctx web.C, w http.ResponseWriter, r *http.Request) {
	bodybytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		c.jsonError(err, w)
		return
	}

	rcfg := ResolversConfig{}
	if err := json.Unmarshal(bodybytes, &rcfg); err != nil {
		c.jsonError(err, w)
		return
	}

	rcfg.ID = 1 // We always operate on the first row
	if err := c.db.Save(&rcfg).Error; err != nil {
		c.jsonError(err, w)
		return
	}

	bytes, err := json.Marshal(&rcfg)
	if err != nil {
		c.jsonError(err, w)
		return
	}

	_, err = w.Write(bytes)
	if err != nil {
		c.jsonError(err, w)
		return
	}
}

func (c *Controller) GetResolvers(ctx web.C, w http.ResponseWriter, r *http.Request) {
	rcfg := ResolversConfig{}
	if err := c.db.First(&rcfg, 1).Error; err != nil {
		c.jsonError(err, w)
		return
	}

	bytes, err := json.Marshal(&rcfg)
	if err != nil {
		c.jsonError(err, w)
		return
	}

	_, err = w.Write(bytes)
	if err != nil {
		c.jsonError(err, w)
		return
	}
}

//
// File operations
//
func (c *Controller) RewriteResolvConf() error {
	// we don't actually rewrite it, just ensure it is a symlink
	c.log.Infoln("Ensuring resolv.conf symlink")

	if err, _ := os.Stat(etcResolvConfPath); err != nil {
		// delete it
		os.Remove(etcResolvConfPath)
	}

	if err := os.Symlink(runResolvConfPath, etcResolvConfPath); err != nil {
		return fmt.Errorf("Failed to ensure symlink: %s", err)
	}
	return nil
}

//
// DB models
//

type ResolversConfig struct {
	ID           int `json:"-"`
	DNSServerIP1 string
	DNSServerIP2 string
	DNSServerIP3 string
}

func (c *ResolversConfig) BeforeSave() error {
	for _, server := range []string{c.DNSServerIP1, c.DNSServerIP2, c.DNSServerIP3} {
		if len(server) > 0 {
			if net.ParseIP(server) == nil {
				return fmt.Errorf("%s is not a valid IP", server)
			}
		}
	}
	return nil
}

func (c *Controller) seedResolvers() {
	c.log.Infoln("Seeding resolvers")
	c.db.FirstOrCreate(&ResolversConfig{
		DNSServerIP1: "8.8.8.8",
		DNSServerIP2: "8.8.4.4",
		DNSServerIP3: "",
	})
}
