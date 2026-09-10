package branch

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v2"

	"gitlab.com/postgres-ai/database-lab/v3/cmd/cli/commands"
	"gitlab.com/postgres-ai/database-lab/v3/cmd/cli/commands/config"
	"gitlab.com/postgres-ai/database-lab/v3/cmd/cli/templates"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

func TestBranchProtectionAnnotation(t *testing.T) {
	future := models.NewLocalTime(time.Date(2099, 1, 2, 12, 0, 0, 0, time.UTC))
	past := models.NewLocalTime(time.Date(2000, 1, 2, 12, 0, 0, 0, time.UTC))

	testCases := []struct {
		name   string
		branch models.BranchView
		want   string
	}{
		{name: "unprotected", branch: models.BranchView{Name: "dev"}, want: ""},
		{name: "protected forever", branch: models.BranchView{Name: "dev", Protected: true}, want: "[protected]"},
		{name: "protected until future", branch: models.BranchView{Name: "dev", Protected: true, ProtectedTill: future},
			want: "[protected until 2099-01-02T12:00:00Z]"},
		{name: "expired protection shows nothing", branch: models.BranchView{Name: "dev", Protected: true, ProtectedTill: past},
			want: ""},
		{name: "scheduled deletion", branch: models.BranchView{Name: "dev", DeleteAt: future},
			want: "[auto-delete at 2099-01-02T12:00:00Z]"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, branchProtectionAnnotation(tc.branch))
		})
	}
}

func TestDedupBranchViews(t *testing.T) {
	branches := []models.BranchView{
		{Name: "dev", Protected: true},
		{Name: "dev", Protected: false},
		{Name: "main"},
	}

	views := dedupBranchViews(branches)
	assert.Len(t, views, 2)
	assert.True(t, views["dev"].Protected, "the first occurrence of a branch name is kept")
}

func TestFormatBranchList(t *testing.T) {
	till := models.NewLocalTime(time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC))

	branches := []models.BranchView{
		{Name: "main"},
		{Name: "dev", Protected: true},
		{Name: "feature", DeleteAt: till},
	}

	out := formatBranchList("main", branches)

	assert.Contains(t, out, "* ", "the current branch is marked with a star")
	assert.Contains(t, out, "  dev  [protected]")
	assert.Contains(t, out, "[auto-delete at 2026-06-20T12:00:00Z]")
	assert.Less(t, strings.Index(out, "dev"), strings.Index(out, "feature"), "branches are listed in sorted order")
}

// newBranchApp builds the branch commands with the connection flags the CLI normally sets from the
// saved environment, and with the current-branch lookup stubbed so no action reads the invoking
// user's CLI config.
func newBranchApp(t *testing.T, serverURL string, writer io.Writer) *cli.App {
	t.Helper()

	previous := loadBranching
	loadBranching = func() (config.Branching, error) { return config.Branching{CurrentBranch: defaultBranch}, nil }

	t.Cleanup(func() { loadBranching = previous })

	return &cli.App{
		Name:     "dblab",
		Writer:   writer,
		Commands: List(),
		Flags: []cli.Flag{
			&cli.StringFlag{Name: commands.URLKey, Value: serverURL},
			&cli.StringFlag{Name: commands.TokenKey},
			&cli.BoolFlag{Name: commands.InsecureKey},
			&cli.DurationFlag{Name: commands.RequestTimeoutKey},
			&cli.StringFlag{Name: commands.FwServerURLKey},
			&cli.StringFlag{Name: commands.FwLocalPortKey},
		},
	}
}

// runBranchApp runs the branch commands against a stub server that fails every request and reports the
// request it made. The failure keeps every action from writing the local CLI config, which only happens
// after a successful response.
func runBranchApp(t *testing.T, args []string) (method, path, body string, runErr error) {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload, _ := io.ReadAll(r.Body)
		method, path, body = r.Method, r.URL.Path, string(payload)

		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	runErr = newBranchApp(t, server.URL, io.Discard).Run(append([]string{"dblab"}, args...))

	return method, path, body, runErr
}

