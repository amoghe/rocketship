package ssh

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"text/template"
	"time"

	"rocketship/regulog"

	"github.com/jinzhu/gorm"
	"github.com/zenazn/goji/web"
)

const (
	SshConfigDirPath = "/etc/ssh"

	SshConfigFileName         = "ssh_config"
	SshKeyRegenMarkerFileName = ".commander_regenerated_keys"

	// Path to files of interest in the ssh dir
	SshConfigFilePath         = SshConfigDirPath + "/" + SshConfigFileName
	SshKeyRegenMarkerFilePath = SshConfigDirPath + "/" + SshKeyRegenMarkerFileName

	// Prefix under which this controller registers endpoints
	URLPrefix = "/ssh"

	// ESshConfig is the endpoint at which configuration for SSH is accessed
	ESshConfig = URLPrefix + "/config"
)

type Controller struct {
	db  *gorm.DB
	mux *web.Mux
	log regulog.Logger
}

func NewController(db *gorm.DB, logger regulog.Logger) *Controller {
	c := Controller{
		mux: web.New(),
		db:  db,
		log: logger,
	}

	c.mux.Get(ESshConfig, c.GetSshConfig)
	c.mux.Put(ESshConfig, c.PutSshConfig)

	return &c
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

func (c *Controller) RewriteFiles() error {
	c.log.Infoln("Rewriting SSH configuration files")
	// helper func to rewrite ssh_config file
	regenerateSshConfigFile := func() error {
		contents, err := c.sshConfigFileContents()
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(SshConfigFilePath, contents, 0644)
		if err != nil {
			return err
		}
		return nil
	}

	// helper func to regenerate host SSH keys
	regenerateHostKeysOnce := func() error {
		if _, err := os.Stat(SshKeyRegenMarkerFilePath); err == nil {
			// Marker exists, keys have been regenerated once before
			return nil
		}

		c.log.Infoln("Regenerating host SSH keys")
		cmd := &exec.Cmd{
			Path: "/usr/bin/ssh-keygen",
			Args: []string{"-A"},
			Dir:  SshConfigDirPath,
		}
		if err := cmd.Run(); err != nil {
			err_prefix := "failed to regenerate SSH host keys"
			switch err.(type) {
			case *exec.ExitError:
				out, _ := cmd.CombinedOutput()
				return fmt.Errorf("%s: %s", err_prefix, out)
			default:
				return fmt.Errorf("%s: %s", err_prefix, err)
			}
		}

		// Touch the marker file
		ioutil.WriteFile(SshKeyRegenMarkerFilePath, []byte(time.Now().String()), 0664)
		return nil
	}

	err1 := regenerateSshConfigFile()
	err2 := regenerateHostKeysOnce()

	if err1 != nil && err2 != nil {
		return fmt.Errorf("errors regenerating ssh config: %s & error regenerating keys: %s", err1, err2)
	} else if err1 != nil {
		return fmt.Errorf("error regenerating ssh config: %s", err1)
	} else if err2 != nil {
		return fmt.Errorf("error regenerating ssh keys: %s", err2)
	}

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
	c.log.Infoln("Migrating SSH tables")
	c.db.AutoMigrate(&SshConfig{AllowPasswordAuth: true, AllowPubkeyAuth: true})
}

func (c *Controller) SeedDB() {
	c.log.Infoln("Seeding SSH tables")
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
