/*
2026 © Postgres.ai
*/

// Package teleport provides the teleport sidecar CLI command.
package teleport

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"
)

const tctlCommandTimeout = 30 * time.Second

// label keys managed by the sidecar. labelInstance identifies the owning
// DBLab instance and is used to reconcile only this instance's resources,
// independently of the operator-facing labelEnvironment.
const (
	labelDBLab       = "dblab"
	labelInstance    = "dblab_instance"
	labelCloneID     = "clone_id"
	labelUser        = "dblab_user"
	labelEnvironment = "environment"
)

// reservedLabels are set by the sidecar and must not be overridden by
// operator-supplied custom labels.
var reservedLabels = map[string]bool{
	labelDBLab:    true,
	labelInstance: true,
	labelCloneID:  true,
	labelUser:     true,
}

// safeYAMLValue matches strings safe to embed in YAML: alphanumeric,
// hyphens, underscores, dots, colons, forward slashes, and at signs.
var safeYAMLValue = regexp.MustCompile(`^[a-zA-Z0-9._:/@\-]+$`)

// safeLabelKey matches characters allowed in Teleport label keys.
var safeLabelKey = regexp.MustCompile(`^[a-zA-Z0-9._/\-]+$`)

// sanitizeYAMLValue validates that a string is safe to embed in a YAML value.
// It rejects values containing characters that could break YAML structure
// (newlines, quotes, braces, etc.) to prevent YAML injection.
func sanitizeYAMLValue(value, fieldName string) (string, error) {
	if value == "" {
		return "", fmt.Errorf("%s must not be empty", fieldName)
	}

	if !safeYAMLValue.MatchString(value) {
		return "", fmt.Errorf(
			"%s contains invalid characters: only alphanumeric, hyphens, underscores, dots, colons, slashes are allowed",
			fieldName,
		)
	}

	return value, nil
}

// sanitizeLabelKey validates that a string is a valid Teleport label key.
func sanitizeLabelKey(key string) (string, error) {
	if key == "" {
		return "", fmt.Errorf("label key must not be empty")
	}

	if !safeLabelKey.MatchString(key) {
		return "", fmt.Errorf("label key %q contains invalid characters", key)
	}

	return key, nil
}

// baseLabels builds the labels common to every DBLab-managed resource:
// operator-supplied custom labels merged with the reserved markers. The
// environment label defaults to the instance ID when the operator did not
// provide one, preserving backwards-compatible behaviour.
func baseLabels(envID string, custom map[string]string) map[string]string {
	labels := make(map[string]string, len(custom)+len(reservedLabels))

	for k, v := range custom {
		labels[k] = v
	}

	labels[labelDBLab] = "true"
	labels[labelInstance] = envID

	if _, ok := labels[labelEnvironment]; !ok {
		labels[labelEnvironment] = envID
	}

	return labels
}

// dbResourceLabels builds the full label set for a clone's DB resource.
func dbResourceLabels(res dbResource, custom map[string]string) map[string]string {
	labels := baseLabels(res.EnvID, custom)
	labels[labelCloneID] = res.CloneID

	if res.Username != "" {
		labels[labelUser] = res.Username
	}

	return labels
}

// renderLabels renders a sanitized, deterministically ordered YAML label block
// (each line indented for the metadata.labels section).
func renderLabels(labels map[string]string) (string, error) {
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	var b strings.Builder

	for _, key := range keys {
		safeKey, err := sanitizeLabelKey(key)
		if err != nil {
			return "", err
		}

		safeValue, err := sanitizeYAMLValue(labels[key], key)
		if err != nil {
			return "", err
		}

		fmt.Fprintf(&b, "    %s: \"%s\"\n", safeKey, safeValue)
	}

	return b.String(), nil
}

// tctlDB represents a Teleport DB resource as returned by tctl get db --format=json.
type tctlDB struct {
	Metadata struct {
		Name   string            `json:"name"`
		Labels map[string]string `json:"labels"`
	} `json:"metadata"`
}

// dbResource holds parameters for creating a Teleport DB resource.
type dbResource struct {
	Name     string
	Port     int
	EnvID    string
	CloneID  string
	Username string
}

func runTctl(ctx context.Context, tctlPath, identityFile, proxyAddr string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, tctlCommandTimeout)
	defer cancel()

	baseArgs := []string{"--identity", identityFile, "--auth-server", proxyAddr}
	fullArgs := append(baseArgs, args...)

	out, err := exec.CommandContext(ctx, tctlPath, fullArgs...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("tctl %s failed: %w: %s", strings.Join(args, " "), err, string(out))
	}

	return out, nil
}

func createDB(ctx context.Context, cfg *Config, res dbResource) error {
	yaml, err := buildDBYAML(res, cfg.Labels)
	if err != nil {
		return err
	}

	return runTctlCreate(ctx, cfg, yaml)
}

