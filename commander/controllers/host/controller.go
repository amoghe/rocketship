package host

import (
	"fmt"
	"net/http"

	"github.com/jinzhu/gorm"
	"github.com/zenazn/goji/web"
)

const (
	// Endpoint at which the hostname can be configured
	EHostname = "/hostname"
	// Endpoint at which domain can be configured
	EDomain = "/domain"

	InterfacesFilePath   = "/etc/network/interfaces"
	DhclientConfFilePath = "/etc/dhcp/dhclient.conf"
)

type Controller struct {
	db  *gorm.DB
	mux *web.Mux

	commitChan      chan func() error
	commitErrorChan chan error
}

func NewController(db *gorm.DB) *Controller {

	c := Controller{
		db:  db,
		mux: web.New(),
	}

	c.mux.Get(EHostname, c.GetHostname)
	c.mux.Put(EHostname, c.PutHostname)

	c.mux.Get(EDomain, c.GetDomain)
	c.mux.Put(EDomain, c.PutDomain)

	return &c
}

func (c *Controller) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c.mux.ServeHTTP(w, r)
}

func (c *Controller) MigrateDB() {
	var (
		profile = DHCPProfile{ID: 1}
		iface   = InterfaceConfig{Name: "eth0", Mode: ModeDHCP, DHCPProfileID: 1}
	)

	c.db.AutoMigrate(&Hostname{})
	c.db.AutoMigrate(&Domain{})
	c.db.AutoMigrate(&DHCPProfile{})
	c.db.AutoMigrate(&InterfaceConfig{})
	c.db.AutoMigrate(&User{})

	// Always ensure first entry exists
	c.db.FirstOrCreate(&Hostname{Hostname: DefaultHostname})
	c.db.FirstOrCreate(&Domain{Domain: DefaultDomain})
	c.db.FirstOrCreate(&profile, profile)
	c.db.FirstOrCreate(&iface, iface)
}

//
// AfterCommit
//

func (c *Controller) AfterCommit() error {
	return nil
}

//
// Helpers
//

func (c *Controller) jsonError(err error, w http.ResponseWriter) {
	// TODO: switch on err type
	w.WriteHeader(http.StatusInternalServerError)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(fmt.Sprintf("{\"error\": \"%s\"}", err.Error())))
}

func chunkSlice(s []string, chunkSize int) (ret [][]string) {
	if chunkSize >= len(s) {
		return append(ret, s)
	}

	start := 0
	end := 0

	for {
		start = end
		end = start + chunkSize

		if end >= len(s) {
			ret = append(ret, s[start:])
			return
		}

		ret = append(ret, s[start:end])
	}

	return
}
