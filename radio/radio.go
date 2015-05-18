package radio

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/smtp"

	"github.com/jpoehls/gophermail"
	"github.com/zenazn/goji/web"
)

const (
	LevelInfo  = "INFO"
	LevelWarn  = "WARN"
	LevelError = "ERROR"
)

// MessageRequest contains fields of the message to be sent. This structure is used
// by callers who wish to send notifications via the radio service when making the
// request to the radio service.
type MessageRequest struct {
	Severity string
	Subject  string
	Body     string
}

type Radio struct {
	mux    *web.Mux
	config Config
}

// New returns an initialized instance of a Radio struct.
func New(c Config) *Radio {
	r := Radio{
		mux:    web.New(),
		config: c,
	}

	r.mux.Post("/email", r.HandleEmailRequest)

	return &r
}

// ServeHTTP allows Radio to be a compliant net/http handler
func (r *Radio) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(res, req)
}

//
// Handlers
//

func (r *Radio) HandleEmailRequest(c web.C, w http.ResponseWriter, req *http.Request) {
	emsg, err := r.parseMessageFromRequest(req)
	if err != nil {
		// Increment error stats
		return
	}

	auth := smtp.PlainAuth("",
		r.config.EmailConfig.AuthUsername,
		r.config.EmailConfig.AuthPassword,
		r.config.EmailConfig.AuthHost)

	addr := fmt.Sprintf("%s:%d",
		r.config.EmailConfig.ServerAddress,
		r.config.EmailConfig.ServerPort)

	err = gophermail.SendMail(addr, auth, &emsg)
	if err != nil {
		// Increment error stats
		return
	}
}

//
// Helpers
//

func (r *Radio) parseMessageFromRequest(req *http.Request) (gophermail.Message, error) {
	var (
		msg  MessageRequest
		emsg gophermail.Message
	)

	reqbody, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return emsg, err
	}

	err = json.Unmarshal(reqbody, &msg)
	if err != nil {
		return emsg, err
	}

	switch severity := msg.Severity; severity {
	case LevelInfo:
		emsg.To = r.config.EmailConfig.InfoRecipients
	case LevelWarn:
		emsg.To = r.config.EmailConfig.WarnRecipients
	case LevelError:
		emsg.To = r.config.EmailConfig.ErrorRecipients
	default:
		return emsg, fmt.Errorf("Invalid email severity %s specified", severity)
	}

	emsg.From = r.config.EmailConfig.DefaultFrom
	emsg.Subject = msg.Subject
	emsg.Body = msg.Body

	return emsg, nil
}
