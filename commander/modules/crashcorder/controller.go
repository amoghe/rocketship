package crashcorder

import (
	"encoding/json"
	"net"
	"os"
	"strings"

	"rocketship/commander/modules/radio"
	"rocketship/crashcorder"
)

const (
	KernelCorePatternFilePath = "/proc/sys/kernel/core_pattern"
	CoresDirPath              = "/cores"
	CorePattern               = "%e_%p_%u_%g_%s_%t"
)

type Controller struct {
}

func NewController() *Controller {
	return &Controller{}
}

func (c *Controller) RewriteCrashcorderConfigFile() error {
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
