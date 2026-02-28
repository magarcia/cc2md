package main

import "github.com/magarcia/ccsession-viewer/cmd"

var (
	version = "dev"
	commit  = "dev"
	date    = "unknown"
)

func main() {
	cmd.SetVersionInfo(version, commit, date)
	cmd.Execute()
}
