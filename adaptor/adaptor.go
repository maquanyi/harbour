package adaptor

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"

	"github.com/Sirupsen/logrus"
)

type UserConfig struct {
	Hostname string // Hostname
	Image    string // Name of the image as it was passed by the operator (eg. could be symbolic)
}

func ParseUserConfig(r *http.Request) {
	var s string
	var config UserConfig

	createMatch, _ := regexp.MatchString("/create", r.URL.Path)
	if createMatch {
		requestBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			logrus.Errorf("Read request body error: %s", err)
			return
		}
		s = strings.TrimRight(string(requestBody), "\n")
		logrus.Debugf("Transforwarding request body: %s", s)
		json.Unmarshal([]byte(s), &config)
		s = "docker://" + config.Image
	}

	runRkt(s)
}
