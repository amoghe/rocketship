package crashcorder

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"rocketship/radio"

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

type Crashcorder struct {
	CoresDirectory    string
	CorePatternTokens []string
	RadioConnectAddr  net.TCPAddr
	Logger            *log.Logger

	stopChan chan bool
	doneChan chan bool
	watcher  *inotify.Watcher
}

func New(coredir string, coretoks []string, connaddr net.TCPAddr, log *log.Logger) *Crashcorder {
	return &Crashcorder{
		CorePatternTokens: coretoks,
		CoresDirectory:    coredir,
		RadioConnectAddr:  connaddr,
		Logger:            log,

		stopChan: make(chan bool),
		doneChan: make(chan bool),
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
			c.Logger.Println("Failed to close existing watcher", err)
		}
	}

	watcher, err := inotify.NewWatcher()
	if err != nil {
		return err
	}

	c.watcher = watcher

	c.Logger.Println("Watching dir", c.CoresDirectory)
	err = watcher.AddWatch(c.CoresDirectory, inotify.IN_CREATE)
	if err != nil {
		return err
	}

	// start monitoring the dir before we configure the kernel to write cores there
	// so that we don't miss any notifications
	go c.watchForCreates()

	f, err := os.Open(KernelCorePatternFilePath)
	if err != nil {
		return err
	}

	f.Write([]byte(strings.Join(c.CorePatternTokens, PatternDelimiter)))
	f.Close()

	return nil
}

// Stop initiates the shutdown of the crashcorder watcher. Optionally it blocks
// till it is fully stopped.
func (c *Crashcorder) Stop(wait bool) {
	c.stopChan <- true

	if wait {
		c.Wait()
	}
}

// Wait will block till the crashcorder is fully stopped
func (c *Crashcorder) Wait() {
	<-c.doneChan
}

//
//  Helpers
//

func (c *Crashcorder) watchForCreates() {
	c.Logger.Println("Starting watch for dir activity")
	select {
	case event := <-c.watcher.Event:
		c.Logger.Println("Received event for file", event.Name)
		if err := c.handleCoreFile(event.Name); err != nil {
			c.Logger.Println("Error handling core file:", err)
		}
	case error := <-c.watcher.Error:
		c.Logger.Println("Watcher error:", error)
		// possibly restart?
	case <-c.stopChan:
		c.Logger.Println("Stopping watch routine")
	}

	c.doneChan <- true
	c.Logger.Println("Stopped watch for dir activity")
}

func (c *Crashcorder) handleCoreFile(name string) error {
	c.Logger.Println("Handling core file", name)
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

	if len(toks) != len(c.CorePatternTokens) {
		err = fmt.Errorf("Unexpected number of tokens in core file")
		return
	}

	coreInfo = make(map[string]string)
	for i, str := range c.CorePatternTokens {
		reason, ok := PatternTokenToString[str]
		if !ok {
			fmt.Println("Unknown token in core pattern", str)
		}
		coreInfo[reason] = toks[i]
	}

	return
}

func (c *Crashcorder) sendRadioMessage(subj string, body string) error {
	msg := radio.MessageRequest{
		Severity: "WARN",
		Subject:  subj,
		Body:     body,
	}

	msgjson, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	resp, err := http.Post(
		"http://"+c.RadioConnectAddr.String()+"/notify",
		"application/json",
		bytes.NewBuffer(msgjson))
	if err != nil {
		return err
	}
	resp.Body.Close()

	return nil
}
