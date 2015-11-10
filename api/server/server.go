package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/huawei-openlab/harbour/adaptor"
	"github.com/huawei-openlab/harbour/engine"
	"github.com/huawei-openlab/harbour/engine/trap"
	"github.com/huawei-openlab/harbour/opts"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/pkg/ioutils"
	"github.com/gorilla/mux"

	"github.com/opencontainers/runc/libcontainer/user"
)

type Server struct {
	router *mux.Router
}

type HttpServer struct {
	srv *http.Server
	l   net.Listener
}

func (s *HttpServer) Serve() error {
	return s.srv.Serve(s.l)
}
func (s *HttpServer) Close() error {
	return s.l.Close()
}

type HttpApiFunc func(eng *engine.Engine, w http.ResponseWriter, r *http.Request, vars map[string]string) error

func httpError(w http.ResponseWriter, err error) {
	if err == nil || w == nil {
		logrus.WithFields(logrus.Fields{"error": err, "writer": w}).Error("unexpected HTTP error handling")
		return
	}
	statusCode := http.StatusInternalServerError
	// FIXME: this is brittle and should not be necessary.
	// If we need to differentiate between different possible error types, we should
	// create appropriate error types with clearly defined meaning.
	errStr := strings.ToLower(err.Error())
	for keyword, status := range map[string]int{
		"not found":             http.StatusNotFound,
		"no such":               http.StatusNotFound,
		"bad parameter":         http.StatusBadRequest,
		"conflict":              http.StatusConflict,
		"impossible":            http.StatusNotAcceptable,
		"wrong login/password":  http.StatusUnauthorized,
		"hasn't been activated": http.StatusForbidden,
	} {
		if strings.Contains(errStr, keyword) {
			statusCode = status
			break
		}
	}

	logrus.WithFields(logrus.Fields{"statusCode": statusCode, "err": err}).Error("HTTP Error")
	http.Error(w, err.Error(), statusCode)
}

func makeHttpHandler(eng *engine.Engine, localMethod string, localRoute string, handlerFunc HttpApiFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// log the request
		logrus.Debugf("Calling %s %s", localMethod, localRoute)

		if err := handlerFunc(eng, w, r, mux.Vars(r)); err != nil {
			logrus.Errorf("Handler for %s %s returned error: %s", localMethod, localRoute, err)
			httpError(w, err)
		}
	}
}

func createRouter(eng *engine.Engine, srv *Server, kube bool) *mux.Router {
	r := mux.NewRouter()
	m := map[string]map[string]HttpApiFunc{
		"GET": {
			"": transForwarding,
		},
		"POST": {
			"": transForwarding,
		},
		"DELETE": {
			"": transForwarding,
		},
	}

	for method, routes := range m {
		keys := []string{}
		for route, _ := range routes {
			keys = append(keys, route)
		}
		sort.Sort(sort.Reverse(sort.StringSlice(keys)))
		for _, route := range keys {
			fct := routes[route]
			logrus.Debugf("Registering %s, %s", method, route)
			// NOTE: scope issue, make sure the variables are local and won't be changed
			localRoute := route
			localFct := fct
			localMethod := method

			// build the handler function
			f := makeHttpHandler(eng, localMethod, localRoute, localFct)

			// add the new route
			if localRoute == "" {
				r.Methods(localMethod).HandlerFunc(f)
			} else {
				r.Path(localRoute).Methods(localMethod).HandlerFunc(f)
			}
		}
	}

	return r
}

func New(eng *engine.Engine, kube bool) *Server {
	srv := &Server{}
	r := createRouter(eng, srv, kube)
	srv.router = r
	return srv
}

