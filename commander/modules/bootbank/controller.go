package bootbank

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/amoghe/distillog"
	"github.com/jinzhu/gorm"
	"github.com/juju/deputy"
	"github.com/zenazn/goji/web"
)

const (
	URLPrefix = "/boot"

	KernelCommandlineFile = "/proc/cmdline"

	Bootbank1 = "BOOTBANK1"
	Bootbank2 = "BOOTBANK2"

	GrubPartitionlabel = "GRUB"

	ImageVersionFile = "/etc/rocketship_version"

	EBootbanks  = URLPrefix + "/banks"
	EBootbankID = EBootbanks + "/:id"
)

type Controller struct {
	db   *gorm.DB
	mux  *web.Mux
	log  distillog.Logger
	lock sync.Mutex
}

func NewController(db *gorm.DB, logger distillog.Logger) *Controller {
	ctrl := &Controller{db: db, mux: web.New(), log: logger}

	ctrl.mux.Get(EBootbanks, ctrl.GetBootbanks)
	ctrl.mux.Get(EBootbankID, ctrl.GetBootbankDetails)
	ctrl.mux.Put(EBootbankID+"/image", ctrl.UploadImageFile)
	ctrl.mux.Put(EBootbankID+"/bootable", ctrl.MarkBootable)

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

type BootbankDetails struct {
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
		ret     BootbankDetails
		version string
		err     error
	)

	readVersionFileFromDir := func(dir string) error {
		if vbytes, err := ioutil.ReadFile(dir + "/" + ImageVersionFile); err == nil {
			version = string(vbytes[0 : len(vbytes)-1]) // leave out trailing newline
			return nil
		} else {
			if perr, ok := err.(*os.PathError); ok {
				if perr.Err == syscall.ENOENT {
					return fmt.Errorf("Unavailable - no image installed")
				} else {
					return fmt.Errorf("Unavailable - %s (%T)", perr.Error(), perr.Err)
				}
			} else {
				return fmt.Errorf("Error reading version info: %s", err)
			}
		}
	}

	if bbLabel == c.currentBootbankLabel() {
		err = readVersionFileFromDir("/")
		ret = BootbankDetails{Version: version, Active: true}
	} else {
		err = withMountedPartition(c.otherBootbankLabel(), true, c.log, readVersionFileFromDir)
		ret = BootbankDetails{Version: version, Active: false}
	}
	if err != nil {
		jsonError(fmt.Errorf("Failed to read version file. %s", err), w)
		return
	}
	if err := json.NewEncoder(w).Encode(ret); err != nil {
		jsonError(err, w)
		return
	}
	return
}

func (c *Controller) UploadImageFile(ctx web.C, w http.ResponseWriter, r *http.Request) {
	if ctx.URLParams["id"] == c.currentBootbankLabel() {
		jsonError(fmt.Errorf("Cannot upload image into currently booted bank"), w)
		return
	}

	file, _, err := r.FormFile("image")
	if err != nil {
		jsonError(err, w)
		return
	}

	err = c.loadImageStreamIntoBootbank(c.otherBootbankLabel(), file)
	if err != nil {
		jsonError(err, w)
		return
	}

	w.WriteHeader(http.StatusOK)
	return
}

func (c *Controller) MarkBootable(ctx web.C, w http.ResponseWriter, r *http.Request) {
	bbLabel := ctx.URLParams["id"]
	if bbLabel != Bootbank1 && bbLabel != Bootbank2 {
		jsonError(fmt.Errorf("Invalid bootbank (%s) specified", bbLabel), w)
		return
	}

	c.log.Infof("Marking bootbank %s as bootable (for next boot)", bbLabel)
	if err := c.makeBootbankBootable(bbLabel); err != nil {
		jsonError(fmt.Errorf("Unable to mark %s bootable: %s", bbLabel, err), w)
		return
	}

	w.WriteHeader(http.StatusOK)
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
		if strings.HasPrefix(token, "root=LABEL") { // root=LABEL=BOOTBANK1
			t := strings.Split(token, "=")
			if len(t) != 3 {
				c.log.Warningf("Cannot infer root label (token: %s). Assuming %s", token, Bootbank1)
				return Bootbank1
			}
			return t[2]
		}
	}

	c.log.Infof("Unable to determine bootbank (no LABEL found on cmdline: %s). Assuming %s", string(b), Bootbank1)
	return Bootbank1
}

