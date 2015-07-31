package host

import (
	"strings"

	. "gopkg.in/check.v1"

	"github.com/jinzhu/gorm"
)

type UsersTestSuite struct {
	db         gorm.DB
	controller *Controller
}

func (ts *UsersTestSuite) SetUpTest(c *C) {
	db, err := gorm.Open("sqlite3", "file::memory:?cache=shared")
	c.Assert(err, IsNil)
	ts.db = db

	ts.controller = NewController(&ts.db)
	ts.controller.MigrateDB()
}

func (ts *UsersTestSuite) TearDownTest(c *C) {
	ts.db.Close()
}

//
// Tests
//

func (ts *UsersTestSuite) TestPasswordHashedBeforeSave(c *C) {
	u := User{
		Name:     "jdoe",
		Uid:      1001,
		Gid:      1001,
		Password: "somepass",
	}

	c.Assert(u.BeforeSave(), IsNil)

	tokens := strings.Split(u.HashedPassword, "$")
	c.Assert(len(tokens), Equals, 4)
	c.Assert(tokens[1], Equals, "6")
}

func (ts *UsersTestSuite) TestPasswordNotSavedToDB(c *C) {
	u1 := User{
		ID:       42,
		Name:     "johndoe",
		Uid:      1001,
		Gid:      1001,
		Password: "somepass",
	}

	err := ts.db.Create(&u1).Error
	c.Assert(err, IsNil)

	u2 := User{}
	err = ts.db.Find(&u2, 42).Error
	c.Assert(err, IsNil)
	c.Assert(len(u2.Password), Equals, 0)
	c.Assert(u2.Name, Equals, u1.Name)
}

func (ts *UsersTestSuite) TestUsernameLength(c *C) {
	badNames := []string{
		strings.Repeat("a", 42), // too long
		"x",           // too short
		"with spaces", // illegal chars
		"*()",         // illegal chars
	}

	user := User{Password: "foobar4242"}
	for _, name := range badNames {
		user.Name = name
		c.Assert(user.BeforeSave(), Not(Equals), IsNil)
	}
}

func (ts *UsersTestSuite) TestPasswordLength(c *C) {
	badPasswords := []string{
		strings.Repeat("a", 42),
		"x",
	}

	user := User{Name: "foobar"}
	for _, pass := range badPasswords {
		user.Password = pass
		c.Assert(user.BeforeSave(), Not(Equals), IsNil)
	}
}

func (ts *UsersTestSuite) TestPasswdFileContents(c *C) {
	users := []User{
		{
			Name:     "test2",
			Password: "password1",
		},
		{
			Name:     "test2",
			Password: "password2",
		},
	}

	for _, u := range users {
		c.Assert(ts.db.Create(&u).Error, IsNil)
	}

	f, err := ts.controller.passwdFileContents()
	c.Assert(err, IsNil)

	tokens := strings.Split(string(f), "\n")
	c.Assert(tokens, HasLen, len(users)+len(defaultUsers)+1) // trailing newline causes one additional token
}

func (ts *UsersTestSuite) TestShadowFileContents(c *C) {
	users := []User{
		{
			Name:     "test2",
			Password: "password1",
		},
		{
			Name:     "test2",
			Password: "password2",
		},
	}

	for _, u := range users {
		c.Assert(ts.db.Create(&u).Error, IsNil)
	}

	f, err := ts.controller.shadowFileContents()
	c.Assert(err, IsNil)

	tokens := strings.Split(string(f), "\n")
	c.Assert(tokens, HasLen, len(users)+len(defaultUsers)+1) // trailing \n causes additional token
}
