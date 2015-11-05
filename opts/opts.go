package opts

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/huawei-openlab/harbour/mflag"
)

const (
	DEFAULTHTTPHOST     = "127.0.0.1"
	DEFAULTUNIXSOCKET   = "/var/run/docker.sock"
	DEFAULTDOCKERSOCKET = "/var/run/docker-real.sock"
	DEFAULTRUNTIME      = "docker"
)

type ListOpts struct {
	values    *[]string
	validator func(val string) (string, error)
}

func (opts *ListOpts) String() string {
	return fmt.Sprintf("%v", []string((*opts.values)))
}

// Set validates if needed the input value and add it to the
// internal slice.
func (opts *ListOpts) Set(value string) error {
	if opts.validator != nil {
		v, err := opts.validator(value)
		if err != nil {
			return err
		}
		value = v
	}
	(*opts.values) = append((*opts.values), value)
	return nil
}

func ParseUnixAddr(addr string, defaultAddr string) (string, error) {
	addr = strings.TrimPrefix(addr, "unix://")
	if strings.Contains(addr, "://") {
		return "", fmt.Errorf("Invalid proto, expected unix: %s", addr)
	}
	if addr == "" {
		addr = defaultAddr
	}
	return fmt.Sprintf("unix://%s", addr), nil
}

func ParseTCPAddr(addr string, defaultAddr string) (string, error) {
	addr = strings.TrimPrefix(addr, "tcp://")
	if strings.Contains(addr, "://") || addr == "" {
		return "", fmt.Errorf("Invalid proto, expected tcp: %s", addr)
	}

	hostParts := strings.Split(addr, ":")
	if len(hostParts) != 2 {
		return "", fmt.Errorf("Invalid bind address format: %s", addr)
	}
	host := hostParts[0]
	if host == "" {
		host = defaultAddr
	}

	p, err := strconv.Atoi(hostParts[1])
	if err != nil && p == 0 {
		return "", fmt.Errorf("Invalid bind address format: %s", addr)
	}
	return fmt.Sprintf("tcp://%s:%d", host, p), nil
}

func ParseHost(defaultTCPAddr, defaultUnixAddr, addr string) (string, error) {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		addr = fmt.Sprintf("unix://%s", defaultUnixAddr)
	}
	addrParts := strings.Split(addr, "://")
	if len(addrParts) == 1 {
		addrParts = []string{"tcp", addrParts[0]}
	}

	switch addrParts[0] {
	case "tcp":
		return ParseTCPAddr(addrParts[1], defaultTCPAddr)
	case "unix":
		return ParseUnixAddr(addrParts[1], defaultUnixAddr)
	case "fd":
		return addr, nil
	default:
		return "", fmt.Errorf("Invalid bind address format: %s", addr)
	}
}

func ValidateHost(val string) (string, error) {
	host, err := ParseHost(DEFAULTHTTPHOST, DEFAULTUNIXSOCKET, val)
	if err != nil {
		return val, err
	}
	return host, nil
}

func HostListVar(values *[]string, names []string, usage string) {
	mflag.Var(&ListOpts{values: values, validator: ValidateHost}, names, usage)
}
