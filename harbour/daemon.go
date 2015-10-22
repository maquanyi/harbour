package main

import (
	"api/server"
	"engine"
	"engine/trap"
	"mflag"
	"opts"

	"github.com/Sirupsen/logrus"
)

func mainDaemon() {
	if mflag.NArg() != 0 {
		mflag.Usage()
		return
	}

	if len(*flDockerSock) == 0 {
		engine.DockerSock = opts.DEFAULTDOCKERSOCKET
	} else {
		engine.DockerSock = *flDockerSock
	}

	if len(*flGroup) > 0 {
		engine.SocketGroup = *flGroup
	}

	eng := engine.New()

	//catch signals
	trap.SignalsHandler(trap.Shutdown)

	var srv *server.Server
	srv = server.New(eng, false)

	serverWait := make(chan error)
	go func() {
		if err := srv.CreateServer(eng, flHosts); err != nil {
			logrus.Errorf("Server error: %v", err)
			serverWait <- err
			return
		}
		serverWait <- nil
	}()
	err := <-serverWait
	if err != nil {
		logrus.Fatalf("Shutting down due to Server error: %v", err)
	}
	<-trap.CleanupDone
}
