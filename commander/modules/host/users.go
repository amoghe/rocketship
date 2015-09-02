package host

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/amoghe/go-crypt"
)

const (
	// Datum from which we start computing uid's for configured users.
	UIDDatum = 2000

	ValidUsernameChars    = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz123456789-_"
	MaxUsernameLen        = 12
	MinUsernameLen        = 2
	MaxPasswordLen        = 32
	MinPasswordLen        = 8
	SaltSize              = 16
	ShadowFileSeparator   = ":"
	PasswordFileSeparator = ":"

	PasswdFilePath = "/etc/passwd"
	ShadowFilePath = "/etc/shadow"
)

var (
	defaultUsers = []DefaultUser{
		//   name          comment      uid    group          homedir
		{User: User{
			Name:    "root",
			Comment: "Superuser",
			Homedir: "/root",
		},
			Uid: 0,
			Gid: int(defaultGroups["root"]),
		},
		{User: User{
			Name:    "daemon",
			Comment: "",
			Homedir: "/usr/sbin",
		},
			Uid: 1,
			Gid: defaultGroups["daemon"],
		},
		{User: User{
			Name:    "bin",
			Comment: "",
			Homedir: "/bin",
		},
			Uid: 2,
			Gid: defaultGroups["bin"],
		},
		{User: User{
			Name:    "sys",
			Comment: "",
			Homedir: "/dev",
		},
			Uid: 3,
			Gid: defaultGroups["sys"],
		},
		{User: User{
			Name:    "www-data",
			Comment: "",
			Homedir: "/var/www",
		},
			Uid: 33,
			Gid: defaultGroups["www-data"],
		},
		{User: User{
			Name:    "libuuid",
			Comment: "",
			Homedir: "/var/lib/uuid",
		},
			Uid: 100,
			Gid: defaultGroups["libuuid"],
		},
		{User: User{
			Name:    "syslog",
			Comment: "",
			Homedir: "/home/syslog",
		},
			Uid: 101,
			Gid: defaultGroups["syslog"],
		},
		{User: User{
			Name:    "messagebus",
			Comment: "",
			Homedir: "/var/run/dbus",
		},
			Uid: 102,
			Gid: defaultGroups["messagebus"],
		},
		{User: User{
			Name:    "sshd",
			Comment: "SSH daemon",
			Homedir: "/var/run/sshd",
		},
			Uid: 105,
			Gid: defaultGroups["nogroup"],
		},
		{User: User{
			Name:    "nobody",
			Comment: "",
			Homedir: "/nonexistent",
		},
			Uid: 65534,
			Gid: defaultGroups["nogroup"],
		},
		//
		// Rocketship users
		//
		{User: User{
			Name:    "radio",
			Comment: "SSH daemon",
			Homedir: "/tmp",
		},
			Uid: 1001,
			Gid: defaultGroups["radio"]},
		// write config file
		{User: User{
			Name:    "crashcorder",
			Comment: "crash reporter daemon",
			Homedir: "/tmp",
		},
			Uid: 1002,
			Gid: defaultGroups["crashcorder"],
		},
	}
)

// GetSystemUser returns a user struct populated with the details of a default ("system")
// user. This is used by other modules to query what uid/gid they should run their programs with.
func GetSystemUser(name string) (DefaultUser, error) {

	for _, user := range defaultUsers {
		if name == user.Name {
			return user, nil
		}
	}
	return DefaultUser{}, fmt.Errorf("No such user")
}

//
// File generators
//

func (c *Controller) RewritePasswdFile() error {
	contents, err := c.passwdFileContents()
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(PasswdFilePath, contents, 0644)
	if err != nil {
		return err
	}

	return nil
}

func (c *Controller) RewriteShadowFile() error {
	contents, err := c.shadowFileContents()
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(ShadowFilePath, contents, 0644)
	if err != nil {
		return err
	}

	return nil
}

func (c *Controller) EnsureHomedirs() error {
	users := []User{}
	failed := map[string]bool{}

	err := c.db.Find(&users).Error
	if err != nil {
		return err
	}

	for _, user := range users {
		dirname := fmt.Sprintf("/home/%s", user.Name)

		err = os.Mkdir(dirname, 0777)
		if err != nil {
			failed[user.Name] = true
			continue
		}

		err = os.Chown(dirname, user.Uid(), user.Gid())
		if err != nil {
			failed[user.Name] = true
			continue
		}

	}

	// TODO:  If there were errors creating any of the homedirs, return them
	return nil
}

func (c *Controller) passwdFileContents() ([]byte, error) {
	var (
		contents = bytes.Buffer{}
		users    = []User{}
	)

	err := c.db.Find(&users).Error
	if err != nil {
		return []byte{}, err
	}

	for _, user := range defaultUsers {
		contents.WriteString(user.PasswdFileEntry())
		contents.WriteString("\n")
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

	for _, user := range defaultUsers {
		contents.WriteString(user.ShadowFileEntry())
		contents.WriteString("\n")
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

// Uid returns the Uid for this user.
func (u User) Uid() int {
	return int(UIDDatum + u.ID)
}

// Gid returns the Gid for this user.
func (u User) Gid() int {
	return int(GIDDatum + u.ID)
}

func (u User) PasswdFileEntry() string {
	shell := "/bin/bash" // TODO: changeme
	return strings.Join([]string{
		u.Name,
		"x",
		fmt.Sprintf("%d", u.Uid()),
		fmt.Sprintf("%d", u.Gid()),
		"",
		"/home/" + u.Name,
		shell,
	}, PasswordFileSeparator)
}

func (u User) ShadowFileEntry() string {
	lastUpdateDays := u.UpdatedAt.Unix() / int64(time.Hour*24)
	return strings.Join([]string{
		u.Name,
		u.HashedPassword,
		fmt.Sprintf("%d", lastUpdateDays),
		"0",     // TODO: ???
		"99999", // TODO: ()
		"7",     // TODO: (warn)
		"",      // TODO: (inactive)
		"",      // TODO: (expire)
		"",      // NYI   (reserved, future)
	}, ShadowFileSeparator)
}

// DefaultUser wraps around the User model to provide additional functionality.
// These are not intended to be persisted to the DB, they exist so that we can treat built-in
// users (baked into the os) specially.
type DefaultUser struct {
	User
	Uid int
	Gid int
}

func (d DefaultUser) PasswdFileEntry() string {
	return strings.Join([]string{
		d.Name,
		"x",
		strconv.Itoa(int(d.Uid)),
		strconv.Itoa(int(d.Gid)),
		d.Comment,
		d.Homedir,
		"/bin/nologin", // not allowed to log in
	}, PasswordFileSeparator)
}

func (d DefaultUser) ShadowFileEntry() string {
	return strings.Join([]string{
		d.Name,
		"*", // not allowed to login
		"15785",
		"0",     // TODO: ???
		"99999", // TODO: ()
		"7",     // TODO: (warn)
		"",      // TODO: (inactive)
		"",      // TODO: (expire)
		"",      // NYI   (reserved, future)
	}, ShadowFileSeparator)
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

func (c *Controller) seedUsers() {
	c.db.FirstOrCreate(&User{Name: "admin", Password: "password"})
}
