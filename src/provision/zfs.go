/*
2019 © Postgres.ai
*/

package provision

import (
	"fmt"
	"regexp"
	"strings"

	"../util"
)

func ZfsCreateClone(r Runner, pool string, name string, snapshot string) error {
	exists, err := ZfsCloneExists(r, name)
	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	cmd := "sudo zfs clone " + pool + "@" + snapshot + " " +
		pool + "/" + name + " -o mountpoint=/" + name + " && " +
		// TODO(anatoly): Refactor using of chown.
		"sudo chown -R postgres /" + name

	out, err := r.Run(cmd)
	if err != nil {
		return fmt.Errorf("zfs clone error %v %v", err, out)
	}

	return nil
}

func ZfsDestroyClone(r Runner, pool string, name string) error {
	exists, err := ZfsCloneExists(r, name)
	if err != nil {
		return err
	}

	if !exists {
		return nil
	}

	cmd := fmt.Sprintf("sudo zfs destroy %s/%s", pool, name)

	_, err = r.Run(cmd)
	if err != nil {
		return err
	}

	return nil
}

func ZfsCloneExists(r Runner, name string) (bool, error) {
	listZfsClonesCmd := fmt.Sprintf(`sudo zfs list`)

	out, err := r.Run(listZfsClonesCmd, false)
	if err != nil {
		return false, err
	}

	return strings.Contains(out, name), nil
}

func ZfsListClones(r Runner, prefix string) ([]string, error) {
	listZfsClonesCmd := fmt.Sprintf(`sudo zfs list`)

	re := regexp.MustCompile(fmt.Sprintf(`(%s[0-9]+)`, prefix))

	out, err := r.Run(listZfsClonesCmd, false)
	if err != nil {
		return []string{}, err
	}

	return util.Unique(re.FindAllString(out, -1)), nil
}

func ZfsCreateSnapshot(r Runner, pool string, snapshot string) error {
	cmd := fmt.Sprintf("sudo zfs snapshot -r %s@%s", pool, snapshot)

	_, err := r.Run(cmd, true)
	if err != nil {
		return err
	}

	return nil
}

func ZfsRollbackSnapshot(r Runner, pool string, snapshot string) error {
	cmd := fmt.Sprintf("sudo zfs rollback -f -r %s@%s", pool, snapshot)

	_, err := r.Run(cmd, true)
	if err != nil {
		return err
	}

	return nil
}