func (s *Server) newServer(proto, addr string) (*HttpServer, error) {
	switch proto {
	case "tcp":
		l, err := net.Listen("tcp", addr)
		if err != nil {
			return nil, err
		}
		return &HttpServer{&http.Server{Addr: addr, Handler: s.router}, l}, nil
	case "unix":
		os.Remove(addr)
		l, err := net.Listen("unix", addr)
		if err != nil {
			return nil, err
		}
		if err := setSocketGroup(addr, engine.SocketGroup); err != nil {
			l.Close()
			return nil, err
		}
		if err := os.Chmod(addr, 0660); err != nil {
			l.Close()
			return nil, err
		}
		return &HttpServer{&http.Server{Addr: addr, Handler: s.router}, l}, nil
	default:
		return nil, fmt.Errorf("Invalid protocol format.")
	}
}

func setSocketGroup(path, group string) error {
	if group == "" {
		return nil
	}
	if err := changeGroup(path, group); err != nil {
		if group != "docker" {
			return err
		}
		logrus.Debugf("Warning: could not change group %s to docker: %v", path, err)
	}
	return nil
}

func changeGroup(path string, nameOrGid string) error {
	gid, err := lookupGidByName(nameOrGid)
	if err != nil {
		return err
	}
	logrus.Debugf("%s group found. gid: %d", nameOrGid, gid)
	return os.Chown(path, 0, gid)
}

func lookupGidByName(nameOrGid string) (int, error) {
	groupFile, err := user.GetGroupPath()
	if err != nil {
		return -1, err
	}
	groups, err := user.ParseGroupFileFilter(groupFile, func(g user.Group) bool {
		return g.Name == nameOrGid || strconv.Itoa(g.Gid) == nameOrGid
	})
	if err != nil {
		return -1, err
	}
	if groups != nil && len(groups) > 0 {
		return groups[0].Gid, nil
	}
	gid, err := strconv.Atoi(nameOrGid)
	if err == nil {
		logrus.Warnf("Could not find GID %d", gid)
		return gid, nil
	}
	return -1, fmt.Errorf("Group %s not found", nameOrGid)
}

func (s *Server) CreateServer(eng *engine.Engine, protoAddrs []string) error {
	var chErrors = make(chan error, len(protoAddrs))

	for _, protoAddr := range protoAddrs {
		protoAddrParts := strings.SplitN(protoAddr, "://", 2)
		if len(protoAddrParts) != 2 {
			return fmt.Errorf("usage: %s PROTO://ADDR [PROTO://ADDR ...]")
		}
		go func() {
			logrus.Debugf("Listening for HTTP on %s (%s)", protoAddrParts[0], protoAddrParts[1])
			srv, err := s.newServer(protoAddrParts[0], protoAddrParts[1])
			if err != nil {
				chErrors <- err
				return
			}
			trap.ShutdownCallback(func() {
				if err := srv.Close(); err != nil {
					logrus.Errorln(err)
				}
			})
			if err = srv.Serve(); err != nil && strings.Contains(err.Error(), "use of closed network connection") {
				err = nil
			}
			chErrors <- err
		}()
	}

	for i := 0; i < len(protoAddrs); i++ {
		err := <-chErrors
		if err != nil {
			logrus.Errorln(err)
			return err
		}
	}

	return nil
}

func hijackServer(w http.ResponseWriter) (io.ReadCloser, io.Writer, error) {
	conn, _, err := w.(http.Hijacker).Hijack()
	if err != nil {
		return nil, nil, err
	}
	// Flush the options to make sure the client sets the raw mode
	conn.Write([]byte{})
	return conn, conn, nil
}

func closeStreams(streams ...interface{}) {
	for _, stream := range streams {
		if tcpc, ok := stream.(interface {
			CloseWrite() error
		}); ok {
			tcpc.CloseWrite()
		} else if closer, ok := stream.(io.Closer); ok {
			closer.Close()
		}
	}
}

