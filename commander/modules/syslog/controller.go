package syslog

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"text/template"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/zenazn/goji/web"

	"rocketship/commander/modules/host"
)

type Controller struct {
	db  *gorm.DB
	mux *web.Mux
}

func NewController(db *gorm.DB) *Controller {
	return &Controller{db: db, mux: web.New()}
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
