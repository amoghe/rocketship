package radio

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/mail"
	"os"
	"strconv"

	"github.com/jinzhu/gorm"
	"github.com/zenazn/goji/web"

	"rocketship/commander/modules/host"
	"rocketship/radio"
)

const (
	RadioPort = 12345

	RadioConfDir  = "/etc/radio"
	RadioConfFile = RadioConfDir + "/radio.conf"
)

type Controller struct {
	db  *gorm.DB
	mux *web.Mux
}

func NewController(db *gorm.DB) *Controller {
	return &Controller{db: db, mux: web.New()}
}

//
// HTTP Handlers
//

func (c *Controller) GetRadioConfig(_ web.C, w http.ResponseWriter, r *http.Request) {
	cfg := RadioConfig{}
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

func (c *Controller) UpdateRadioConfig(_ web.C, w http.ResponseWriter, r *http.Request) {
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		c.jsonError(err, w)
		return
	}

	resource := RadioConfigResource{}
	if err = json.Unmarshal(reqBody, &resource); err != nil {
		c.jsonError(err, w)
		return
	}

	if err = c.db.Save(RadioConfig(resource)).Error; err != nil {
		c.jsonError(err, w)
		return
	}

	w.Write(reqBody)
}

// Get
func (c *Controller) GetInfoRecipients(_ web.C, w http.ResponseWriter, r *http.Request) {
	c.getRecipients(w, r, &[]InfoRecipient{})
}
func (c *Controller) GetWarnRecipients(_ web.C, w http.ResponseWriter, r *http.Request) {
	c.getRecipients(w, r, &[]WarnRecipient{})
}
func (c *Controller) GetErrorRecipients(_ web.C, w http.ResponseWriter, r *http.Request) {
	c.getRecipients(w, r, &[]ErrorRecipient{})
}

// Add
func (c *Controller) AddInfoRecipient(_ web.C, w http.ResponseWriter, r *http.Request) {
	c.addRecipient(w, r, InfoRecipient{})
}
func (c *Controller) AddWarnRecipient(_ web.C, w http.ResponseWriter, r *http.Request) {
	c.addRecipient(w, r, WarnRecipient{})
}
func (c *Controller) AddErrorRecipient(_ web.C, w http.ResponseWriter, r *http.Request) {
	c.addRecipient(w, r, ErrorRecipient{})
}

// Del
func (c *Controller) DeleteInfoRecipient(ctx web.C, w http.ResponseWriter, r *http.Request) {
	c.deleteRecipient(ctx, w, r, &InfoRecipient{})
}
func (c *Controller) DeleteWarnRecipient(ctx web.C, w http.ResponseWriter, r *http.Request) {
	c.deleteRecipient(ctx, w, r, &WarnRecipient{})
}
func (c *Controller) DeleteErrorRecipient(ctx web.C, w http.ResponseWriter, r *http.Request) {
	c.deleteRecipient(ctx, w, r, &ErrorRecipient{})
}

// respond with a list of the requested type of email recipients. Type is indicated via the 'er'
// parameter which should be a slice of the appropriate struct/model.
func (c *Controller) getRecipients(w http.ResponseWriter, r *http.Request, er interface{}) {
	if err := c.db.Find(er).Error; err != nil {
		c.jsonError(err, w)
		return
	}

	enc := json.NewEncoder(w)
	if err := enc.Encode(er); err != nil {
		c.jsonError(err, w)
		return
	}

	return
}

// add an email recipient to the specified table (specified via the er param, which should be a
// pointer to an empty model struct). Also respond with the created recipient.
func (c *Controller) addRecipient(w http.ResponseWriter, r *http.Request, er interface{}) {
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		c.jsonError(err, w)
		return
	}

	resource := EmailRecipient{}
	if err = json.Unmarshal(reqBody, &resource); err != nil {
		c.jsonError(err, w)
		return
	}

	var recp interface{}
	switch er.(type) {
	case InfoRecipient:
		recp = &InfoRecipient{Email: resource.Email}
	case WarnRecipient:
		recp = &WarnRecipient{Email: resource.Email}
	case ErrorRecipient:
		recp = &ErrorRecipient{Email: resource.Email}
	default:
		err = fmt.Errorf("cannot save unsupported recipient type")
	}

	if err := c.db.Create(recp).Error; err != nil {
		c.jsonError(err, w)
		return
	}

	enc := json.NewEncoder(w)
	if err := enc.Encode(recp); err != nil {
		c.jsonError(err, w)
		return
	}

	return
}

// add an email recipient to the specified table (specified via the er param, which should be a
// pointer to an empty model struct). Write the deleted recipient in the response body.
func (c *Controller) deleteRecipient(ctx web.C, w http.ResponseWriter, r *http.Request, er interface{}) {

	deleteEmailRecipient := func(id int, recp interface{}) error {
		if err := c.db.Find(recp, id).Error; err != nil {
			return err
		}
		if err := c.db.Delete(recp).Error; err != nil {
			return err
		}
		return nil
	}

	id, err := c.extractIdFromPath(ctx)
	if err != nil {
		c.jsonError(err, w)
		return
	}

	err = deleteEmailRecipient(id, er)
	if err != nil {
		c.jsonError(err, w)
		return
	}

	enc := json.NewEncoder(w)
	if err := enc.Encode(er); err != nil {
		c.jsonError(err, w)
		return
	}

	return
}