func TestBranchCommandDispatch(t *testing.T) {
	testCases := []struct {
		name       string
		args       []string
		wantMethod string
		wantPath   string
		wantErr    string
	}{
		{name: "bare command lists", args: []string{"branch"}, wantMethod: http.MethodGet, wantPath: "/branches"},
		{name: "list subcommand lists", args: []string{"branch", "list"}, wantMethod: http.MethodGet, wantPath: "/branches"},
		{name: "ls alias lists", args: []string{"branch", "ls"}, wantMethod: http.MethodGet, wantPath: "/branches"},
		{name: "positional name creates", args: []string{"branch", "dev"}, wantMethod: http.MethodPost, wantPath: "/branch"},
		{name: "create subcommand creates", args: []string{"branch", "create", "dev"}, wantMethod: http.MethodPost,
			wantPath: "/branch"},
		{name: "delete flag deletes", args: []string{"branch", "--delete", "dev"}, wantMethod: http.MethodDelete,
			wantPath: "/branch/dev"},
		{name: "delete subcommand deletes", args: []string{"branch", "delete", "dev"}, wantMethod: http.MethodDelete,
			wantPath: "/branch/dev"},
		{name: "rm alias deletes", args: []string{"branch", "rm", "dev"}, wantMethod: http.MethodDelete, wantPath: "/branch/dev"},
		{name: "protected flag updates protection", args: []string{"branch", "--protected", "2h", "dev"},
			wantMethod: http.MethodPatch, wantPath: "/branch/dev"},
		{name: "switch subcommand resolves the branch", args: []string{"branch", "switch", "dev"}, wantMethod: http.MethodGet,
			wantPath: "/branches", wantErr: "cannot confirm if branch exists"},
		{name: "create help creates a branch named help", args: []string{"branch", "create", "help"},
			wantMethod: http.MethodPost, wantPath: "/branch"},
		{name: "delete help deletes the branch named help", args: []string{"branch", "delete", "help"},
			wantMethod: http.MethodDelete, wantPath: "/branch/help"},
		{name: "switch h resolves the branch named h", args: []string{"branch", "switch", "h"}, wantMethod: http.MethodGet,
			wantPath: "/branches", wantErr: "cannot confirm if branch exists"},
		{name: "list help lists", args: []string{"branch", "list", "help"}, wantMethod: http.MethodGet, wantPath: "/branches"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotMethod, gotPath, _, err := runBranchApp(t, tc.args)

			assert.Equal(t, tc.wantMethod, gotMethod)
			assert.Equal(t, tc.wantPath, gotPath)

			if tc.wantErr != "" {
				assert.ErrorContains(t, err, tc.wantErr)
			}
		})
	}
}

func TestBranchVerbsNeverCreateBranches(t *testing.T) {
	for _, args := range [][]string{
		{"branch", "list"},
		{"branch", "ls"},
		{"branch", "delete", "dev"},
		{"branch", "rm", "dev"},
		{"branch", "switch", "dev"},
		{"branch", "help"},
	} {
		t.Run(strings.Join(args[1:], " "), func(t *testing.T) {
			gotMethod, gotPath, gotBody, _ := runBranchApp(t, args)

			assert.False(t, gotMethod == http.MethodPost && gotPath == "/branch",
				"a verb must not reach branch creation, got body %q", gotBody)
		})
	}
}

func TestBranchNameIsRequired(t *testing.T) {
	testCases := []struct {
		name string
		args []string
	}{
		{name: "create without a name", args: []string{"branch", "create"}},
		{name: "delete without a name", args: []string{"branch", "delete"}},
		{name: "delete flag with an empty name", args: []string{"branch", "--delete", ""}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotMethod, _, _, err := runBranchApp(t, tc.args)

			assert.ErrorContains(t, err, "BRANCH_NAME is required")
			assert.Empty(t, gotMethod, "no request must be made without a name")
		})
	}
}

