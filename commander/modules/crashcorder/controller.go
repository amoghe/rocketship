package crashcorder

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/jinzhu/gorm"

	"rocketship/commander/modules/host"
	"rocketship/commander/modules/radio"
	"rocketship/crashcorder"
)

const (
	URLPrefix = "/crashcorder"

	KernelCorePatternFilePath = "/proc/sys/kernel/core_pattern"
	CoresDirPath              = "/cores"
	CorePattern               = "%e_%p_%u_%g_%s_%t"

	CrashcorderConfDir  = "/etc/crashcorder"
	CrashcorderConfFile = CrashcorderConfDir + "/crashcorder.conf"
)

type Controller struct {
}

func NewController(*gorm.DB) *Controller {
	return &Controller{}
}

// ServeHTTP satisfies the http.Handler interface (net/http as well as goji)
func (c *Controller) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// No handlers (yet)
	return
}

// RoutePrefix returns the prefix under which this router handles endpoints
func (c *Controller) RoutePrefix() string {
	return URLPrefix
}

func (c *Controller) RewriteFiles() error {
	// ensure crashcorder dir
	if _, err := os.Stat(CrashcorderConfDir); os.IsNotExist(err) {
		err = os.Mkdir(CrashcorderConfDir, 0750)
		if err != nil {
			return err
		}
	}

	// write config file
	contents, err := c.crashcorderConfigFileContents()
	if err != nil {
		return fmt.Errorf("Failed to generate radio config file contents: %s", err)
	}
	err = ioutil.WriteFile(CrashcorderConfFile, contents, 0644)
	if err != nil {
		return fmt.Errorf("Failed to write file: %s", err)
	}

	// ensure dir and file perms
	ccUser, _ := host.GetSystemUser("crashcorder")

	// ensure dir and file perms
	for _, f := range []string{CrashcorderConfDir, CrashcorderConfFile} {
		os.Chown(f, int(ccUser.Uid), int(ccUser.Gid))
	}

	return nil
}

func (c *Controller) crashcorderConfigFileContents() ([]byte, error) {
	cfg := crashcorder.Config{
		CorePatternTokens: strings.Split(CorePattern, "_"),
		CoresDirectory:    CoresDirPath,
		RadioConnectAddr:  net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: radio.RadioPort},
	}

	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return []byte{}, err
	}
	return b, nil
}

func (c *Controller) configureKernelCorePattern() error {
	if f, err := os.Open(KernelCorePatternFilePath); err == nil {
		defer f.Close()
		f.Write([]byte(CorePattern))
		return nil
	} else {
		return err
	}
}

//
// DB
// TODO: add DB models/tables to manage whether crashcorder is enabled
//

func (c *Controller) MigrateDB() {}
func (c *Controller) SeedDB()    {}
