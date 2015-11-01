package crashcorder

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"

	"rocketship/radio"

	"github.com/amoghe/distillog"
	"golang.org/x/exp/inotify"
)

const (
	KernelCorePatternFilePath = "/proc/sys/kernel/core_pattern"

	NotificationSubject = "Unexpected process crash"

	PatternDelimiter = "_"
)

var (
	PatternTokenToString = map[string]string{
		"%e": "Executable",
		"%p": "PID",
		"%u": "UID",
		"%g": "GID",
		"%s": "Reason",
		"%t": "Time",
	}
)

// Config holds the configuration for the crashcorder
type Config struct {
	CorePatternTokens []string
	CoresDirectory    string
	RadioConnectAddr  net.TCPAddr
}

// Crashcorder holds all the state for an instance of the crash detector.
type Crashcorder struct {
	Config Config
	Logger distillog.Logger

	stopChan chan bool
	doneChan chan bool
	watcher  *inotify.Watcher
}

func New(cfg Config, log distillog.Logger) *Crashcorder {
	return &Crashcorder{
		Config: cfg,
		Logger: log,

		stopChan: make(chan bool, 1),
		doneChan: make(chan bool, 1),
	}
}

// Start initializes the core file watcher and starts watching for core notifications.
func (c *Crashcorder) Start() error {
	if c.stopChan == nil {
		return fmt.Errorf("No stopChan initialized")
	}

	if c.watcher != nil {
		err := c.watcher.Close()
		if err != nil {
			c.Logger.Infoln("Failed to close existing watcher", err)
		}
	}

	watcher, err := inotify.NewWatcher()
	if err != nil {
		return err
	}

	c.watcher = watcher

	c.Logger.Infoln("Watching dir", c.Config.CoresDirectory)
	err = watcher.AddWatch(c.Config.CoresDirectory, inotify.IN_CREATE)
	if err != nil {
		return err
	}

	// start monitoring the dir before we configure the kernel to write cores there
	// so that we don't miss any notifications
	go c.watchForCreates()

	// c.configureKernelCorePattern()

	return nil
}

// Stop initiates the shutdown of the crashcorder watcher. Optionally it blocks
// till it is fully stopped.
func (c *Crashcorder) Stop(wait bool) {
	c.Logger.Infoln("Stopping watch routine")
	c.stopChan <- true

	if wait {
		c.Wait()
	}

	c.Logger.Infoln("Stopped crashcorder")
}

// Wait blocks till the crashcorder is fully stopped
func (c *Crashcorder) Wait() {
	c.Logger.Infoln("Blocking till watch routine has stopped")
	<-c.doneChan
}

//
//  Helpers
//

func (c *Crashcorder) configureKernelCorePattern() error {
	f, err := os.Open(KernelCorePatternFilePath)
	if err != nil {
		return err
	}
	defer f.Close()

	f.Write([]byte(strings.Join(c.Config.CorePatternTokens, PatternDelimiter)))
	return nil
}

func (c *Crashcorder) watchForCreates() {
	c.Logger.Debugln("Starting watch for dir activity")

loop:
	for {
		select {
		case <-c.stopChan:
			c.Logger.Infoln("Watch routine requested to stop")
			break loop
		case event := <-c.watcher.Event:
			c.Logger.Infoln("Received event for file", event.Name)
			if err := c.handleCoreFile(event.Name); err != nil {
				c.Logger.Infoln("Error handling core file:", err)
			}
		case error := <-c.watcher.Error:
			c.Logger.Infoln("Watcher error:", error)
			// possibly restart?
		}
	}

	c.doneChan <- true
	c.Logger.Infoln("Stopped watch for dir activity")
}

func (c *Crashcorder) handleCoreFile(name string) error {
	c.Logger.Infoln("Handling core file", name)
	coreinfo, err := c.extractCoreFileInfo(name)
	if err != nil {
		return err
	}

	mailbody, err := json.Marshal(coreinfo)
	if err != nil {
		return err
	}

	err = c.sendRadioMessage(NotificationSubject, string(mailbody))
	if err != nil {
		return err
	}

	return nil
}

func (c *Crashcorder) extractCoreFileInfo(name string) (coreInfo map[string]string, err error) {
	toks := strings.Split(name, PatternDelimiter)

	if len(toks) != len(c.Config.CorePatternTokens) {
		err = fmt.Errorf("Unexpected number of tokens in core file")
		return
	}

	coreInfo = make(map[string]string)
	for i, str := range c.Config.CorePatternTokens {
		reason, ok := PatternTokenToString[str]
		if !ok {
			c.Logger.Warningln("Unknown token in core pattern", str)
		}
		coreInfo[reason] = toks[i]
	}

	return
}

func (c *Crashcorder) sendRadioMessage(subj string, body string) error {
	msg := radio.MessageRequest{
		Severity: radio.LevelWarn,
		Subject:  subj,
		Body:     body,
	}

	msgjson, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	resp, err := http.Post(
		"http://"+c.Config.RadioConnectAddr.String()+radio.EmailEndpoint,
		"application/json",
		bytes.NewBuffer(msgjson))
	if err != nil {
		return err
	}
	resp.Body.Close()

	return nil
}
