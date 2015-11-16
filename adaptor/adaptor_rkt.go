package adaptor

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os/exec"
	"regexp"
	"strings"

	"github.com/Sirupsen/logrus"
)

const (
	GET = iota
	POST
	DELETE
)

type UserConfig struct {
	Hostname string // Hostname
	Image    string // Name of the image as it was passed by the operator (eg. could be symbolic)
}

func Rkt_Rundockercmd(r *http.Request, method int) error {

	if method == DELETE {
		return rktCmdRm(r)
	}

	createMatch, _ := regexp.MatchString("/create", r.URL.Path)
	if createMatch {
		return rktCmdRun(r)
	}

	listMatch, _ := regexp.MatchString("/containers/json", r.URL.Path)
	if listMatch {
		return rktCmdList(r)
	}

	imageMatch, _ := regexp.MatchString("/images/json", r.URL.Path)
	if imageMatch {
		return rktCmdImage(r)
	}

	versionMatch, _ := regexp.MatchString("/version", r.URL.Path)
	if versionMatch {
		return rktCmdVersion(r)
	}

	return nil
}

func rktCmdRun(r *http.Request) error {
	var cmdStr string
	var config UserConfig

	requestBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logrus.Errorf("Read request body error: %s", err)
		return err
	}
	cmdStr = strings.TrimRight(string(requestBody), "\n")
	logrus.Debugf("Transforwarding request body: %s", cmdStr)
	json.Unmarshal([]byte(cmdStr), &config)
	cmdStr = "docker://" + config.Image

	err = run(exec.Command("rkt", "--insecure-skip-verify", "--interactive", "--mds-register=false", "run", cmdStr))

	return err
}

func rktCmdList(r *http.Request) error {
	var cmdStr string

	requestBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logrus.Errorf("Read request body error: %s", err)
		return err
	}

	cmdStr = strings.TrimRight(string(requestBody), "\n")
	logrus.Debugf("Transforwarding request body: %s", cmdStr)

	cmdStr = "list"

	err = run(exec.Command("rkt", cmdStr))

	return err
}

func rktCmdImage(r *http.Request) error {
	var cmdStr string

	requestBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logrus.Errorf("Read request body error: %s", err)
		return err
	}

	cmdStr = strings.TrimRight(string(requestBody), "\n")
	logrus.Debugf("Transforwarding request body: %s", cmdStr)

	cmdStr = "list"

	err = run(exec.Command("rkt", "image", cmdStr))

	return err
}

func rktCmdVersion(r *http.Request) error {
	var cmdStr string

	requestBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logrus.Errorf("Read request body error: %s", err)
		return err
	}

	cmdStr = strings.TrimRight(string(requestBody), "\n")
	logrus.Debugf("Transforwarding request body: %s", cmdStr)

	cmdStr = "version"

	err = run(exec.Command("rkt", cmdStr))

	return err
}

func rktCmdRm(r *http.Request) error {
	var cmdStr string

	requestBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logrus.Errorf("Read request body error: %s", err)
		return err
	}

	cmdStr = strings.TrimRight(string(requestBody), "\n")
	logrus.Debugf("Transforwarding request body: %s", cmdStr)

	cmdStr = "gc"

	err = run(exec.Command("rkt", cmdStr))

	return err
}
