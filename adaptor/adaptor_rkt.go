package adaptor

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os/exec"
	"regexp"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/huawei-openlab/harbour/utils"
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
		rmMatch, _ := regexp.MatchString("/containers/", r.URL.Path)
		if rmMatch {
			return rktCmdRm(r)
		}
		rmiMatch, _ := regexp.MatchString("/images/", r.URL.Path)
		if rmiMatch {
			return rktCmdRmi(r)
		}
	}

	// docker run --> rkt run
	runMatch, _ := regexp.MatchString("/containers/create", r.URL.Path)
	if runMatch {
		return rktCmdRun(r)
	}

	// docker ps --> rkt list
	listMatch, _ := regexp.MatchString("/containers/json", r.URL.Path)
	if listMatch {
		return rktCmdList(r)
	}

	// docker images --> rkt image list
	imageMatch, _ := regexp.MatchString("/images/json", r.URL.Path)
	if imageMatch {
		return rktCmdImage(r)
	}

	// docker version --> rkt version
	versionMatch, _ := regexp.MatchString("/version", r.URL.Path)
	if versionMatch {
		return rktCmdVersion(r)
	}

	// docker stats --> rkt status
	statsMatch, _ := regexp.MatchString("/stats", r.URL.Path)
	if statsMatch {
		return rktCmdStats(r)
	}

	// docker pull --> rkt fetch
	fetchMatch, _ := regexp.MatchString("/images/create", r.URL.Path)
	if fetchMatch {
		return rktCmdFetch(r)
	}

	// docker attach --> rkt enter
	enterMatch, _ := regexp.MatchString(".*/containers/.*/json", r.URL.Path)
	if enterMatch {
		return rktCmdEnter(r)
	}

	// docker save --> rkt export
	exportMatch, _ := regexp.MatchString(".*/images/.*/get", r.URL.Path)
	if exportMatch {
		return rktCmdExport(r)
	}

	// docker inspect --> rkt image cat-manifest
	manifestMatch, _ := regexp.MatchString(".*/images/.*/json", r.URL.Path)
	if manifestMatch {
		return rktCmdCatmanifest(r)
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
	cmdStr = "rkt " + "--interactive " + "--insecure-skip-verify " + "--mds-register=false " + "run "
	cmdStr += "docker://" + config.Image

	err = utils.Run(exec.Command("/bin/sh", "-c", cmdStr))

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

	err = utils.Run(exec.Command("rkt", cmdStr))

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

	err = utils.Run(exec.Command("rkt", "image", cmdStr))

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

	err = utils.Run(exec.Command("rkt", cmdStr))

	return err
}

func rktCmdRm(r *http.Request) error {
	var cmdStr string
	var rktID []string

	requestBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logrus.Errorf("Read request body error: %s", err)
		return err
	}

	cmdStr = strings.TrimRight(string(requestBody), "\n")
	logrus.Debugf("Transforwarding request body: %s", cmdStr)

	rktID = strings.SplitAfter(r.URL.Path, "containers/")
	if len(rktID) < 2 {
		return nil
	}

	if rktID[1] == "all" {
		cmdStr = "rkt gc"
	} else {
		cmdStr = "rkt rm --insecure-skip-verify " + rktID[1]
	}

	err = utils.Run(exec.Command("/bin/sh", "-c", cmdStr))

	return err
}

func rktCmdRmi(r *http.Request) error {
	var cmdStr string
	var imgID []string

	requestBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logrus.Errorf("Read request body error: %s", err)
		return err
	}

	cmdStr = strings.TrimRight(string(requestBody), "\n")
	logrus.Debugf("Transforwarding request body: %s", cmdStr)

	imgID = strings.SplitAfter(r.URL.Path, "images/")
	if len(imgID) < 2 {
		return nil
	}

	cmdStr = "rkt image rm " + imgID[1]

	err = utils.Run(exec.Command("/bin/sh", "-c", cmdStr))

	return err
}