//
// Helpers
//

func (c *Controller) extractIdFromPath(ctx web.C) (int, error) {
	idStr, there := ctx.URLParams["id"]
	if !there {
		return -1, fmt.Errorf("missing id")
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return -1, fmt.Errorf("invalid id specified")

	}
	return id, nil
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

func (c *Controller) RewriteRadioConfFile() error {
	// ensure radio conf dir with appropriate perms
	if _, err := os.Stat(RadioConfDir); os.IsNotExist(err) {
		os.Mkdir(RadioConfDir, 0750)
	}

	// write file
	contents, err := c.radioConfFileContents()
	if err != nil {
		return fmt.Errorf("Failed to generate config file contents: %s", err)
	}

	err = ioutil.WriteFile(RadioConfFile, contents, 0644)
	if err != nil {
		return fmt.Errorf("Failed to write file: %s", err)
	}

	radioUsr, _ := host.GetSystemUser("radio")
	for _, f := range []string{RadioConfDir, RadioConfFile} {
		os.Chown(f, int(radioUsr.Uid), int(radioUsr.Gid))
	}

	return nil
}

func (c *Controller) radioConfFileContents() ([]byte, error) {
	radioCfg := RadioConfig{}
	if err := c.db.First(&radioCfg).Error; err != nil {
		return []byte{}, err
	}

	makeEmailAddrSet := func(emails []string) (addrs []mail.Address, err error) {
		addrSet := map[string]bool{}
		for _, e := range emails {
			addrSet[e] = true
		}

		for k, _ := range addrSet {
			addr, err := mail.ParseAddress(k)
			if err != nil {
				// log error
			} else {
				addrs = append(addrs, *addr)
			}
		}

		return
	}

	_, err := host.GetSystemUser("radio")
	if err != nil {
		return []byte{}, fmt.Errorf("Failed to fetch radio user config")
	}

	cfg := radio.Config{
		EmailConfig: radio.EmailSettings{
			DefaultFrom:   radioCfg.DefaultFrom,
			ServerAddress: radioCfg.ServerAddress,
			ServerPort:    radioCfg.ServerPort,
			AuthHost:      radioCfg.AuthHost,
			AuthUsername:  radioCfg.AuthUsername,
			AuthPassword:  radioCfg.AuthPassword,
		},
		ProcessConfig: radio.ProcessSettings{
			ListenPort: RadioPort,
		},
	}

	for _, x := range []struct {
		source interface{}
		target *[]mail.Address
	}{
		// read_from      , assign_to
		{InfoRecipient{}, &cfg.EmailConfig.InfoRecipients},
		{WarnRecipient{}, &cfg.EmailConfig.WarnRecipients},
		{ErrorRecipient{}, &cfg.EmailConfig.ErrorRecipients},
	} {
		emails := []string{}
		if err := c.db.Model(x.source).Pluck("email", &emails).Error; err != nil {
			return []byte{}, err
		}
		addrs, err := makeEmailAddrSet(emails)
		if err != nil {
			return []byte{}, err
		}
		*x.target = addrs
	}

	ret, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return []byte{}, err
	}

	return ret, nil
}

//
// DB Models
//

type RadioConfig struct {
	DefaultFrom mail.Address

	ServerAddress string
	ServerPort    uint16

	AuthHost     string
	AuthUsername string
	AuthPassword string
}

type EmailRecipient struct {
	ID    int
	Email string
}

type InfoRecipient EmailRecipient

type WarnRecipient EmailRecipient

type ErrorRecipient EmailRecipient

//
// Callbacks
//

func (e *InfoRecipient) BeforeSave(txn *gorm.DB) error {
	_, err := mail.ParseAddress(e.Email)
	if err != nil {
		return err
	}
	return nil
}

func (e *WarnRecipient) BeforeSave(txn *gorm.DB) error {
	_, err := mail.ParseAddress(e.Email)
	if err != nil {
		return err
	}
	return nil
}

func (e *ErrorRecipient) BeforeSave(txn *gorm.DB) error {
	_, err := mail.ParseAddress(e.Email)
	if err != nil {
		return err
	}
	return nil
}

//
// Resources
//

// Resource is the exact same as what we store in the DB. No massaging needed here.
type RadioConfigResource RadioConfig

//
// Seeds
//

func (c *Controller) SeedRadioConfig() {
	for _, table := range []interface{}{
		&RadioConfig{},
		&InfoRecipient{},
		&WarnRecipient{},
		&ErrorRecipient{},
	} {
		c.db.AutoMigrate(table)
	}

	// always create one row in the radio config table (singleton row)
	c.db.FirstOrCreate(&RadioConfig{})
}
