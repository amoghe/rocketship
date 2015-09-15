package host

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"

	"rocketship/regulog"

	_ "github.com/mattn/go-sqlite3"
	. "gopkg.in/check.v1"

	"github.com/jinzhu/gorm"
	"github.com/zenazn/goji/web"
)

type UsersTestSuite struct {
	db         gorm.DB
	controller *Controller
}

func (ts *UsersTestSuite) SetUpTest(c *C) {
	db, err := gorm.Open("sqlite3", "file::memory:?cache=shared")
	c.Assert(err, IsNil)

	// Comment this to enable db logs during tests
	db.SetLogger(log.New(ioutil.Discard, "", 0))
	ts.db = db

	ts.controller = NewController(&ts.db, regulog.NewNull(""))
	ts.controller.MigrateDB()
	ts.controller.SeedDB()
}

func (ts *UsersTestSuite) TearDownTest(c *C) {
	ts.db.Close()
}

//
// Tests
//

func (ts *UsersTestSuite) TestGetCreateEndpointHandlers(c *C) {

	createUser := func(name string) UserResource {
		jsonStrFmt := `{
			"Name":     "%s",
			"Password": "weakpass",
			"Comment":  "test user"
		}`

		jsonStr := fmt.Sprintf(jsonStrFmt, name)
		req, err := http.NewRequest("POST", "/dont/care", bytes.NewBufferString(jsonStr))
		c.Assert(err, IsNil)

		rec := httptest.NewRecorder()

		// perform the request
		ts.controller.CreateUser(web.C{}, rec, req)

		// check that response is valid json resource
		bodybytes, err := ioutil.ReadAll(rec.Body)
		c.Assert(err, IsNil)

		if rec.Code != http.StatusOK {
			c.Log("Response code:", rec.Code, ". Body:", string(bodybytes))
			c.Fail()
		}

		resource := UserResource{}
		err = json.Unmarshal(bodybytes, &resource)
		c.Assert(err, IsNil)

		c.Assert(resource.ID, Not(Equals), 0)
		c.Assert(resource.Name, Equals, name)
		c.Assert(resource.Password, Equals, "")

		return resource
	}

	getUsers := func() []UserResource {
		req, err := http.NewRequest("GET", "/dont/care", &bytes.Buffer{})
		c.Assert(err, IsNil)

		rec := httptest.NewRecorder()

		// perform the request
		ts.controller.GetUsers(web.C{}, rec, req)

		// check that response is valid json resource
		bodybytes, err := ioutil.ReadAll(rec.Body)
		c.Assert(err, IsNil)

		if rec.Code != http.StatusOK {
			c.Log("Response code:", rec.Code, ". Body:", string(bodybytes))
			c.Fail()
		}

		resources := []UserResource{}
		err = json.Unmarshal(bodybytes, &resources)
		c.Assert(err, IsNil)

		return resources
	}

	deleteUser := func(id int64) UserResource {
		req, err := http.NewRequest("GET", "/dont/care", &bytes.Buffer{})
		c.Assert(err, IsNil)

		rec := httptest.NewRecorder()

		// perform the request
		ts.controller.DeleteUser(web.C{URLParams: map[string]string{"id": fmt.Sprint(id)}}, rec, req)

		// check that response is valid json resource
		bodybytes, err := ioutil.ReadAll(rec.Body)
		c.Assert(err, IsNil)

		if rec.Code != http.StatusOK {
			c.Log("Response code:", rec.Code, ". Body:", string(bodybytes))
			c.Fail()
		}

		resource := UserResource{}
		err = json.Unmarshal(bodybytes, &resource)
		c.Assert(err, IsNil)

		return resource
	}

	// create 2 users
	u1 := createUser("testUser1")
	u2 := createUser("testUser2")

	// ensure they appear in the 'get' response
	c.Assert(getUsers(), HasLen, 3) // ["admin", "testUser1", "testUser2"]

	// delete the users
	c.Assert(deleteUser(u1.ID), DeepEquals, u1)
	c.Assert(deleteUser(u2.ID), DeepEquals, u2)

	// ensure they don't appear in the 'get' response
	c.Assert(getUsers(), HasLen, 1) // ["admin"]
}

//
// Validations tests
//

func (ts *UsersTestSuite) TestPasswordHashedBeforeSave(c *C) {
	u := User{
		Name:     "jdoe",
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

func (ts *UsersTestSuite) TestUsernameLengthValidation(c *C) {
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

func (ts *UsersTestSuite) TestPasswordLengthValidation(c *C) {
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

func (ts *UsersTestSuite) TestCannotDeleteLastUser(c *C) {
	err := ts.db.Delete(&User{}, 1).Error
	c.Assert(err, Not(IsNil))
}

//
// File generation tests
//

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
	// trailing newline causes one additional token
	// seed user in db causes one additional token
	c.Assert(tokens, HasLen, len(users)+len(defaultUsers)+1+1)
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
	// trailing \n causes additional token
	// seed user in db causes one additional token
	c.Assert(tokens, HasLen, len(users)+len(defaultUsers)+1+1)
}
