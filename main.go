package main

import (
	goflag "flag"
	"fmt"
	"os"

	"github.com/golang/glog"
	"github.com/spf13/pflag"

	"github.com/choerodon/go-register-server/cmd"
)

func init() {
	goflag.Set("logtostderr", "true")
}

func main() {
	command := cmd.NewServerCommand()

	pflag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	goflag.CommandLine.Parse([]string{})

	defer glog.Flush()

	if err := command.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
