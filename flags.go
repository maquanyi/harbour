package main

import (
	"fmt"
	"os"

	"github.com/huawei-openlab/harbour/mflag"
	"github.com/huawei-openlab/harbour/opts"
)

type command struct {
	name        string
	description string
}

var (
	flVersion    = mflag.Bool([]string{"v", "-version"}, false, "Print version information and quit")
	flDaemon     = mflag.Bool([]string{"d", "-daemon"}, false, "Enable daemon mode")
	flDockerSock = mflag.String([]string{"-docker-sock"}, opts.DEFAULTDOCKERSOCKET, "Path to docker sock file")
	flDebug      = mflag.Bool([]string{"D", "-debug"}, false, "Enable debug mode")
	flGroup      = mflag.String([]string{"G", "-group"}, "docker", "Group for the unix socket")
	flHelp       = mflag.Bool([]string{"h", "-help"}, false, "Print usage")
	// these are initialized in init() below
	flHosts []string
)

var (
	commands = []command{}
)

func init() {
	opts.HostListVar(&flHosts, []string{"H", "-host"}, "Daemon socket(s) to connect to")
	mflag.Usage = func() {
		fmt.Fprint(os.Stdout, "Usage: harbour [OPTIONS] [arg...]\n\nOptions:\n")

		mflag.CommandLine.SetOutput(os.Stdout)
		mflag.PrintDefaults()

		help := "\nCommands:\n"

		for _, cmd := range commands {
			help += fmt.Sprintf("    %-10.10s%s\n", cmd.name, cmd.description)
		}

		help += "\nRun 'harbour COMMAND --help' for more information on a command."
		fmt.Fprintf(os.Stdout, "%s\n", help)
	}
}