func TestBareFormFlagsCannotPrecedeSubcommand(t *testing.T) {
	testCases := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{name: "parent branch before create", args: []string{"branch", "--parent-branch", "main", "create", "dev"},
			wantErr: "--parent-branch cannot precede the create subcommand"},
		{name: "snapshot before create", args: []string{"branch", "--snapshot-id", "snap", "create", "dev"},
			wantErr: "--snapshot-id cannot precede the create subcommand"},
		{name: "protected before list", args: []string{"branch", "--protected", "2h", "list"},
			wantErr: "--protected cannot precede the list subcommand"},
		{name: "delete flag before delete", args: []string{"branch", "--delete", "victim", "delete", "dev"},
			wantErr: "--delete cannot precede the delete subcommand"},
		{name: "short delete flag before rm", args: []string{"branch", "-d", "victim", "rm", "dev"},
			wantErr: "--delete cannot precede the delete subcommand"},
		{name: "protected before the built-in help", args: []string{"branch", "--protected", "2h", "help"},
			wantErr: "--protected cannot precede the help subcommand"},
		{name: "delete flag before the built-in help alias", args: []string{"branch", "--delete", "victim", "h"},
			wantErr: "--delete cannot precede the help subcommand"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotMethod, _, _, err := runBranchApp(t, tc.args)

			assert.ErrorContains(t, err, tc.wantErr)
			assert.Empty(t, gotMethod, "a rejected flag must not reach the api")
		})
	}
}

func TestCreateSubcommandFlagsReachTheRequest(t *testing.T) {
	testCases := []struct {
		name     string
		args     []string
		wantBody string
	}{
		{name: "snapshot", args: []string{"branch", "create", "--snapshot-id", "snap", "dev"}, wantBody: `"snapshotID":"snap"`},
		{name: "parent branch", args: []string{"branch", "create", "--parent-branch", "other", "dev"},
			wantBody: `"baseBranch":"other"`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, gotBody, _ := runBranchApp(t, tc.args)

			assert.Contains(t, gotBody, tc.wantBody)
		})
	}
}

func TestCurrentBranchCannotBeDeleted(t *testing.T) {
	for _, args := range [][]string{{"branch", "--delete", defaultBranch}, {"branch", "delete", defaultBranch}} {
		t.Run(strings.Join(args[1:], " "), func(t *testing.T) {
			gotMethod, _, _, err := runBranchApp(t, args)

			assert.ErrorContains(t, err, "because it is the current one")
			assert.Empty(t, gotMethod, "no request must be made for the current branch")
		})
	}
}

func TestVerbNamedBranchStillCreatable(t *testing.T) {
	gotMethod, gotPath, gotBody, _ := runBranchApp(t, []string{"branch", "create", "list"})

	assert.Equal(t, http.MethodPost, gotMethod)
	assert.Equal(t, "/branch", gotPath)
	assert.Contains(t, gotBody, `"branchName":"list"`)
}

// branchHelp renders the help of the given branch invocation with the templates the CLI installs.
func branchHelp(t *testing.T, args ...string) string {
	t.Helper()

	previousCommand, previousSubcommand := cli.CommandHelpTemplate, cli.SubcommandHelpTemplate
	cli.CommandHelpTemplate, cli.SubcommandHelpTemplate = templates.CustomCommandHelpTemplate, templates.CustomSubcommandHelpTemplate

	t.Cleanup(func() { cli.CommandHelpTemplate, cli.SubcommandHelpTemplate = previousCommand, previousSubcommand })

	out := &strings.Builder{}

	assert.NoError(t, newBranchApp(t, "http://127.0.0.1:1", out).Run(append([]string{"dblab", "branch"}, args...)))

	return out.String()
}

func TestBranchHelpDescribesBareForm(t *testing.T) {
	out := branchHelp(t, "--help")

	assert.Contains(t, out, "USAGE:\n   dblab branch command [command options] [BRANCH_NAME]")
	assert.Contains(t, out, "DESCRIPTION:\n   Without arguments, lists branches.")
	assert.Contains(t, out, "--delete value, -d value")
	assert.Contains(t, out, "delete, rm")
	assert.Contains(t, out, "delete a branch")
}

func TestBranchLeafHelpHasNoCommandsBlock(t *testing.T) {
	for _, args := range [][]string{{"create", "--help"}, {"delete", "-h"}, {"help", "delete"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			out := branchHelp(t, args...)

			assert.Regexp(t, `^USAGE:\n   dblab branch (create|delete) \[command options\] BRANCH_NAME\n`, out)
			assert.NotContains(t, out, "COMMANDS:")
		})
	}
}
