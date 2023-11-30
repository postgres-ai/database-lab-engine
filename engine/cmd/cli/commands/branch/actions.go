/*
2022 © Postgres.ai
*/

// Package branch provides commands to manage DLE branches.
package branch

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/urfave/cli/v2"

	"gitlab.com/postgres-ai/database-lab/v3/cmd/cli/commands"
	"gitlab.com/postgres-ai/database-lab/v3/cmd/cli/commands/config"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/client/dblabapi/types"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util"
)

const (
	defaultBranch = "main"

	snapshotTemplate = `{{range .}}snapshot {{.ID}} {{.Branch | formatBranch}}
DataStateAt: {{.DataStateAt | formatDSA }}{{if and (ne .Message "-") (ne .Message "")}}
    {{.Message}}{{end}}

{{end}}`
)

// Create a new template and parse the letter into it.
var logTemplate = template.Must(template.New("branchLog").Funcs(
	template.FuncMap{
		"formatDSA": func(dsa string) string {
			p, err := time.Parse(util.DataStateAtFormat, dsa)
			if err != nil {
				return ""
			}
			return p.Format(time.RFC1123Z)
		},
		"formatBranch": func(dsa []string) string {
			if len(dsa) == 0 {
				return ""
			}

			return "(HEAD -> " + strings.Join(dsa, ", ") + ")"
		},
	}).Parse(snapshotTemplate))

func switchLocalContext(branchName string) error {
	dirname, err := config.GetDirname()
	if err != nil {
		return err
	}

	filename := config.BuildFileName(dirname)

	cfg, err := config.Load(filename)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	if len(cfg.Environments) == 0 {
		return errors.New("no environments found. Use `dblab init` to create a new environment before branching")
	}

	currentEnv := cfg.Environments[cfg.CurrentEnvironment]
	currentEnv.Branching.CurrentBranch = branchName

	cfg.Environments[cfg.CurrentEnvironment] = currentEnv

	if err := config.SaveConfig(filename, cfg); err != nil {
		return commands.ToActionError(err)
	}

	return err
}

func list(cliCtx *cli.Context) error {
	dblabClient, err := commands.ClientByCLIContext(cliCtx)
	if err != nil {
		return err
	}

	// Create a new branch.
	if branchName := cliCtx.Args().First(); branchName != "" {
		return create(cliCtx)
	}

	// Delete branch.
	if branchName := cliCtx.String("delete"); branchName != "" {
		return deleteBranch(cliCtx)
	}

	// List branches.
	branches, err := dblabClient.ListBranches(cliCtx.Context)
	if err != nil {
		return err
	}

	if len(branches) == 0 {
		_, err = fmt.Fprintln(cliCtx.App.Writer, "No branches found")
		return err
	}

	formatted := formatBranchList(cliCtx, branches)

	_, err = fmt.Fprint(cliCtx.App.Writer, formatted)

	return err
}

func formatBranchList(cliCtx *cli.Context, branches []string) string {
	baseBranch := getBaseBranch(cliCtx)

	s := strings.Builder{}

	for _, branch := range branches {
		var prefixStar = "  "

		if baseBranch == branch {
			prefixStar = "* "
			branch = "\033[1;32m" + branch + "\033[0m"
		}

		s.WriteString(prefixStar + branch + "\n")
	}

	return s.String()
}

func switchBranch(cliCtx *cli.Context) error {
	branchName := cliCtx.Args().First()

	if branchName == "" {
		return errors.New("branch name must not be empty")
	}

	if err := isBranchExist(cliCtx, branchName); err != nil {
		return fmt.Errorf("cannot confirm if branch exists: %w", err)
	}

	if err := switchLocalContext(branchName); err != nil {
		return commands.ToActionError(err)
	}

	_, err := fmt.Fprintf(cliCtx.App.Writer, "Switched to branch '%s'\n", branchName)

	return err
}

func isBranchExist(cliCtx *cli.Context, branchName string) error {
	dblabClient, err := commands.ClientByCLIContext(cliCtx)
	if err != nil {
		return err
	}

	branches, err := dblabClient.ListBranches(cliCtx.Context)
	if err != nil {
		return err
	}

	for _, branch := range branches {
		if branch == branchName {
			return nil
		}
	}

	return fmt.Errorf("invalid reference: %s", branchName)
}