// buildDBYAML produces the Teleport DB resource YAML for a clone.
func buildDBYAML(res dbResource, custom map[string]string) ([]byte, error) {
	safeName, err := sanitizeYAMLValue(res.Name, "name")
	if err != nil {
		return nil, fmt.Errorf("invalid db resource: %w", err)
	}

	labelBlock, err := renderLabels(dbResourceLabels(res, custom))
	if err != nil {
		return nil, fmt.Errorf("invalid db resource: %w", err)
	}

	yaml := fmt.Sprintf(`kind: db
version: v3
metadata:
  name: "%s"
  labels:
%sspec:
  protocol: postgres
  uri: "127.0.0.1:%d"
  tls:
    mode: insecure
`, safeName, labelBlock, res.Port)

	return []byte(yaml), nil
}

func removeDB(ctx context.Context, cfg *Config, name string) error {
	if _, err := sanitizeYAMLValue(name, "name"); err != nil {
		return fmt.Errorf("invalid db resource: %w", err)
	}

	_, err := runTctl(ctx, cfg.TctlPath, cfg.TeleportIdentity, cfg.TeleportProxy, "rm", fmt.Sprintf("db/%s", name))

	return err
}

func createApp(ctx context.Context, cfg *Config, name, uri, envID string) error {
	yaml, err := buildAppYAML(name, uri, envID, cfg.Labels)
	if err != nil {
		return err
	}

	return runTctlCreate(ctx, cfg, yaml)
}

// buildAppYAML produces the Teleport app resource YAML for a DBLab UI app.
func buildAppYAML(name, uri, envID string, custom map[string]string) ([]byte, error) {
	safeName, err := sanitizeYAMLValue(name, "name")
	if err != nil {
		return nil, fmt.Errorf("invalid app resource: %w", err)
	}

	safeURI, err := sanitizeYAMLValue(uri, "uri")
	if err != nil {
		return nil, fmt.Errorf("invalid app resource: %w", err)
	}

	labelBlock, err := renderLabels(baseLabels(envID, custom))
	if err != nil {
		return nil, fmt.Errorf("invalid app resource: %w", err)
	}

	yaml := fmt.Sprintf(`kind: app
version: v3
metadata:
  name: "%s"
  labels:
%sspec:
  uri: "%s"
  insecure_skip_verify: true # DBLab engine uses a self-signed certificate; connection is localhost-only
`, safeName, labelBlock, safeURI)

	return []byte(yaml), nil
}

func listDBs(ctx context.Context, cfg *Config) (map[string]bool, error) {
	out, err := runTctl(ctx, cfg.TctlPath, cfg.TeleportIdentity, cfg.TeleportProxy, "get", "db", "--format=json")
	if err != nil {
		return nil, err
	}

	return parseListDBsOutput(out, cfg.EnvironmentID)
}

// parseListDBsOutput filters tctl JSON output for DBLab-managed databases
// matching the given environment.
func parseListDBsOutput(data []byte, envID string) (map[string]bool, error) {
	var dbs []tctlDB
	if err := json.Unmarshal(data, &dbs); err != nil {
		return nil, fmt.Errorf("failed to parse tctl output: %w", err)
	}

	result := make(map[string]bool, len(dbs))

	for _, db := range dbs {
		if ownedByInstance(db.Metadata.Labels, envID) {
			result[db.Metadata.Name] = true
		}
	}

	return result, nil
}

// ownedByInstance reports whether a Teleport DB resource is managed by this
// DBLab instance. Resources created before the dblab_instance label existed
// are matched by the legacy environment label.
func ownedByInstance(labels map[string]string, envID string) bool {
	if labels[labelDBLab] != "true" {
		return false
	}

	if labels[labelInstance] == envID {
		return true
	}

	return labels[labelInstance] == "" && labels[labelEnvironment] == envID
}

func appExists(ctx context.Context, cfg *Config, name string) (bool, error) {
	_, err := runTctl(ctx, cfg.TctlPath, cfg.TeleportIdentity, cfg.TeleportProxy, "get", fmt.Sprintf("app/%s", name), "--format=json")
	if err == nil {
		return true, nil
	}

	if strings.Contains(err.Error(), "not found") {
		return false, nil
	}

	return false, err
}

func runTctlCreate(ctx context.Context, cfg *Config, yamlData []byte) error {
	baseArgs := []string{
		"--identity", cfg.TeleportIdentity,
		"--auth-server", cfg.TeleportProxy,
		"create", "-f", "-",
	}

	cmd := exec.CommandContext(ctx, cfg.TctlPath, baseArgs...)
	cmd.Stdin = bytes.NewReader(yamlData)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("tctl create failed: %w: %s", err, string(out))
	}

	return nil
}
