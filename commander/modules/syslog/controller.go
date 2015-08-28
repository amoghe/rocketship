package syslog

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"text/template"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/zenazn/goji/web"

	"rocketship/commander/modules/host"
)

const (
	// Prefix under which API endpoints are rooted
	URLPrefix = "/syslog"
)

type Controller struct {
	db  *gorm.DB
	mux *web.Mux
}

func NewController(db *gorm.DB) *Controller {
	return &Controller{db: db, mux: web.New()}
}

// ServeHTTP satisfies the http.Handler interface (net/http as well as goji)
func (c *Controller) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c.mux.ServeHTTP(w, r)
}

// RoutePrefix returns the prefix under which this router handles endpoints
func (c *Controller) RoutePrefix() string {
	return URLPrefix
}

//
// File generators
//

func (c *Controller) RewriteSyslogConfFile() error {
	contents, err := c.syslogConfFileContents()
	if err != nil {
		return err
	}

	err = ioutil.WriteFile("/etc/rsyslog.conf", contents, 0644)
	if err != nil {
		return err
	}

	return nil
}

func (c *Controller) syslogConfFileContents() ([]byte, error) {

	type _templateData struct {
		GenTime string
		Uid     uint32
		Gid     uint32
	}

	tmpl, err := template.New("syslog.conf").Parse(templateStr)
	if err != nil {
		return []byte{}, err
	}

	syslogUser, err := host.GetSystemUser("syslog")
	if err != nil {
		return []byte{}, fmt.Errorf("Unable to lookup details for syslog user")
	}

	syslogData := _templateData{
		GenTime: time.Now().String(),
		Uid:     syslogUser.Uid,
		Gid:     syslogUser.Gid,
	}

	retbuf := &bytes.Buffer{}
	err = tmpl.Execute(retbuf, syslogData)
	if err != nil {
		return []byte{}, err
	}

	return retbuf.Bytes(), nil
}
