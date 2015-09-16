package radio

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/mail"
	"os"
	"strconv"
	"sync"

	"github.com/amoghe/go-upstart"
	"github.com/jinzhu/gorm"
	"github.com/zenazn/goji/web"

	"rocketship/commander/modules/host"
	"rocketship/radio"
	"rocketship/regulog"
)

const (
	NoApplyEnvKey = "noapply"

	URLPrefix = "/radio"

	EInfoRecipients  = URLPrefix + "/recipients/info"
	EWarnRecipients  = URLPrefix + "/recipients/warn"
	EErrorRecipients = URLPrefix + "/recipients/error"

	EInfoRecipientsID  = EInfoRecipients + "/:id"
	EWarnRecipientsID  = EWarnRecipients + "/:id"
	EErrorRecipientsID = EErrorRecipients + "/:id"

	RadioPort = 12345

	RadioConfDir  = "/etc/radio"
	RadioConfFile = RadioConfDir + "/radio.conf"
)

type Controller struct {
	db   *gorm.DB
	mux  *web.Mux
	log  regulog.Logger
	lock sync.Mutex
}

func NewController(db *gorm.DB, logger regulog.Logger) *Controller {
	ctrl := &Controller{db: db, mux: web.New(), log: logger}

	ctrl.mux.Get(EInfoRecipients, ctrl.GetInfoRecipients)
	ctrl.mux.Get(EWarnRecipients, ctrl.GetWarnRecipients)
	ctrl.mux.Get(EErrorRecipients, ctrl.GetErrorRecipients)

	ctrl.mux.Post(EInfoRecipients, ctrl.AddInfoRecipient)
	ctrl.mux.Post(EWarnRecipients, ctrl.AddWarnRecipient)
	ctrl.mux.Post(EErrorRecipients, ctrl.AddErrorRecipient)

	ctrl.mux.Delete(EInfoRecipientsID, ctrl.DeleteInfoRecipient)
	ctrl.mux.Delete(EWarnRecipientsID, ctrl.DeleteWarnRecipient)
	ctrl.mux.Delete(EErrorRecipientsID, ctrl.DeleteErrorRecipient)

	return ctrl
}

// ServeHTTP satisfies the http.Handler interface (net/http as well as goji)
func (c *Controller) ServeHTTPC(ctx web.C, w http.ResponseWriter, r *http.Request) {
	c.lock.Lock()
	c.mux.ServeHTTPC(ctx, w, r)
	c.lock.Unlock()
	return
}

// RoutePrefix returns the prefix under which this router handles endpoints
func (c *Controller) RoutePrefix() string {
	return URLPrefix
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
func (c *Controller) AddInfoRecipient(ctx web.C, w http.ResponseWriter, r *http.Request) {
	c.addRecipient(w, r, InfoRecipient{})
	c.maybeFlapRadio(ctx)
}
func (c *Controller) AddWarnRecipient(ctx web.C, w http.ResponseWriter, r *http.Request) {
	c.addRecipient(w, r, WarnRecipient{})
	c.maybeFlapRadio(ctx)
}
func (c *Controller) AddErrorRecipient(ctx web.C, w http.ResponseWriter, r *http.Request) {
	c.addRecipient(w, r, ErrorRecipient{})
	c.maybeFlapRadio(ctx)
}

// Del
func (c *Controller) DeleteInfoRecipient(ctx web.C, w http.ResponseWriter, r *http.Request) {
	c.deleteRecipient(ctx, w, r, &InfoRecipient{})
	c.maybeFlapRadio(ctx)
}
func (c *Controller) DeleteWarnRecipient(ctx web.C, w http.ResponseWriter, r *http.Request) {
	c.deleteRecipient(ctx, w, r, &WarnRecipient{})
	c.maybeFlapRadio(ctx)
}
func (c *Controller) DeleteErrorRecipient(ctx web.C, w http.ResponseWriter, r *http.Request) {
	c.deleteRecipient(ctx, w, r, &ErrorRecipient{})
	c.maybeFlapRadio(ctx)
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

func (c *Controller) maybeFlapRadio(ctx web.C) error {
	if _, there := ctx.Env[NoApplyEnvKey]; there {
		c.log.Infoln("Skipping apply radio config to system (\"noapply\" present in env)")
		return nil
	}
	if err := c.RewriteFiles(); err != nil {
		return err
	}
	if err := upstart.RestartJob("radio"); err != nil {
		return err
	}
	return nil
}

//
// File generators
//

func (c *Controller) RewriteFiles() error {
	c.log.Infoln("Rewriting radio configuration file")
	// ensure radio conf dir
	if _, err := os.Stat(RadioConfDir); os.IsNotExist(err) {
		if err := os.Mkdir(RadioConfDir, 0750); err != nil {
			return fmt.Errorf("Failed to ensure radio config dir: %s", err)
		}
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

	// ensure dir and file perms
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

func (c *Controller) MigrateDB() {
	c.log.Infoln("Migrating radio configuration tables")
	for _, table := range []interface{}{
		&RadioConfig{},
		&InfoRecipient{},
		&WarnRecipient{},
		&ErrorRecipient{},
	} {
		c.db.AutoMigrate(table)
	}
}

func (c *Controller) SeedDB() {
	c.log.Infoln("Seeding radio configuration")
	// always create one row in the radio config table (singleton row)
	c.db.FirstOrCreate(&RadioConfig{})
}
