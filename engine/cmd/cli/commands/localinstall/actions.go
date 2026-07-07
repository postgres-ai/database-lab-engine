/*
2026 © Postgres.ai
*/

package localinstall

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/urfave/cli/v2"
	"golang.org/x/term"

	"gitlab.com/postgres-ai/database-lab/v3/cmd/cli/commands"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

const logicalRetrievalMode = "logical"

// installOptions holds the resolved local-install flags.
type installOptions struct {
	sourceURL     string
	password      string
	provider      string
	dockerImage   string
	dockerTag     string
	sharedBuffers string
	dbnames       []string
	start         bool
	noStart       bool
	yes           bool
}

func optionsFromContext(c *cli.Context) installOptions {
	return installOptions{
		sourceURL:     c.String("source-url"),
		password:      c.String("password"),
		provider:      c.String("provider"),
		dockerImage:   c.String("docker-image"),
		dockerTag:     c.String("docker-tag"),
		sharedBuffers: c.String("shared-buffers"),
		dbnames:       c.StringSlice("dbname"),
		start:         c.Bool("start"),
		noStart:       c.Bool("no-start"),
		yes:           c.Bool("yes"),
	}
}

func (o installOptions) validate() error {
	if o.sourceURL == "" {
		return errors.New("--source-url is required")
	}

	if o.start && o.noStart {
		return errors.New("--start and --no-start are mutually exclusive")
	}

	return nil
}

func localInstall(cliCtx *cli.Context) error {
	opts := optionsFromContext(cliCtx)
	if err := opts.validate(); err != nil {
		return err
	}

	client, err := commands.ClientByCLIContext(cliCtx)
	if err != nil {
		return err
	}

	password, err := resolvePassword(opts.password)
	if err != nil {
		return err
	}

	opts.password = password

	ctx := cliCtx.Context
	w := cliCtx.App.Writer

	proposal, err := client.ProbeSource(ctx, models.ProbeSourceRequest{URL: opts.sourceURL, Password: opts.password})
	if err != nil {
		return err
	}

	if opts.provider != "" {
		proposal.DetectedProvider = opts.provider
	}

	_, _ = fmt.Fprintln(w, renderPreview(proposal))

	if !opts.yes {
		confirmed, err := readConfirmation(os.Stdin, w)
		if err != nil {
			return err
		}

		if !confirmed {
			_, _ = fmt.Fprintln(w, "Aborted.")
			return nil
		}
	}

	projection, err := buildProjection(proposal, opts)
	if err != nil {
		return err
	}

	status, err := client.Status(ctx)
	if err != nil {
		return err
	}

	if _, err := client.ApplyConfig(ctx, projection); err != nil {
		return err
	}

	_, _ = fmt.Fprintln(w, "Configuration applied.")

	if shouldStartRefresh(status.Retrieving.Status, opts.start, opts.noStart) {
		if _, err := client.FullRefresh(ctx); err != nil {
			return err
		}

		_, _ = fmt.Fprintln(w, "Full refresh started.")
	}

	return nil
}

// resolvePassword returns the flag value, or prompts on a TTY when it is empty.
// With no TTY and no flag it returns an empty password so the engine can fall
// back to a pre-existing config password / PGPASSWORD.
func resolvePassword(flagValue string) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}

	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return "", nil
	}

	_, _ = fmt.Fprint(os.Stderr, "Source password: ")

	raw, err := term.ReadPassword(int(os.Stdin.Fd()))

	_, _ = fmt.Fprintln(os.Stderr)

	if err != nil {
		return "", fmt.Errorf("failed to read password: %w", err)
	}

	return string(raw), nil
}

// readConfirmation reads a yes/no answer; only "y"/"yes" (case-insensitive)
// confirm.
func readConfirmation(in io.Reader, out io.Writer) (bool, error) {
	_, _ = fmt.Fprint(out, "Apply this configuration? [y/N]: ")

	line, err := bufio.NewReader(in).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, fmt.Errorf("failed to read confirmation: %w", err)
	}

	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true, nil
	}

	return false, nil
}

// shouldStartRefresh reports whether the CLI must explicitly trigger a full
// refresh. --no-start always wins. A pending instance auto-refreshes on apply,
// and an already-running refresh/snapshot would reject a second trigger, so the
// CLI skips those states; otherwise --start forces a refresh.
func shouldStartRefresh(status models.RetrievalStatus, start, noStart bool) bool {
	if noStart {
		return false
	}

	switch status {
	case models.Pending, models.Refreshing, models.Snapshotting:
		return false
	}

	return start
}

// renderPreview formats a human-readable summary of the proposal. It is built
// from the password-free ProposedConfig, so no credential can leak into output.
func renderPreview(p *models.ProposedConfig) string {
	var b strings.Builder

	b.WriteString("Proposed configuration:\n")
	fmt.Fprintf(&b, "  Source:         %s@%s:%d/%s\n", p.Source.Username, p.Source.Host, p.Source.Port, p.Source.DBName)
	fmt.Fprintf(&b, "  Provider:       %s\n", p.DetectedProvider)
	fmt.Fprintf(&b, "  PostgreSQL:     %d\n", p.PgMajorVersion)

	if p.CollationVersion != "" {
		fmt.Fprintf(&b, "  Collation:      %s\n", p.CollationVersion)
	}

	fmt.Fprintf(&b, "  Docker image:   %s\n", displayImage(p))

	if p.SharedBuffers != "" {
		fmt.Fprintf(&b, "  shared_buffers: %s\n", p.SharedBuffers)
	}

	if len(p.Databases) > 0 {
		fmt.Fprintf(&b, "  Databases:      %s\n", strings.Join(p.Databases, ", "))
	}

	if p.SharedPreloadLibraries != "" {
		fmt.Fprintf(&b, "  Preload libs:   %s\n", p.SharedPreloadLibraries)
	}

	if warning := glibcWarning(p); warning != "" {
		b.WriteString(warning)
	}

	return b.String()
}