func transForwarding(eng *engine.Engine, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	var requestBody []byte

	logrus.Debugf("Request get: %v", r)
	logrus.Debugf("Request's url :%v", r.URL)
	logrus.Debugf("Request's url path: %v", r.URL.Path)

	if engine.ContainerRuntime == opts.RKTRUNTIME {
		adaptor.ParseUserConfig(r)
		return nil
	}

	// For docker exec, we have to check if the request body contains detach flag
	// so that proper action mode can be chosen.
	execMatch, err := regexp.MatchString(".*/exec/.*/start", r.URL.Path)
	if err != nil {
		return err
	}
	if execMatch {
		var err error
		requestBody, err = ioutil.ReadAll(r.Body)
		if err != nil {
			logrus.Errorf("Read request body error: %s", err)
			return err
		}
		logrus.Debugf("Transforwarding request body: %s", strings.TrimRight(string(requestBody), "\n"))
		tempReader := bytes.NewBuffer(requestBody)
		newBody := ioutil.NopCloser(tempReader)
		r.Body = newBody
	}

	r.URL.Scheme = "http"
	r.URL.Host = "unix.sock"
	r.RequestURI = ""

	action, err := urlActionSelector(r.URL.Path, r.URL.RawQuery, requestBody)
	if err != nil {
		logrus.Errorf("UrlActionSelector error: %s", err)
		return err
	}

	switch action {
	case "stream":
		{
			logrus.Debugf("Stream mode is running")
			dial, err := net.Dial("unix", engine.DockerSock)
			if tcpConn, ok := dial.(*net.TCPConn); ok {
				tcpConn.SetKeepAlive(true)
				tcpConn.SetKeepAlivePeriod(30 * time.Second)
			}
			clientconn := httputil.NewClientConn(dial, nil)

			defer clientconn.Close()
			// Server hijacks the connection, error 'connection closed' expected
			resp, err := clientconn.Do(r)
			if err != nil {
				logrus.Errorf("Stream fail: %s", err)
				return err
			}
			if resp.Header.Get("Content-Type") == "application/json" {
				w.Header().Set("Content-Type", "application/json")
			}
			w.WriteHeader(resp.StatusCode)
			if closeNotifier, ok := w.(http.CloseNotifier); ok {
				finished := make(chan struct{})
				defer close(finished)
				go func() {
					select {
					case <-finished:
					case <-closeNotifier.CloseNotify():
						logrus.Debugf("Client disconnceted")
						clientconn.Close()
					}
				}()
			}

			outStream := ioutils.NewWriteFlusher(w)
			outStream.Write(nil)
			_, err = io.Copy(outStream, resp.Body)
			return err
		}
	case "fetchStream":
		{
			logrus.Debugf("fetchStream mode is running")

			resp, err := initClient(r)

			if err != nil {
				logrus.Errorf("fetchStream fail: %s", err)
				return err
			}

			w.WriteHeader(resp.StatusCode)

			err = copyBody(w, resp.Body)
			if err != nil {
				return err
			}
		}
	case "presistConn":
		{
			logrus.Debugf("presist mode is running")

			dial, err := net.Dial("unix", engine.DockerSock)
			if tcpConn, ok := dial.(*net.TCPConn); ok {
				tcpConn.SetKeepAlive(true)
				tcpConn.SetKeepAlivePeriod(30 * time.Second)
			}
			clientconn := httputil.NewClientConn(dial, nil)

			defer clientconn.Close()
			// Server hijacks the connection, error 'connection closed' expected
			_, err = clientconn.Do(r)
			if err != nil {
				logrus.Errorf("presistConn fail: %s", err)
				return err
			}

			inStream, outStream, err := hijackServer(w)
			if err != nil {
				return err
			}
			defer closeStreams(inStream, outStream)

			if _, ok := r.Header["Upgrade"]; ok {
				fmt.Fprintf(outStream, "HTTP/1.1 101 UPGRADED\r\nContent-Type: application/vnd.docker.raw-stream\r\nConnection: Upgrade\r\nUpgrade: tcp\r\n\r\n")
			} else {
				fmt.Fprintf(outStream, "HTTP/1.1 200 OK\r\nContent-Type: application/vnd.docker.raw-stream\r\n\r\n")
			}

			rwc, br := clientconn.Hijack()

			defer func() {
				rwc.Close()
			}()

			receiveStdout := make(chan error, 1)
			sendStdin := make(chan error, 1)

			go func() {
				logrus.Debugf("start copy")
				_, err := io.Copy(outStream, br)
				logrus.Debugf("copy end")
				receiveStdout <- err
			}()
			go func() {
				logrus.Debugf("start copy inStream")
				io.Copy(rwc, inStream)
				logrus.Debugf("end copy inStream")

				if conn, ok := rwc.(interface {
					CloseWrite() error
				}); ok {
					if err := conn.CloseWrite(); err != nil {
						//logrus.Debugf("Couldn't send EOF: %s", err)
					}
				}
				sendStdin <- nil
			}()
			if err := <-receiveStdout; err != nil {
				logrus.Debugf("Error receiveStdout: %s", err)
				return err
			}
		}
	case "other":
		{
			resp, err := initClient(r)

			if err != nil {
				return err
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(resp.StatusCode)

			stream, _ := ioutil.ReadAll(resp.Body)
			io.WriteString(w, string(stream))
			resp.Body.Close()
		}
	}

	return nil
}

//choose the type of command
func urlActionSelector(url string, query string, body []byte) (string, error) {
	//the action keywords is contained in str arrays,it has THREE models of action
	fetchStream := []string{".+/images/.+/get", ".*/images/get", ".*/containers/.*/exec", ".*/containers/.*/export"}
	presistConn := []string{".*/containers/.*/attach.*"}
	stream := []string{"/events$", ".*/containers/.*/logs", "/build$"}

	chooser := func(arrParttens []string) bool {
		for _, v := range arrParttens {
			match, err := regexp.MatchString(v, url)
			if err != nil {
				logrus.Errorf("regexp error: %s", err)
				return false
			}
			if match {
				logrus.Debugf("%s matches regexp pattern %s", url, v)
				return match
			}
		}
		return false
	}

	if chooser(fetchStream) {
		return "fetchStream", nil
	}

	if chooser(presistConn) {
		return "presistConn", nil
	}

	if chooser(stream) {
		return "stream", nil
	}

	if len(body) > 0 {
		var respBody map[string]interface{}
		if err := json.Unmarshal(body, &respBody); err != nil {
			return "", err
		}
		detachConfig, ok := respBody["Detach"]
		if !ok {
			err := errors.New("Can not find detach config in the response body")
			return "", err
		}
		if detachConfig.(bool) {
			return "fetchStream", nil
		} else {
			return "presistConn", nil
		}
	}

	// For docker stats, we have to see if the query contains "stream=1"
	statsMatch, err := regexp.MatchString(".*/containers/.*/stats", url)
	if err != nil {
		return "", err
	}
	if statsMatch {
		logrus.Debugf("%s matches regexp pattern .*/containers/.*/stats", url)
		if strings.Contains(query, "stream=1") {
			return "fetchStream", nil
		}
	}

	return "other", nil
}

//client init,return the response
func initClient(r *http.Request) (*http.Response, error) {
	unixDial := func(proto, addr string) (net.Conn, error) {
		return net.Dial("unix", engine.DockerSock)
	}
	tr := &http.Transport{
		Dial:              unixDial,
		DisableKeepAlives: true,
	}
	client := &http.Client{Transport: tr}

	resp, err := client.Do(r)
	if err != nil {
		logrus.Errorf("client do fail: %s", err)
		return nil, err
	}

	return resp, nil
}

//copy body(if its a file) to responseWriter
//usually used in download images or other files
func copyBody(w http.ResponseWriter, body io.ReadCloser) error {
	_, err := io.Copy(w, body)
	if err != nil {
		logrus.Errorf("copy action fail: %s", err)
		return err
	}
	return nil
}
