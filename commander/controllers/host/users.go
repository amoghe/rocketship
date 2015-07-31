package host

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/amoghe/go-crypt"
	"github.com/jinzhu/gorm"
)

const (
	// Users
	UIDDatum              = 2000
	GIDDatum              = 2000
	ValidUsernameChars    = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz123456789-_"
	MaxUsernameLen        = 12
	MinUsernameLen        = 2
	MaxPasswordLen        = 32
	MinPasswordLen        = 8
	SaltSize              = 16
	ShadowFileSeparator   = ":"
	PasswordFileSeparator = ":"
)

//
// File generators
//

func (c *Controller) passwdFileContents() ([]byte, error) {
	var (
		contents = bytes.Buffer{}
		users    = []User{}
	)

	err := c.db.Find(&users).Error
	if err != nil {
		return []byte{}, err
	}

	for _, user := range users {
		contents.WriteString(user.PasswdFileEntry())
		contents.WriteString("\n")
	}

	return contents.Bytes(), nil
}

func (c *Controller) shadowFileContents() ([]byte, error) {
	var (
		contents = bytes.Buffer{}
		users    = []User{}
	)

	err := c.db.Find(&users).Error
	if err != nil {
		return []byte{}, err
	}

	for _, user := range users {
		contents.WriteString(user.ShadowFileEntry())
		contents.WriteString("\n")
	}

	return contents.Bytes(), nil
}

//
// DB Models
//

type User struct {
	ID             int64
	Name           string
	Comment        string
	Uid            uint32
	Gid            uint32
	Homedir        string
	Login          bool
	Password       string `sql:"-"`
	HashedPassword string

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (u *User) BeforeSave() error {
	makeSalt := func() ([]byte, error) {
		buf := make([]byte, SaltSize+8)
		_, err := rand.Read(buf)
		if err != nil {
			return nil, fmt.Errorf("Random read failed: %v", err)
		}

		return []byte(base64.StdEncoding.EncodeToString(buf))[:SaltSize], nil
	}

	if len(u.Password) > 0 {
		salt, err := makeSalt()
		if err != nil {
			return err
		}

		u.HashedPassword, err = crypt.Crypt(u.Password, fmt.Sprintf("$6$%16s$", salt))
		if err != nil {
			return err
		}
	} else {
		u.HashedPassword = ""
	}

	return nil
}

func (u *User) BeforeCreate() error {
	EBadUsernameChar := fmt.Errorf("Username can only contain upper/lower case alphabets " +
		"and numbers.")
	EBadUsernameLen := fmt.Errorf("Username must be between %d and %d chars",
		MinUsernameLen, MaxUsernameLen)
	EBadPasswordLen := fmt.Errorf("Password must be between %d and %d chars",
		MinPasswordLen, MaxPasswordLen)

	if len(u.Password) < MinPasswordLen || len(u.Password) > MaxPasswordLen {
		return EBadPasswordLen
	}
	if len(u.Name) < MinUsernameLen || len(u.Name) > MaxUsernameLen {
		return EBadUsernameLen
	}
	for _, char := range []byte(u.Name) {
		if !strings.Contains(ValidUsernameChars, string(char)) {
			return EBadUsernameChar
		}
	}
	return nil
}

//
// Helpers
//

func (u User) PasswdFileEntry() string {
	shell := "/bin/bash" // TODO: changeme
	return strings.Join([]string{
		u.Name,
		"x",
		fmt.Sprintf("%d", UIDDatum+u.Uid),
		fmt.Sprintf("%d", GIDDatum+u.Gid),
		"",
		"/home/" + u.Name,
		shell,
	}, PasswordFileSeparator)
}

func (u User) ShadowFileEntry() string {
	lastUpdateDays := u.UpdatedAt.Unix() / int64(time.Hour*24)
	fields := []string{
		u.Name,
		u.HashedPassword,
		fmt.Sprintf("%d", lastUpdateDays),
		"0",     // TODO: ???
		"99999", // TODO: ()
		"7",     // TODO: (warn)
		"",      // TODO: (inactive)
		"",      // TODO: (expire)
		"",      // NYI   (reserved, future)
	}

	return strings.Join(fields, ShadowFileSeparator)
}

//
// Resources
//

type UserResource struct {
	ID       int64
	Name     string
	Password string // WRITE ONLY
	Comment  string
}

func (u UserResource) ToUserModel() User {
	return User{
		ID:       u.ID,
		Name:     u.Name,
		Comment:  u.Comment,
		Password: u.Password,
	}
}

func (u *UserResource) FromUserModel(m User) {
	u.ID = m.ID
	u.Name = m.Name
	u.Comment = m.Comment

	// NEVER return the password
	// u.Password = m.Password
}

//
// DB Seed
//

func SeedUsers(db *gorm.DB) {
	db.AutoMigrate(&User{})

	defaultUsers := []User{
		{},
	}
	for _, u := range defaultUsers {
		db.FirstOrCreate(&u)
	}
}
