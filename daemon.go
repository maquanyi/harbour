package main

import (
	"github.com/Sirupsen/logrus"
	"github.com/huawei-openlab/harbour/api/server"
	"github.com/huawei-openlab/harbour/engine"
	"github.com/huawei-openlab/harbour/engine/trap"
	"github.com/huawei-openlab/harbour/mflag"
	"github.com/huawei-openlab/harbour/opts"
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

	if len(*flRuntime) == 0 {
		engine.ContainerRuntime = opts.DEFAULTRUNTIME
	} else {
		engine.ContainerRuntime = *flRuntime
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
