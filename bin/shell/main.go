package main

import (
	"os"

	"rocketship/shell"
)

func main() {
	shell := shell.New()
	defer shell.Close()

	shell.HistoryFilePath = "" // TODO: set this to users homedir
	shell.Outputln("Welcome, ", os.Getenv("USER"))

	shell.Run()
}
