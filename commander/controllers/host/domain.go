package host

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

var (
	DefaultDomain = ""
)

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
// DB Models
//

type Domain struct {
	ID     int64 `json:"-"`
	Domain string
}
