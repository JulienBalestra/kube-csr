package main

import (
	"flag"
	"os"

	"github.com/golang/glog"

	"github.com/JulienBalestra/kube-csr/cmd"
)

func init() {
	flag.CommandLine.Parse([]string{})
}

func main() {
	command, exitCode := cmd.NewCommand()
	err := command.Execute()
	if err != nil {
		os.Exit(1)
	}
	if *exitCode != 0 {
		glog.Errorf("Exiting on error: %d", *exitCode)
		os.Exit(*exitCode)
	}
}
