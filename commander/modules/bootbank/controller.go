package bootbank

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"rocketship/regulog"

	"github.com/jinzhu/gorm"
	"github.com/juju/deputy"
	"github.com/zenazn/goji/web"
)

const (
	URLPrefix = "/system"

	KernelCommandlineFile = "/proc/cmdline"

	Bootbank1 = "BOOTBANK1"
	Bootbank2 = "BOOTBANK2"

	GrubPartitionlabel = "GRUB"

	ImageVersionFile = "/etc/rocketship_version"

	EBootbanks  = URLPrefix + "/bootbanks"
	EBootbankID = EBootbanks + "/:id"
)

type Controller struct {
	db   *gorm.DB
	mux  *web.Mux
	log  regulog.Logger
	lock sync.Mutex
}

func NewController(db *gorm.DB, logger regulog.Logger) *Controller {
	ctrl := &Controller{db: db, mux: web.New(), log: logger}

	ctrl.mux.Get(EBootbanks, ctrl.GetBootbanks)
	ctrl.mux.Get(EBootbankID, ctrl.GetBootbankDetails)

	return ctrl
}

// ServeHTTP satisfies the http.Handler interface (net/http as well as goji)
func (c *Controller) ServeHTTPC(ctx web.C, w http.ResponseWriter, r *http.Request) {
	c.lock.Lock()
	c.mux.ServeHTTPC(ctx, w, r)
	c.lock.Unlock()
	return
}

// RoutePrefix returns the URL prefix under which this controller serves its routes
func (c *Controller) RoutePrefix() string {
	return URLPrefix
}

// These satisfy the controller interface.
func (c *Controller) SeedDB()             {}
func (c *Controller) MigrateDB()          {}
func (c *Controller) RewriteFiles() error { return nil }

//
// Response Entities
//

type VersionDetails struct {
	Version string
	Active  bool
}

//
// HTTP handlers
//

func (c *Controller) GetBootbanks(ctx web.C, w http.ResponseWriter, r *http.Request) {
	enc := json.NewEncoder(w)

	if err := enc.Encode([]string{Bootbank1, Bootbank2}); err != nil {
		jsonError(err, w)
	}
}

func (c *Controller) GetBootbankDetails(ctx web.C, w http.ResponseWriter, r *http.Request) {
	bbLabel, _ := ctx.URLParams["id"]
	if bbLabel != Bootbank1 && bbLabel != Bootbank2 {
		jsonError(fmt.Errorf("Invalid bootbank (%s) specified", bbLabel), w)
		return
	}

	var (
		version string
		err     error
		ret     VersionDetails
	)

	readVersionFileFrom := func(dir string) error {
		if vbytes, err := ioutil.ReadFile(dir + "/" + ImageVersionFile); err != nil {
			return err
		} else {
			version = string(vbytes)
		}
		return nil
	}

	if bbLabel == c.currentBootbankLabel() {
		err = readVersionFileFrom("/")
		ret = VersionDetails{Version: version, Active: true}
	} else {
		err = withMountedPartition(c.otherBootbankLabel(), readVersionFileFrom)
		ret = VersionDetails{Version: version, Active: false}
	}
	if err != nil {
		jsonError(fmt.Errorf("Failed to read version file: %s", err), w)
		return
	}
	if err := json.NewEncoder(w).Encode(ret); err != nil {
		jsonError(err, w)
		return
	}
	return
}

func (c *Controller) currentBootbankLabel() string {
	b, err := ioutil.ReadFile(KernelCommandlineFile)
	if err != nil {
		c.log.Infof("Unable to determine bootbank (failed to read cmdline: %s). Assuming %s",
			err, Bootbank1)
		return Bootbank1
	}

	for _, token := range strings.Split(string(b), " ") {
		if strings.HasPrefix(token, "LABEL") {
			return strings.Split(token, "=")[1]
		}
	}

	c.log.Infoln("Unable to determine bootbank (no LABEL found on cmdline). Assuming", Bootbank1)
	return Bootbank1
}