func rktCmdStats(r *http.Request) error {
	var cmdStr string
	var rktID []string

	requestBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logrus.Errorf("Read request body error: %s", err)
		return err
	}

	cmdStr = strings.TrimRight(string(requestBody), "\n")
	logrus.Debugf("Transforwarding request body: %s", cmdStr)

	rktID = strings.SplitAfter(r.URL.Path, "containers/")
	if len(rktID) < 2 {
		return nil
	}

	rktID = strings.Split(rktID[1], "/stats")
	if len(rktID) < 1 {
		return nil
	}

	cmdStr = "rkt status " + rktID[0]

	err = utils.Run(exec.Command("/bin/sh", "-c", cmdStr))

	return err
}

func rktCmdFetch(r *http.Request) error {
	var cmdStr string
	var imgID []string
	var imgStr string

	requestBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logrus.Errorf("Read request body error: %s", err)
		return err
	}

	cmdStr = strings.TrimRight(string(requestBody), "\n")
	logrus.Debugf("Transforwarding request body: %s", cmdStr)

	url := r.URL.Query()
	imgID = url["fromImage"]

	if len(imgID) < 1 {
		return nil
	} else {
		imgStr = imgID[0]
	}

	urlMatch, _ := regexp.MatchString("coreos.com", imgStr)
	if !urlMatch {
		imgStr = "docker://" + imgStr
	}

	logrus.Debugf("The image for rkt is : %s", imgStr)

	cmdStr = "rkt fetch --insecure-skip-verify " + imgStr

	err = utils.Run(exec.Command("/bin/sh", "-c", cmdStr))

	return err
}

func rktCmdEnter(r *http.Request) error {
	var cmdStr string
	var rktID []string

	requestBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logrus.Errorf("Read request body error: %s", err)
		return err
	}

	cmdStr = strings.TrimRight(string(requestBody), "\n")
	logrus.Debugf("Transforwarding request body: %s", cmdStr)

	rktID = strings.SplitAfter(r.URL.Path, "containers/")
	if len(rktID) < 2 {
		return nil
	}

	rktID = strings.Split(rktID[1], "/json")
	if len(rktID) < 1 {
		return nil
	}

	cmdStr = "rkt enter " + rktID[0] + " /bin/sh"

	err = utils.Run(exec.Command("/bin/sh", "-c", cmdStr))

	return err
}

func rktCmdExport(r *http.Request) error {
	var cmdStr string
	var rktID []string

	requestBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logrus.Errorf("Read request body error: %s", err)
		return err
	}

	cmdStr = strings.TrimRight(string(requestBody), "\n")
	logrus.Debugf("Transforwarding request body: %s", cmdStr)

	rktID = strings.SplitAfter(r.URL.Path, "images/")
	if len(rktID) < 2 {
		return nil
	}

	rktID = strings.Split(rktID[1], "/get")
	if len(rktID) < 1 {
		return nil
	}

	cmdStr = "rkt image export " + rktID[0] + " " + strings.TrimRight(string(rktID[0]), " ") + ".aci"

	err = utils.Run(exec.Command("/bin/sh", "-c", cmdStr))

	return err
}

func rktCmdCatmanifest(r *http.Request) error {
	var cmdStr string
	var rktID []string

	requestBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logrus.Errorf("Read request body error: %s", err)
		return err
	}

	cmdStr = strings.TrimRight(string(requestBody), "\n")
	logrus.Debugf("Transforwarding request body: %s", cmdStr)

	rktID = strings.SplitAfter(r.URL.Path, "images/")
	if len(rktID) < 2 {
		return nil
	}

	rktID = strings.Split(rktID[1], "/json")
	if len(rktID) < 1 {
		return nil
	}

	cmdStr = "rkt image cat-manifest " + rktID[0]

	err = utils.Run(exec.Command("/bin/sh", "-c", cmdStr))

	return err
}
