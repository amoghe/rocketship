package syslog

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"text/template"
	"time"

	"github.com/amoghe/distillog"
	"github.com/jinzhu/gorm"
	"github.com/zenazn/goji/web"
)

const (
	// Prefix under which API endpoints are rooted
	URLPrefix = "/syslog"
)

type Controller struct {
	db  *gorm.DB
	mux *web.Mux
	log distillog.Logger
}

func NewController(db *gorm.DB, log distillog.Logger) *Controller {
	// TODO: endpoints to en/disable the syslog daemon.
	return &Controller{db: db, mux: web.New(), log: log}
}

// ServeHTTP satisfies the http.Handler interface (net/http as well as goji)
func (c *Controller) ServeHTTPC(ctx web.C, w http.ResponseWriter, r *http.Request) {
	c.mux.ServeHTTP(w, r)
}

// RoutePrefix returns the prefix under which this router handles endpoints
func (c *Controller) RoutePrefix() string {
	return URLPrefix
}

//
// File generators
//

func (c *Controller) RewriteFiles() error {
	c.log.Infoln("Rewriting syslog configuration files")
	contents, err := c.syslogConfFileContents()
	if err != nil {
		c.log.Errorln("Failed to generate syslog config:", err)
		return err
	}

	err = ioutil.WriteFile("/etc/rsyslog.conf", contents, 0644)
	if err != nil {
		c.log.Errorln("Failed to write syslog conf file:", err)
		return err
	}

	return nil
}

func (c *Controller) syslogConfFileContents() ([]byte, error) {

	type _templateData struct {
		GenTime   string
		Username  string
		Groupname string
	}

	tmpl, err := template.New("syslog.conf").Parse(templateStr)
	if err != nil {
		return []byte{}, err
	}

	syslogData := _templateData{
		GenTime:   time.Now().String(),
		Username:  "syslog",
		Groupname: "syslog",
	}

	retbuf := &bytes.Buffer{}
	err = tmpl.Execute(retbuf, syslogData)
	if err != nil {
		return []byte{}, err
	}

	return retbuf.Bytes(), nil
}

//
// DB state
// TODO: tables to track whether syslog is enabled
//

func (c *Controller) MigrateDB() {}
func (c *Controller) SeedDB()    {}