func (c *Controller) otherBootbankLabel() string {
	if c.currentBootbankLabel() == Bootbank1 {
		return Bootbank2
	} else {
		return Bootbank1
	}
}

func (c *Controller) loadImageIntoBootbank(banklabel string, imgFilePath string) error {

	unpackImageIntoDir := func(dirname string) error {
		cmd := exec.Command("tar",
			"--extract",
			"--file="+imgFilePath,
			"--preserve-permissions", // Our job is only to load dir
			"--numeric-owner",
			"-C",
			dirname,
			".",
		)

		d := deputy.Deputy{
			Errors:  deputy.FromStderr,
			Timeout: 60 * time.Second,
		}
		if err := d.Run(cmd); err != nil {
			c.log.Warningln("Failed to unpack image:", err)
		}

		return nil
	}

	if banklabel == c.currentBootbankLabel() {
		return fmt.Errorf("Bootbank is currently active")
	}

	if err := withMountedPartition("/dev/disk/by-label/"+banklabel, unpackImageIntoDir); err != nil {
		return err
	}

	return nil
}

func (c *Controller) makeBootbankBootable(banklabel string) error {

	// write the file in-place. TODO: make atomic using 'mv'
	writeGrubFile := func(mountpoint string) error {
		bootDir := mountpoint + "/boot"
		grubDir := bootDir + "/grub"
		grubFilePath := grubDir + "/grub.cfg"

		if err := os.MkdirAll(grubDir, 0755); err != nil {
			return fmt.Errorf("Failed to ensure grub dir: %s", err)
		}
		if fileContents, err := c.grubConfContents(banklabel); err != nil {
			return fmt.Errorf("Failed to generate grub.conf contents: %s", err)
		} else {
			if err := ioutil.WriteFile(grubFilePath, []byte(fileContents), 0444); err != nil {
				return fmt.Errorf("Failed to write grub config: %s", err)
			}
		}
		return nil
	}

	return withMountedPartition("/dev/disk/by-label/"+GrubPartitionlabel, writeGrubFile)
}

func (c *Controller) grubConfContents(bootableBankLabel string) (string, error) {
	type __bootbankEntry struct {
		MenuEntry         string
		PartitionLabel    string
		KernelCmdlineOpts string
	}

	type __grubTemplateData struct {
		DefaultBootEntry string
		HiddenTimeout    uint32
		BootbankEntries  []__bootbankEntry
	}

	// Given a bootbank label, return a good grub menu entry for it
	menuEntryForBootbankLabel := func(label string) string {
		switch label {
		case Bootbank1:
			return "Rocketship1"
		case Bootbank2:
			return "Rocketship2"
		default:
			c.log.Warningf("Unexpected bootbank label encountered: ", label)
			return "Unknown"
		}
	}

	var tdata = __grubTemplateData{
		DefaultBootEntry: menuEntryForBootbankLabel(bootableBankLabel),
		HiddenTimeout:    5,
		BootbankEntries: []__bootbankEntry{
			{menuEntryForBootbankLabel(Bootbank1), Bootbank1, "rw quiet splash"},
			{menuEntryForBootbankLabel(Bootbank2), Bootbank2, "rw quiet splash"},
		},
	}

	template, err := template.New("grub.cfg").Parse(grubConfTemplateStr)
	if err != nil {
		return "", err
	}

	output := &bytes.Buffer{}
	if err := template.Execute(output, tdata); err != nil {
		return "", err
	}

	return output.String(), nil
}

func withMountedPartition(partition string, f func(string) error) error {
	tempDir, err := ioutil.TempDir(os.TempDir(), "tempMountedPartition")
	if err != nil {
		return err
	}

	if err := syscall.Mount(partition, tempDir, "ext4", syscall.MS_RDONLY, ""); err != nil {
		return err
	}
	defer syscall.Unmount(tempDir, 0)

	return f(tempDir)
}

func jsonError(err error, w http.ResponseWriter) {
	// TODO: switch on err type
	w.WriteHeader(http.StatusInternalServerError)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(fmt.Sprintf("{\"error\": \"%s\"}", err.Error())))
}
