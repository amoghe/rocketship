package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"rocketship/commander/modules/host"
	"strings"

	"github.com/parnurzeal/gorequest"
	"golang.org/x/crypto/ssh/terminal"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	// The 'gorequest' SuperAgent we'll use for all our requests.
	req = gorequest.New()

	listCmdStr   = "list"
	createCmdStr = "create"
	deleteCmdStr = "delete"

	getCmd    = kingpin.Command(listCmdStr, "List users")
	createCmd = kingpin.Command(createCmdStr, "Create a user")
	deleteCmd = kingpin.Command(deleteCmdStr, "Delete a user")

	// create opts
	name    = createCmd.Flag("name", "Name of user to be created").String()
	comment = createCmd.Flag("comment", "Comment string (admin purpose)").String()

	// delete opts
	id = deleteCmd.Flag("id", "ID of user to be deleted").Default("0").Int()
)

func main() {
	kingpin.Version("1.0")
	mode := kingpin.Parse()

	switch mode {
	case listCmdStr:
		doListUsers()
	case createCmdStr:
		doCreateUser()
	case deleteCmdStr:
		doDeleteUser()
	default:
		fmt.Println("Unknown subcommand:", mode)
		os.Exit(1)
	}
}

func doListUsers() {

	_, body, errs := req.Get("http://localhost:8888" + host.EUsers).End()
	if errs != nil {
		fmt.Println(errs)
		os.Exit(1)
	}

	users := []host.UserResource{}
	if err := json.Unmarshal([]byte(body), &users); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Printf("ID\tName\tComment\n")
	fmt.Printf("--\t----\t-------\n")
	for _, user := range users {
		fmt.Printf("%2d\t%s\t%s\n", user.ID, user.Name, user.Comment)
	}
}

func doCreateUser() {
	var ()

	promptPassword := func(prompt string) string {
		fmt.Printf(prompt)
		defer fmt.Println("")

		pass, err := terminal.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			fmt.Println("Failed to read password:", err)
			os.Exit(1)
		}

		return string(pass)
	}

	if len(*name) <= 0 {
		fmt.Println("Cannot have empty user name")
		os.Exit(1)
	}

	pass1 := promptPassword("Enter password:")
	pass2 := promptPassword("Re-enter password:")

	if string(pass1) != string(pass2) {
		fmt.Println("Passwords do not match")
		os.Exit(1)
	}

	user := host.UserResource{
		Name:     *name,
		Comment:  *comment,
		Password: pass1,
	}

	res, body, errs := req.
		Post("http://localhost:8888" + host.EUsers).
		Send(user).
		End()
	if errs != nil {
		fmt.Println(errs)
		os.Exit(1)
	}
	if res.StatusCode != http.StatusOK {
		fmt.Println("Error response from server:")
		fmt.Println("\tCode:\t", res.StatusCode)
		fmt.Println("\tBody:\t", body)
		os.Exit(1)
	}

	fmt.Println("Hostname updated succesfully")
}

func doDeleteUser() {
	if *id <= 0 {
		fmt.Println("Must specify user ID to delete")
		os.Exit(1)
	}

	endpoint := strings.Replace(host.EUsersID, ":id", fmt.Sprintf("%d", *id), 1)
	fmt.Println("endpoint:", endpoint)

	res, body, errs := req.
		Delete("http://localhost:8888" + endpoint).
		End()
	if errs != nil {
		fmt.Println(errs)
		os.Exit(1)
	}
	if res.StatusCode != http.StatusOK {
		fmt.Println("Error response from server:")
		fmt.Println("\tCode:\t", res.StatusCode)
		fmt.Println("\tBody:\t", body)
		os.Exit(1)
	}

	user := host.UserResource{}
	if err := json.Unmarshal([]byte(body), &user); err != nil {
		fmt.Println("Deleted user response contains invalid JSON. It is likely that user deletion has ")
		fmt.Println("succeeded. Please confirm by running the \"list\" command.")
		os.Exit(1)
	}

	fmt.Println("Deleted user:")
	fmt.Println("\tID\t:", user.ID)
	fmt.Println("\tName\t:", user.Name)
	fmt.Println("\tComment\t:", user.Comment)
}
