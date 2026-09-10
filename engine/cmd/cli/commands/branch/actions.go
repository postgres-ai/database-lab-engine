/*
2022 © Postgres.ai
*/

// Package branch provides commands to manage DLE branches.
package branch

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
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

// create a new template and parse the letter into it.
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

// bareFormFlag is a flag of the bare `dblab branch` form, with the spelling to use instead when a
// subcommand is given. None of these flags may gain an EnvVars or FilePath source: the guard below
// reads IsSet, which is also true for a value that did not come from the command line.
type bareFormFlag struct {
	name, hint string
}

var bareFormFlags = []bareFormFlag{
	{name: "delete", hint: "use either `dblab branch --delete BRANCH_NAME` or `dblab branch delete BRANCH_NAME`"},
	{name: "parent-branch", hint: "place it after the subcommand: `dblab branch create --parent-branch VALUE BRANCH_NAME`"},
	{name: "snapshot-id", hint: "place it after the subcommand: `dblab branch create --snapshot-id VALUE BRANCH_NAME`"},
	{name: "protected", hint: "it applies to the bare form only: `dblab branch --protected VALUE BRANCH_NAME`"},
}

// rejectBareFormFlags runs before `dblab branch` dispatches, and fails when a bare-form flag is
// followed by a subcommand, the built-in help included. urfave/cli resolves the subcommand before
// the bare-form action runs, so the flag would otherwise be parsed and then dropped:
// `dblab branch --snapshot-id X create dev` would create from the wrong snapshot and
// `dblab branch --protected 2h list` would list without changing any protection.
func rejectBareFormFlags(cliCtx *cli.Context) error {
	subcommand := subcommandNamed(cliCtx.Command, cliCtx.Args().First())
	if subcommand == nil {
		return nil
	}

	for _, flag := range bareFormFlags {
		if !cliCtx.IsSet(flag.name) {
			continue
		}

		return commands.NewActionError(fmt.Sprintf("--%s cannot precede the %s subcommand; %s",
			flag.name, subcommand.Name, flag.hint))
	}

	return nil
}

func subcommandNamed(command *cli.Command, name string) *cli.Command {
	if command == nil || name == "" {
		return nil
	}

	for _, subcommand := range command.Subcommands {
		if subcommand.HasName(name) {
			return subcommand
		}
	}

	return nil
}

// branchAction dispatches the bare `dblab branch` form: --protected updates the protection of the
// named branch, a positional name creates a branch, --delete removes one, and no arguments lists
// them. The list, create, delete and switch subcommands express the same operations unambiguously
// and take precedence, so a name that collides with one of them never silently creates a branch.
func branchAction(cliCtx *cli.Context) error {
	branchName := cliCtx.Args().First()

	// update branch protection.
	if cliCtx.IsSet("protected") {
		if branchName == "" {
			return commands.NewActionError("BRANCH_NAME is required to update protection")
		}

		return updateBranch(cliCtx)
	}

	// create a new branch.
	if branchName != "" {
		return create(cliCtx)
	}

	// delete branch. an explicitly empty name reaches the guard in deleteBranch.
	if cliCtx.IsSet("delete") {
		return deleteBranch(cliCtx)
	}

	return listBranches(cliCtx)
}

func listBranches(cliCtx *cli.Context) error {
	dblabClient, err := commands.ClientByCLIContext(cliCtx)
	if err != nil {
		return err
	}

	branches, err := dblabClient.ListBranchesView(cliCtx.Context)
	if err != nil {
		return err
	}

	if len(branches) == 0 {
		_, err = fmt.Fprintln(cliCtx.App.Writer, "No branches found")
		return err
	}

	formatted := formatBranchList(getBaseBranch(cliCtx), branches)

	_, err = fmt.Fprint(cliCtx.App.Writer, formatted)

	return err
}

