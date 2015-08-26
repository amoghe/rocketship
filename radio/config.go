package radio

import (
	"net/mail"
)

// EmailSettings holds the configuration pertaining to the delivery of email
type EmailSettings struct {
	DefaultFrom     mail.Address
	InfoRecipients  []mail.Address
	WarnRecipients  []mail.Address
	ErrorRecipients []mail.Address

	ServerAddress string
	ServerPort    uint16

	AuthHost     string
	AuthUsername string
	AuthPassword string
}

// ProcessSettings holds the configuration pertinent to running the process
type ProcessSettings struct {
	ListenPort uint16
}

// Config is a shell structure to hold the various configs needed by the Radio service
type Config struct {
	EmailConfig   EmailSettings
	ProcessConfig ProcessSettings
}
