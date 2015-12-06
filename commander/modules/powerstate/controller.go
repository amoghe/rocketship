package powerstate

import (
	"distillog"
	"fmt"
	"net/http"
	"os/exec"
	"sync"

	"github.com/jinzhu/gorm"
	"github.com/zenazn/goji/web"
)

const (
	URLPrefix = "/powerstate"
	EReboot   = URLPrefix + "/reboot"
	EShutdown = URLPrefix + "/shutdown"
)

type Controller struct {
	mux  *web.Mux
	log  distillog.Logger
	lock sync.Mutex
}

func NewController(db *gorm.DB, logger distillog.Logger) *Controller {
	ctrl := &Controller{mux: web.New(), log: logger}

	ctrl.mux.Put(EReboot, ctrl.DoReboot)
	ctrl.mux.Put(EShutdown, ctrl.DoShutdown)

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
// Handlers
//

func (c *Controller) DoReboot(ctx web.C, w http.ResponseWriter, r *http.Request) {
	cmd := exec.Command("shutdown", "-r", "now", "user initiated reboot")
	if err := cmd.Start(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("%s", err.Error())))
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (c *Controller) DoShutdown(ctx web.C, w http.ResponseWriter, r *http.Request) {
	cmd := exec.Command("shutdown", "-h", "now", "user initiated shutdown (halt)")
	if err := cmd.Start(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("%s", err.Error())))
		return
	}
	w.WriteHeader(http.StatusOK)
}