func formatBranchList(baseBranch string, branches []models.BranchView) string {
	views := dedupBranchViews(branches)

	names := make([]string, 0, len(views))
	for name := range views {
		names = append(names, name)
	}

	sort.Strings(names)

	s := strings.Builder{}

	for _, name := range names {
		prefixStar := "  "
		display := name

		if baseBranch == name {
			prefixStar = "* "
			display = "\033[1;32m" + name + "\033[0m"
		}

		s.WriteString(prefixStar + display)

		if annotation := branchProtectionAnnotation(views[name]); annotation != "" {
			s.WriteString("  " + annotation)
		}

		s.WriteString("\n")
	}

	return s.String()
}

// dedupBranchViews keeps the first view per branch name, since a branch may be reported once
// per pool.
func dedupBranchViews(branches []models.BranchView) map[string]models.BranchView {
	views := make(map[string]models.BranchView, len(branches))

	for _, branch := range branches {
		if _, ok := views[branch.Name]; !ok {
			views[branch.Name] = branch
		}
	}

	return views
}

// branchProtectionAnnotation renders the protection or scheduled-deletion state of a branch for
// the list output, or an empty string when neither is active. It uses the expiry-aware
// IsProtected so an expired timed protection is not shown as still protecting.
func branchProtectionAnnotation(branch models.BranchView) string {
	if branch.IsProtected() && branch.ProtectedTill != nil {
		return "[protected until " + branch.ProtectedTill.Format(time.RFC3339) + "]"
	}

	if branch.IsProtected() {
		return "[protected]"
	}

	if branch.DeleteAt != nil {
		return "[auto-delete at " + branch.DeleteAt.Format(time.RFC3339) + "]"
	}

	return ""
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
	branchName := cliCtx.Args().First()

	if branchName == "" {
		return commands.NewActionError("BRANCH_NAME is required to create a branch")
	}

	dblabClient, err := commands.ClientByCLIContext(cliCtx)
	if err != nil {
		return err
	}

	baseBranch := cliCtx.String("parent-branch")
	snapshotID := cliCtx.String("snapshot-id")

	if baseBranch != "" && snapshotID != "" {
		return commands.NewActionError("either --parent-branch or --snapshot-id must be specified")
	}

	if baseBranch == "" {
		baseBranch = getBaseBranch(cliCtx)
	}

	branchRequest := types.BranchCreateRequest{
		BranchName: branchName,
		BaseBranch: baseBranch,
		SnapshotID: snapshotID,
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

func updateBranch(cliCtx *cli.Context) error {
	dblabClient, err := commands.ClientByCLIContext(cliCtx)
	if err != nil {
		return err
	}

	protected, duration, err := commands.ParseProtectedFlag(cliCtx)
	if err != nil {
		return err
	}

	branchName := cliCtx.Args().First()

	updateRequest := types.BranchUpdateRequest{
		Protected:                 protected,
		ProtectionDurationMinutes: duration,
	}

	branch, err := dblabClient.UpdateBranch(cliCtx.Context, branchName, updateRequest)
	if err != nil {
		return err
	}

	commandResponse, err := json.MarshalIndent(branch, "", "    ")
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(cliCtx.App.Writer, string(commandResponse))

	return err
}

func getBaseBranch(cliCtx *cli.Context) string {
	baseBranch := cliCtx.String(commands.CurrentBranch)

	if baseBranch == "" {
		baseBranch = defaultBranch
	}

	return baseBranch
}

// deleteBranch removes the branch named by the --delete flag of the bare `dblab branch` form, or by
// the positional argument of the `delete` subcommand.
func deleteBranch(cliCtx *cli.Context) error {
	branchName := cliCtx.String("delete")
	if branchName == "" {
		branchName = cliCtx.Args().First()
	}

	if branchName == "" {
		return commands.NewActionError("BRANCH_NAME is required to delete a branch")
	}

	dblabClient, err := commands.ClientByCLIContext(cliCtx)
	if err != nil {
		return err
	}

	branching, err := loadBranching()
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

// loadBranching reads the branching state of the current environment. It is a variable so tests can
// run deleteBranch without the invoking user's CLI config.
var loadBranching = getBranchingFromEnv

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
