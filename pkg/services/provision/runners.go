/*
2019 © Postgres.ai
*/

package provision

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"

	"gitlab.com/postgres-ai/database-lab/pkg/log"

	"github.com/pkg/errors"
)

const (
	// LogsEnabledDefault defines logs enabling.
	LogsEnabledDefault = true

	// Hidden marks a hidden output.
	Hidden = "HIDDEN"
)

type Runner interface {
	Run(string, ...bool) (string, error)
}

type RunnerError struct {
	Msg        string
	ExitStatus int
	Stderr     string
}

func NewRunnerError(command string, stderr string, e error) error {
	exitStatus := 0

	switch err := e.(type) {
	case RunnerError:
		return err

	case (*exec.ExitError):
		// SO: https://stackoverflow.com/questions/10385551/get-exit-code-go.
		// The program has exited with an exit code != 0

		// This works on both Unix and Windows. Although package
		// syscall is generally platform dependent, WaitStatus is
		// defined for both Unix and Windows and in both cases has
		// an ExitStatus() method with the same signature.

		if status, ok := err.Sys().(syscall.WaitStatus); ok {
			exitStatus = status.ExitStatus()
		}
	}

	msg := fmt.Sprintf(`RunnerError(cmd="%s", inerr="%v", stderr="%s" exit="%d")`,
		command, e, stderr, exitStatus)

	return RunnerError{
		Msg:        msg,
		ExitStatus: exitStatus,
		Stderr:     stderr,
	}
}

func (e RunnerError) Error() string {
	return e.Msg
}

// Local.
type LocalRunner struct {
}

func NewLocalRunner() *LocalRunner {
	r := &LocalRunner{}

	return r
}

func (r *LocalRunner) Run(command string, options ...bool) (string, error) {
	command = strings.Trim(command, " \n")
	if len(command) == 0 {
		return "", errors.New("empty command")
	}

	logsEnabled := parseOptions(options...)

	logCommand := Hidden
	if logsEnabled {
		logCommand = command
	}

	log.Dbg(fmt.Sprintf(`Run(Local): "%s"`, logCommand))

	var out bytes.Buffer
	var stderr bytes.Buffer

	if runtime.GOOS == "windows" {
		return "", errors.New("Windows is not supported")
	}

	cmd := exec.Command("/bin/bash", "-c", command)

	cmd.Stdout = &out
	cmd.Stderr = &stderr

	// TODO(anatoly): Remove hot fix of pg_ctl endless wait.
	if strings.Contains(command, "start") {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	// Psql with the file option returns error response to stderr with
	// success exit code. In that case err will be nil, but we need
	// to treat the case as error and read proper output.
	err := cmd.Run()
	psqlErr := strings.Contains(command, "psql") && len(stderr.String()) > 0

	// TODO(anatoly): Remove hotfix.
	psqlErr = psqlErr && !strings.Contains(stderr.String(), "unable to resolve host")

	if err != nil || psqlErr {
		runnerErr := NewRunnerError(logCommand, stderr.String(), err)

		return "", runnerErr
	}

	outFormatted := strings.Trim(out.String(), " \n")

	logOut := Hidden
	if logsEnabled {
		logOut = outFormatted
	}

	log.Dbg(fmt.Sprintf(`Run(Local): output "%s"`, logOut))

	if stderrStr := stderr.String(); len(stderrStr) > 0 {
		log.Dbg("Run(Local): stderr", stderr.String())
	}

	return outFormatted, nil
}

// Utils.
func parseOptions(options ...bool) bool {
	logsEnabled := LogsEnabledDefault
	if len(options) > 0 {
		logsEnabled = options[0]
	}

	return logsEnabled
}
