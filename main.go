package main

import "github.com/magarcia/cc2md/cmd"

var (
	version = "dev"
	commit  = "dev"
	date    = "unknown"
)

func main() {
	cmd.SetVersionInfo(version, commit, date)
	cmd.Execute()
}
