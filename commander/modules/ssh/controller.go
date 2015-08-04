package ssh

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"text/template"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/zenazn/goji/web"
)

type Controller struct {
	db  *gorm.DB
	mux *web.Mux
}

func NewController(db *gorm.DB) *Controller {
	c := Controller{
		mux: web.New(),
		db:  db,
	}

	c.mux.Get("/ssh_config", c.GetSshConfig)
	c.mux.Put("/ssh_config", c.PutSshConfig)

	return &c
}

//
// Handlers
//

func (c Controller) GetSshConfig(_ web.C, w http.ResponseWriter, r *http.Request) {
	cfg := SshConfig{}
	err := c.db.First(&cfg).Error
	if err != nil {
		c.jsonError(err, w)
		return
	}

	enc := json.NewEncoder(w)
	if err := enc.Encode(cfg); err != nil {
		c.jsonError(err, w)
		return
	}

	return
}

func (c Controller) PutSshConfig(_ web.C, w http.ResponseWriter, r *http.Request) {
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		c.jsonError(err, w)
		return
	}

	resource := SshConfigResource{}
	if err = json.Unmarshal(reqBody, &resource); err != nil {
		c.jsonError(err, w)
		return
	}

	model := resource.ToSshConfigModel()
	if err = c.db.Save(&model).Error; err != nil {
		c.jsonError(err, w)
		return
	}

	w.Write(reqBody)
}

func (c *Controller) jsonError(err error, w http.ResponseWriter) {
	// TODO: switch on err type
	w.WriteHeader(http.StatusInternalServerError)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(fmt.Sprintf("{\"error\": \"%s\"}", err.Error())))

}

//
// File generators
//

func (c *Controller) RewriteSshConfigFile() error {
	return nil
}

func (c *Controller) sshConfigFileContents() ([]byte, error) {
	type templateData struct {
		GenTime           string
		AllowPasswordAuth string
		AllowPubkeyAuth   string
	}

	sshCfg := SshConfig{}
	err := c.db.First(&sshCfg).Error
	if err != nil {
		return []byte{}, fmt.Errorf("unable to fetch ssh config from db: %v", err)
	}

	tdata := templateData{time.Now().String(), "no", "no"}
	if sshCfg.AllowPasswordAuth {
		tdata.AllowPasswordAuth = "yes"
	}
	if sshCfg.AllowPubkeyAuth {
		tdata.AllowPubkeyAuth = "yes"
	}

	tmpl, err := template.New("sshd.conf").Parse(templateStr)
	if err != nil {
		return []byte{}, err
	}

	retbuf := &bytes.Buffer{}
	err = tmpl.Execute(retbuf, tdata)
	if err != nil {
		return []byte{}, err
	}

	return retbuf.Bytes(), nil
}

//
// DB Models
//

type SshConfig struct {
	ID                int
	AllowPasswordAuth bool
	AllowPubkeyAuth   bool
}

//
// Seed
//

func (c *Controller) MigrateDB() {
	c.db.AutoMigrate(&SshConfig{AllowPasswordAuth: true, AllowPubkeyAuth: true})
}

func (c *Controller) SeedSshdConfig() {
	c.db.FirstOrCreate(&SshConfig{ID: 1, AllowPasswordAuth: true})
}

//
// Resources
//

type SshConfigResource struct {
	AllowPasswordAuth bool
	AllowPubkeyAuth   bool
}

func (s SshConfigResource) ToSshConfigModel() SshConfig {
	return SshConfig{
		ID:                1,
		AllowPasswordAuth: s.AllowPasswordAuth,
		AllowPubkeyAuth:   s.AllowPubkeyAuth,
	}
}

func (s *SshConfigResource) FromSshConfigModel(m SshConfig) {
	s.AllowPasswordAuth = m.AllowPasswordAuth
	s.AllowPubkeyAuth = m.AllowPubkeyAuth
}
