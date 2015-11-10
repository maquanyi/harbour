// Operations to convert docker client cmd to rkt cmd.

package adaptor

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

type Err struct {
	Message string
	File    string
	Path    string
	Func    string
	Line    int
}

func (e *Err) Error() string {
	return fmt.Sprintf("[%v:%v] %v", e.File, e.Line, e.Message)
}

func runRkt(urlPath string) error {
	err := run(exec.Command("rkt", "--insecure-skip-verify", "--interactive", "--mds-register=false", "run", urlPath))
	return err
}

func run(cmd *exec.Cmd) error {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return errorf(err.Error())
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return errorf(err.Error())
	}
	go io.Copy(os.Stdout, stdout)
	go io.Copy(os.Stderr, stderr)
	return cmd.Run()
}

func errorf(format string, args ...interface{}) error {
	msg := fmt.Sprintf(format, args...)
	pc, filePath, lineNo, ok := runtime.Caller(1)
	if !ok {
		return &Err{
			Message: msg,
			File:    "unknown_file",
			Path:    "unknown_path",
			Func:    "unknown_func",
			Line:    0,
		}
	}
	return &Err{
		Message: msg,
		File:    filepath.Base(filePath),
		Path:    filePath,
		Func:    runtime.FuncForPC(pc).Name(),
		Line:    lineNo,
	}
}