func create(cliCtx *cli.Context) error {
	dblabClient, err := commands.ClientByCLIContext(cliCtx)
	if err != nil {
		return err
	}

	branchName := cliCtx.Args().First()

	branchRequest := types.BranchCreateRequest{
		BranchName: branchName,
		BaseBranch: getBaseBranch(cliCtx),
	}

	branch, err := dblabClient.CreateBranch(cliCtx.Context, branchRequest)
	if err != nil {
		return err
	}

	if err := switchLocalContext(branchName); err != nil {
		return commands.ToActionError(err)
	}

	_, err = fmt.Fprintf(cliCtx.App.Writer, "Switched to new branch '%s'\n", branch.Name)

	return err
}

func getBaseBranch(cliCtx *cli.Context) string {
	baseBranch := cliCtx.String(commands.CurrentBranch)

	if baseBranch == "" {
		baseBranch = defaultBranch
	}

	return baseBranch
}

func deleteBranch(cliCtx *cli.Context) error {
	dblabClient, err := commands.ClientByCLIContext(cliCtx)
	if err != nil {
		return err
	}

	branchName := cliCtx.String("delete")

	branching, err := getBranchingFromEnv()
	if err != nil {
		return err
	}

	if branching.CurrentBranch == branchName {
		return fmt.Errorf("cannot delete branch %q because it is the current one", branchName)
	}

	if err = dblabClient.DeleteBranch(cliCtx.Context, types.BranchDeleteRequest{
		BranchName: branchName,
	}); err != nil {
		return err
	}

	if err := switchLocalContext(defaultBranch); err != nil {
		return commands.ToActionError(err)
	}

	_, err = fmt.Fprintf(cliCtx.App.Writer, "Deleted branch '%s'\n", branchName)

	return err
}

func commit(cliCtx *cli.Context) error {
	dblabClient, err := commands.ClientByCLIContext(cliCtx)
	if err != nil {
		return err
	}

	cloneID := cliCtx.String("clone-id")
	message := cliCtx.String("message")

	snapshotRequest := types.SnapshotCloneCreateRequest{
		CloneID: cloneID,
		Message: message,
	}

	snapshot, err := dblabClient.CreateSnapshotForBranch(cliCtx.Context, snapshotRequest)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(cliCtx.App.Writer, "Created new snapshot '%s'\n", snapshot.SnapshotID)

	return err
}

func history(cliCtx *cli.Context) error {
	dblabClient, err := commands.ClientByCLIContext(cliCtx)
	if err != nil {
		return err
	}

	branchName := cliCtx.Args().First()

	if branchName == "" {
		branchName = getBaseBranch(cliCtx)
	}

	logRequest := types.LogRequest{BranchName: branchName}

	snapshots, err := dblabClient.BranchLog(cliCtx.Context, logRequest)
	if err != nil {
		return err
	}

	formattedLog, err := formatSnapshotLog(snapshots)
	if err != nil {
		return err
	}

	_, err = fmt.Fprint(cliCtx.App.Writer, formattedLog)

	return err
}

func getBranchingFromEnv() (config.Branching, error) {
	branching := config.Branching{}

	dirname, err := config.GetDirname()
	if err != nil {
		return branching, err
	}

	filename := config.BuildFileName(dirname)

	cfg, err := config.Load(filename)
	if err != nil && !os.IsNotExist(err) {
		return branching, err
	}

	if len(cfg.Environments) == 0 {
		return branching, errors.New("no environments found. Use `dblab init` to create a new environment before branching")
	}

	branching = cfg.Environments[cfg.CurrentEnvironment].Branching

	return branching, nil
}

func formatSnapshotLog(snapshots []models.SnapshotDetails) (string, error) {
	sb := &strings.Builder{}

	if err := logTemplate.Execute(sb, snapshots); err != nil {
		return "", fmt.Errorf("executing template: %w", err)
	}

	return sb.String(), nil
}
