package fqdn

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/jinzhu/gorm"
	"github.com/zenazn/goji/web"
)

const (
	// Default hostname for the system
	DefaultHostname = "ncc1701"
	DefaultDomain   = ""

	// Endpoint at which the hostname can be configured
	EHostname = "/hostname"
	// Endpoint at which domain can be configured
	EDomain = "/domain"
)

type Controller struct {
	db  *gorm.DB
	mux *web.Mux

	commitChan      chan func() error
	commitErrorChan chan error
}

func NewController(db *gorm.DB,
	commitChan chan func() error,
	commitErrorChan chan error) *Controller {

	c := Controller{
		db:              db,
		mux:             web.New(),
		commitChan:      commitChan,
		commitErrorChan: commitErrorChan,
	}

	c.mux.Get(EHostname, c.GetHostname)
	c.mux.Put(EHostname, c.PutHostname)

	c.mux.Get(EDomain, c.GetDomain)
	c.mux.Put(EDomain, c.PutDomain)

	return &c
}

// ServeHTTP allows the controller to act as a http.Handler. It delegates to the internal mux.
func (c *Controller) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c.mux.ServeHTTP(w, r)
}

// MigrateDB performs DB migrations on the tables that this controller is responsible for.
func (c *Controller) MigrateDB() {
	c.db.AutoMigrate(&Hostname{})
	c.db.AutoMigrate(&Domain{})

	// Always ensure first entry exists
	c.db.FirstOrCreate(&Hostname{Hostname: DefaultHostname})
	c.db.FirstOrCreate(&Domain{Domain: DefaultDomain})
}

//
// Handlers
//

func (c *Controller) GetHostname(w http.ResponseWriter, r *http.Request) {
	host := Hostname{}
	err := c.db.First(&host, 1).Error
	if err != nil {
		c.jsonError(err, w)
		return
	}

	bytes, err := json.Marshal(&host)
	if err != nil {
		c.jsonError(err, w)
		return
	}

	_, err = w.Write(bytes)
	if err != nil {
		c.jsonError(err, w)
		return
	}

	return
}

func (c *Controller) GetDomain(w http.ResponseWriter, r *http.Request) {
	domain := Domain{}
	err := c.db.First(&domain, 1).Error
	if err != nil {
		c.jsonError(err, w)
		return
	}

	bytes, err := json.Marshal(&domain)
	if err != nil {
		c.jsonError(err, w)
		return
	}

	_, err = w.Write(bytes)
	if err != nil {
		c.jsonError(err, w)
		return
	}

	return
}

func (c *Controller) PutHostname(w http.ResponseWriter, r *http.Request) {
	host := Hostname{ID: 1}

	bodybytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		c.jsonError(err, w)
		return
	}

	err = json.Unmarshal(bodybytes, &host)
	if err != nil {
		c.jsonError(err, w)
		return
	}
	host.ID = 1

	err = c.db.Save(&host).Error
	if err != nil {
		err = fmt.Errorf("Failed to persist configuration (%s)", err)
		c.jsonError(err, w)
		return
	}

	c.commitChan <- c.AfterCommit
	err = <-c.commitErrorChan
	if err != nil {
		err = fmt.Errorf("Failed to generate configuration file (%s)", err)
		c.jsonError(err, w)
		return
	}

	w.Write(bodybytes)
}

func (c *Controller) PutDomain(w http.ResponseWriter, r *http.Request) {
	domain := Domain{ID: 1}

	bodybytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		c.jsonError(err, w)
		return
	}

	err = json.Unmarshal(bodybytes, &domain)
	if err != nil {
		c.jsonError(err, w)
		return
	}
	domain.ID = 1

	err = c.db.Save(&domain).Error
	if err != nil {
		err = fmt.Errorf("Failed to persist configuration (%s)", err)
		c.jsonError(err, w)
		return
	}

	w.Write(bodybytes)
}

//
// AfterCommit
//

func (c *Controller) AfterCommit() error {
	contents, err := c.hostnameFileContents()
	if err != nil {
		return err
	}

	err = ioutil.WriteFile("/etc/hostname", contents, 0644)
	if err != nil {
		return err
	}

	return nil
}

//
// Helpers
//

func (c *Controller) hostnameFileContents() ([]byte, error) {
	host := Hostname{}

	err := c.db.First(&host, 1).Error
	if err != nil {
		return nil, err
	}

	return []byte(host.Hostname + "\n"), nil
}

func (c *Controller) jsonError(err error, w http.ResponseWriter) {
	// TODO: switch on err type
	w.WriteHeader(http.StatusInternalServerError)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(fmt.Sprintf("{\"error\": \"%s\"}", err.Error())))
}
