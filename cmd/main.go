package main

import (
	"github.com/ssvlabsinfra/ssv-benchmark/cli"
)

var (
	// AppName is the application name
	AppName = "ssv-analyzer"

	// Version is the app version
	Version = "latest"
)

func main() {
	cli.Execute(AppName, Version)
}
