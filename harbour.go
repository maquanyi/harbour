// harbour project main.go
package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/huawei-openlab/harbour/mflag"
	"github.com/huawei-openlab/harbour/opts"

	"github.com/Sirupsen/logrus"
)

func main() {
	mflag.Parse()

	if *flVersion {
		showVersion()
		return
	}

	if *flHelp {
		mflag.Usage()
		return
	}

	if *flDebug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	if len(*flRuntime) != 0 {
		if *flRuntime != opts.DEFAULTRUNTIME && *flRuntime != opts.RKTRUNTIME {
			fmt.Println("Invalid container runtime")
			return
		}
	}

	if len(flHosts) == 0 {
		defaultHost := fmt.Sprintf("unix://%s", opts.DEFAULTUNIXSOCKET)
		flHosts = append(flHosts, defaultHost)
	}

	_, ok := exec.LookPath("docker")
	if ok != nil {
		logrus.Fatal("Can't find docker")
	}

	if *flDaemon {
		mainDaemon()
		return
	}

	if len(flHosts) > 1 {
		fmt.Fprintf(os.Stderr, "Please specify only one -H")
		os.Exit(0)
	}

	// If no flag specified, print help info.
	mflag.Usage()
}

func showVersion() {
	fmt.Printf("harbour version 0.0.1\n")
}