func displayImage(p *models.ProposedConfig) string {
	if p.ResolvedImage != "" {
		return p.ResolvedImage
	}

	return "(resolved from the UI catalog)"
}

// glibcWarning warns when a PG15+ libc source has a collation version but the
// selected image is not glibc-pinned, which risks collation/index mismatch.
func glibcWarning(p *models.ProposedConfig) string {
	if p.PgMajorVersion < 15 || p.CollationVersion == "" {
		return ""
	}

	if strings.Contains(p.DockerTag, "glibc") {
		return ""
	}

	return fmt.Sprintf("  WARNING: no glibc-pinned image matched collation version %s; "+
		"using %s. Collation/index inconsistencies are possible if the container glibc "+
		"differs from the source.\n", p.CollationVersion, displayImage(p))
}

// buildProjection assembles the nested ConfigProjection JSON the engine's
// POST /admin/config expects. It mirrors the YAML paths of the projection tags.
func buildProjection(p *models.ProposedConfig, o installOptions) (json.RawMessage, error) {
	// the discrete fields are always written — overwriting any stale values left
	// in the config scaffold — so the persisted source block stays self-consistent;
	// the full connection string is added only when it carries options the discrete
	// fields cannot represent, and the engine prefers it when present.
	connection := map[string]interface{}{
		"host":     p.Source.Host,
		"port":     p.Source.Port,
		"dbname":   p.Source.DBName,
		"username": p.Source.Username,
	}

	if o.password != "" {
		connection["password"] = o.password
	}

	source := map[string]interface{}{"connection": connection}

	if sourceCarriesExtraParams(o.sourceURL) {
		source["connectionString"] = o.sourceURL
	}

	options := map[string]interface{}{"source": source}

	if databases := databaseSet(p, o.dbnames); len(databases) > 0 {
		options["databases"] = databases
	}

	projection := map[string]interface{}{
		"retrievalMode": logicalRetrievalMode,
		"retrieval": map[string]interface{}{
			"spec": map[string]interface{}{
				"logicalDump": map[string]interface{}{"options": options},
			},
		},
	}

	if image := composeImage(p.ResolvedImage, o.dockerImage, o.dockerTag); image != "" {
		projection["databaseContainer"] = map[string]interface{}{"dockerImage": image}
	}

	if sharedBuffers := firstNonEmpty(o.sharedBuffers, p.SharedBuffers); sharedBuffers != "" {
		projection["databaseConfigs"] = map[string]interface{}{
			"configs": map[string]interface{}{"shared_buffers": sharedBuffers},
		}
	}

	return json.Marshal(projection)
}

// databaseSet builds the databases map for the dump options. Explicit --dbname
// flags win; otherwise the probed databases are used. An empty result lets the
// engine enumerate every source database.
func databaseSet(p *models.ProposedConfig, dbnames []string) map[string]interface{} {
	names := dbnames
	if len(names) == 0 {
		names = p.Databases
	}

	if len(names) == 0 {
		return nil
	}

	set := make(map[string]interface{}, len(names))
	for _, name := range names {
		set[name] = map[string]interface{}{}
	}

	return set
}

// composeImage resolves the final docker image. An explicit --docker-image
// replaces the engine-resolved reference; --docker-tag then sets/replaces the
// tag on whichever base is in effect. A digest-pinned base (`...@sha256:...`)
// cannot carry a tag, so --docker-tag is ignored for it.
func composeImage(resolved, imageOverride, tagOverride string) string {
	base := resolved
	if imageOverride != "" {
		base = imageOverride
	}

	if tagOverride == "" || base == "" || strings.Contains(base, "@") {
		return base
	}

	return replaceTag(base, tagOverride)
}

// hasTag reports whether ref already carries a tag (a colon after the final
// path separator), ignoring a registry host:port.
func hasTag(ref string) bool {
	return strings.LastIndex(ref, ":") > strings.LastIndex(ref, "/")
}

func replaceTag(ref, tag string) string {
	if hasTag(ref) {
		return ref[:strings.LastIndex(ref, ":")] + ":" + tag
	}

	return ref + ":" + tag
}

// sourceCarriesExtraParams reports whether sourceURL holds libpq options beyond
// host/port/dbname/user, which can only be preserved via a connection string.
func sourceCarriesExtraParams(sourceURL string) bool {
	trimmed := strings.TrimSpace(sourceURL)

	if isURIConnString(trimmed) {
		return strings.Contains(trimmed, "?")
	}

	basic := map[string]bool{"host": true, "port": true, "dbname": true, "user": true, "password": true}

	for _, field := range strings.Fields(trimmed) {
		key, _, ok := strings.Cut(field, "=")
		if !ok {
			continue
		}

		if !basic[strings.ToLower(key)] {
			return true
		}
	}

	return false
}

func isURIConnString(s string) bool {
	lower := strings.ToLower(s)
	return strings.HasPrefix(lower, "postgresql://") || strings.HasPrefix(lower, "postgres://")
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}

	return ""
}
