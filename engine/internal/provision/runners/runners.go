/*
2019 © Postgres.ai
*/

// Package runners provides an interface to execute commands.
package runners

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"

	"github.com/pkg/errors"
)

const (
	// LogsEnabledDefault defines logs enabling.
	LogsEnabledDefault = true

	// Hidden marks a hidden output.
	Hidden = "HIDDEN"

	sudoCmd    = "sudo"
	sudoParams = "--non-interactive"
)

// Runner runs commands.
type Runner interface {
	Run(string, ...bool) (string, error)
}

// RunnerError represents a runner error.
type RunnerError struct {
	Msg        string
	ExitStatus int
	Stderr     string
}

// NewRunnerError creates a new RunnerError instance.
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

// Error returns runner error.
func (e RunnerError) Error() string {
	return e.Msg
}

// LocalRunner represents implementation of a local runner.
type LocalRunner struct {
	UseSudo bool
}

// NewLocalRunner creates a new LocalRunner instance.
func NewLocalRunner(useSudo bool) *LocalRunner {
	r := &LocalRunner{
		UseSudo: useSudo,
	}

	return r
}

// Run executes command.
func (r *LocalRunner) Run(command string, options ...bool) (string, error) {
	command = strings.Trim(command, " \n")
	if len(command) == 0 {
		return "", errors.New("empty command")
	}

	logsEnabled := parseOptions(options...)

	logCommand := Hidden
	if logsEnabled {
		logCommand = command
		log.Dbg(fmt.Sprintf(`Run(Local): "%s"`, logCommand))
	}

	if runtime.GOOS == "windows" {
		return "", errors.New("Windows is not supported")
	}

	if r.UseSudo && !strings.HasPrefix(command, sudoCmd+" ") {
		command = fmt.Sprintf("%s %s %s", sudoCmd, sudoParams, command)
	}

	cmd := exec.Command("/bin/bash", "-c", command)

	var (
		out    bytes.Buffer
		stderr bytes.Buffer
	)

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

	if logsEnabled {
		log.Dbg(fmt.Sprintf(`Run(Local): output "%s"`, outFormatted))
	}

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
