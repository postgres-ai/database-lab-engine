/*
2019 © Postgres.ai
*/

package provision

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"../log"
)

const (
	STATE_DIR  = "joe-run"
	STATE_FILE = "joestate.json"
)

func getSockFilePath() string {
	bindir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	dir, _ := filepath.Abs(filepath.Dir(bindir))
	return dir + string(os.PathSeparator) + STATE_DIR + string(os.PathSeparator) + "sshsock"
}

// TODO(anatoly): Use runners.

// Establish SSH tunnel.
func OpenSshTunnel(ip string, port uint, keyPath string) error {
	var err error

	sockFilePath := getSockFilePath()

	cmd := "ssh -o 'StrictHostKeyChecking no' -i " + keyPath +
		" -f -N -M -S " + sockFilePath + " -L " +
		strconv.FormatUint(uint64(port), 10) + ":localhost:5432" +
		" ubuntu@" + ip + " &"

	_, err = runCommand(cmd, false)
	if err == nil {
		log.Dbg("Opening SSH tunnel " + log.OK)
	} else {
		log.Dbg("Opening SSH tunnel " + log.FAIL)
	}

	return err
}

// Close SSH tunnel.
func CloseSshTunnel(ip string) error {
	var err error

	sockFilePath := getSockFilePath()

	cmd := "ssh -S " + sockFilePath + " -O exit ubuntu@" + ip + " &"

	_, err = runCommand(cmd, false)
	if err == nil {
		log.Dbg("Closing SSH tunnel " + log.OK)
	} else {
		log.Dbg("Closing SSH tunnel " + log.FAIL)
	}

	return err
}

// Check SSH tunnel existance.
func SshTunnelExists(dbUsername string, dbPassword string, port uint) bool {
	log.Dbg("Checking SSH tunnel...")

	cmd := "PGPASSWORD=" + dbPassword +
		" psql -t -q -h localhost -p " +
		strconv.FormatUint(uint64(port), 10) +
		" --user=" + dbUsername +
		" postgres -c \"select '1';\" || echo 0"

	outb, err := runCommand(cmd, false)
	out := strings.Trim(string(outb), "\n ")

	if err != nil || out != "1" {
		log.Err("Check SSH tunnel:", err, out)
		return false
	}

	return true
}

func runCommand(command string, debug bool) (string, error) {
	if debug {
		log.Dbg(fmt.Sprintf(">> Exec: %s", command))
	}

	var out bytes.Buffer
	var stderr bytes.Buffer

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", command)
	} else {
		cmd = exec.Command("/bin/bash", "-c", command)
	}

	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		log.Dbg(fmt.Sprintf(">> Error: %v Output: %s", err, stderr.String()))
		return "", fmt.Errorf("RunCommand \"%s\" error: %v %v", command, err, stderr)
	}

	if debug {
		log.Dbg(fmt.Sprintf(">> Output: %s", out.String()))
	}

	return out.String(), nil
}