func (c *Controller) otherBootbankLabel() string {
	if c.currentBootbankLabel() == Bootbank1 {
		return Bootbank2
	} else {
		return Bootbank1
	}
}

func (c *Controller) loadImageStreamIntoBootbank(banklabel string, stream io.Reader) error {

	unpackImageIntoDir := func(dirname string) error {
		cmd := exec.Command("tar",
			"--gunzip",
			"--extract",
			"--file=-",               // read from the file we put on its stdin
			"--preserve-permissions", // Our job is only to load dir
			"--numeric-owner",        // Our job is only to load dir
			"-C",
			dirname,
			".",
		)

		cmd.Stdin = stream

		c.log.Infoln("Unpacking image into bootbank", banklabel)
		c.log.Debugln("Running: ", strings.Join(cmd.Args, " "))
		if output, err := cmd.CombinedOutput(); err != nil {
			c.log.Errorf("Failed to unpack uploaded system image (%T):%s", err, err)
			c.log.Errorln("Combined stdout/stderr output follows:")
			for _, line := range strings.Split(string(output), "\n") {
				//c.log.Errorln(line)
				_ = line
			}
			return err
		}

		c.log.Infoln("Unpack completed succesfully")
		return nil
	}

	if banklabel == c.currentBootbankLabel() {
		return fmt.Errorf("Bootbank is currently active")
	}

	return withMountedPartition(banklabel, false, c.log, unpackImageIntoDir)
}

func (c *Controller) loadImageFileIntoBootbank(banklabel string, imgFilePath string) error {

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

	if err := withMountedPartition(banklabel, false, c.log, unpackImageIntoDir); err != nil {
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

		err := os.MkdirAll(grubDir, 0755)
		if err != nil {
			return fmt.Errorf("Failed to ensure grub dir: %s", err)
		}

		fileContents, err := c.grubConfContents(banklabel)
		if err != nil {
			return fmt.Errorf("Failed to generate grub.conf contents: %s", err)
		}

		err = ioutil.WriteFile(grubFilePath, []byte(fileContents), 0444)
		if err != nil {
			return fmt.Errorf("Failed to write grub config: %s", err)
		}

		return nil
	}

	return withMountedPartition(GrubPartitionlabel, false, c.log, writeGrubFile)
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

func withMountedPartition(partitionLabel string, readonly bool, log distillog.Logger, f func(string) error) error {
	tempDir, err := ioutil.TempDir(os.TempDir(), "tempMountedPartition")
	if err != nil {
		return err
	}

	flags := 0
	if readonly {
		flags |= syscall.MS_RDONLY
	}

	partitionPath := "/dev/disk/by-label/" + partitionLabel
	log.Debugf("Mounting %s on %s (readonly: %t) (flags: %X)", partitionPath, tempDir, readonly, uintptr(flags))
	if err := syscall.Mount(partitionPath, tempDir, "ext4", uintptr(flags), ""); err != nil {
		log.Errorln("mount syscall failed:", err)
		return err
	}
	defer func() {
		log.Debugf("Unmounting %s (from %s)", partitionPath, tempDir)
		syscall.Unmount(tempDir, 0)
		os.RemoveAll(tempDir)
	}()

	return f(tempDir)
}

func jsonError(err error, w http.ResponseWriter) {
	// TODO: switch on err type
	w.WriteHeader(http.StatusInternalServerError)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(fmt.Sprintf("{\"error\": \"%s\"}", err.Error())))
}
