package host

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/jinzhu/gorm"
)

const (
	// Minimum length of hostname string
	MinHostnameLength = 1

	// Default hostname for the system
	DefaultHostname = "ncc1701"
)

//
// Endpoint handlers
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

	w.Write(bodybytes)
}

//
// File generators
//

// RewriteHostnameFile rewrites the hostname file.
func (c *Controller) RewriteHostnameFile() error {
	c.log.Infoln("Rewriting hostname file")

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

// returns contents of the hostname file.
func (c *Controller) hostnameFileContents() ([]byte, error) {
	host := Hostname{}

	err := c.db.First(&host, 1).Error
	if err != nil {
		return nil, err
	}

	return []byte(host.Hostname + "\n"), nil
}

//
// DB Models
//

type Hostname struct {
	ID       int64 `json:"-"`
	Hostname string
}

func (h *Hostname) BeforeSave(txn *gorm.DB) error {
	// Length check
	if len(h.Hostname) < MinHostnameLength {
		return fmt.Errorf("Hostname cannot be shorter than %d chars", MinHostnameLength)
	}
	// Invalid chars check
	for _, char := range []string{" ", ".", "/"} {
		if strings.Contains(h.Hostname, char) {
			return fmt.Errorf("Hostname cannot contain %s", char)
		}
	}
	return nil
}

//
// Resource
//

type HostnameResource struct {
	Hostname string
}

func (h HostnameResource) ToHostnameModel() Hostname {
	return Hostname{Hostname: h.Hostname}
}

func (h *HostnameResource) FromHostnameModel(m Hostname) {
	h.Hostname = m.Hostname
}

//
// DB Seed
//

func (c *Controller) seedHostname() {
	c.log.Infoln("Seeding hostname")
	c.db.FirstOrCreate(&Hostname{Hostname: DefaultHostname})
}
