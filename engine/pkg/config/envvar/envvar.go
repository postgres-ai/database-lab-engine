// Package envvar expands "${VAR}" and "$VAR" placeholders in config files
// using the process environment.
//
// Expansion happens on the parsed YAML tree, before the document is decoded
// into config structs. Reaching every string scalar is the point: the retrieval
// job specs are `map[string]interface{}` decoded per job at runtime, so the
// credentials inside them cannot be named by any list of struct fields.
//
// A placeholder is recognized only when it is the whole value of a scalar. A
// value that merely contains a "$" — a password, a regex backreference such as
// "***$1", a dollar-quoted SQL block — is a literal and reaches its consumer
// byte for byte. That rule is what makes whole-document expansion safe with no
// escape syntax and no list of exempt fields.
package envvar

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"

	"gopkg.in/yaml.v3"
)

// ErrUnsetEnv is returned when a config file references an environment variable
// that is not set. Callers can test for it with errors.Is.
var ErrUnsetEnv = errors.New("required environment variable is not set")

const strTag = "!!str"

// placeholderRE matches a value that is exactly "${VAR}" or "$VAR", and nothing
// else. Anchoring both ends is the whole safety property: it is what keeps a
// literal "$" in a secret, a regex, or a SQL block from being read as a
// reference.
var placeholderRE = regexp.MustCompile(`^\$\{([A-Za-z_][A-Za-z0-9_]*)\}$|^\$([A-Za-z_][A-Za-z0-9_]*)$`)

// retypeSafeRE matches the shape of the resolved values that yaml re-resolves
// to the very same text: canonical booleans and integers. Everything else keeps
// the string tag it was parsed with, so a password of "no", "0755", or "1e5"
// cannot be silently re-typed into false, 493, or 100000 on its way into the
// config. Integers are range-checked in retypeSafe: above uint64 yaml falls
// back to a float and the value would arrive in exponent notation.
var retypeSafeRE = regexp.MustCompile(`^(true|false|0|-?[1-9][0-9]*)$`)

func retypeSafe(value string) bool {
	if !retypeSafeRE.MatchString(value) {
		return false
	}

	if value == "true" || value == "false" {
		return true
	}

	if value[0] == '-' {
		_, err := strconv.ParseInt(value, 10, 64)
		return err == nil
	}

	_, err := strconv.ParseUint(value, 10, 64)

	return err == nil
}

// IsPlaceholder reports whether s is an unresolved environment placeholder.
// Consumers that read the config file raw — the admin API does, so that saving
// it cannot persist resolved secrets — use this to tell a deferred reference
// apart from a value they can interpret.
func IsPlaceholder(s string) bool {
	return placeholderName(s) != ""
}

// placeholderName returns the variable named by s when s is a whole-value
// placeholder, and an empty string otherwise.
func placeholderName(s string) string {
	match := placeholderRE.FindStringSubmatch(s)
	if match == nil {
		return ""
	}

	if match[1] != "" {
		return match[1]
	}

	return match[2]
}

// ExpandStrict resolves s when it is a whole-value "${VAR}" or "$VAR"
// reference, and returns it unchanged otherwise. It returns an error if the
// referenced variable is unset, so a misconfigured token fails loudly at
// startup instead of silently resolving to an empty string.
//
// This is the single-value entry point, used for values that never pass through
// a config document — the CLI's per-environment token. Config files go through
// ExpandNode.
func ExpandStrict(s string) (string, error) {
	name := placeholderName(s)
	if name == "" {
		return s, nil
	}

	value, ok := os.LookupEnv(name)
	if !ok {
		return "", fmt.Errorf("environment variable %q is not set", name)
	}

	return value, nil
}

// ExpandNode resolves placeholders in every string scalar of a parsed YAML
// document, in place. Subtrees named by skipPaths (dotted mapping paths such
// as "observer.replacementRules") are left untouched: a Go regexp replacement
// of exactly "${name}" is a named backreference there, not a reference.
//
// A placeholder written without quotes is re-typed from the value it resolves
// to when that value is a canonical boolean or an integer yaml can hold, so it
// can stand in for a port or a flag as well as a string. Every other value
// stays a string, and so do quoted placeholders and mapping keys.
func ExpandNode(root *yaml.Node, skipPaths ...string) error {
	return walk(root, "", skipSet(skipPaths), func(n *yaml.Node, _ string, isKey bool) error {
		return expandScalar(n, isKey)
	})
}

// Reference is one placeholder occurrence: the variable it names, the dotted
// mapping path of the scalar holding it (a key is reported at its mapping's
// path), and whether that scalar is a key.
type Reference struct {
	Path  string
	Name  string
	IsKey bool
}

// References lists every placeholder occurrence in a parsed document, in
// document order, skipping the same subtrees ExpandNode skips.
func References(root *yaml.Node, skipPaths ...string) []Reference {
	var refs []Reference

	_ = walk(root, "", skipSet(skipPaths), func(n *yaml.Node, path string, isKey bool) error {
		if name := placeholderName(n.Value); name != "" {
			refs = append(refs, Reference{Path: path, Name: name, IsKey: isKey})
		}

		return nil
	})

	return refs
}

func skipSet(paths []string) map[string]struct{} {
	skip := make(map[string]struct{}, len(paths))
	for _, p := range paths {
		skip[p] = struct{}{}
	}

	return skip
}

// walk visits every string scalar below n with its dotted path. The document
// node and sequences do not extend the path; mapping values extend it with
// their key.
func walk(n *yaml.Node, path string, skip map[string]struct{}, visit func(*yaml.Node, string, bool) error) error {
	if n.Kind == yaml.ScalarNode && isStringTag(n.Tag) {
		return visit(n, path, false)
	}

	if n.Kind != yaml.MappingNode {
		for _, child := range n.Content {
			if err := walk(child, path, skip, visit); err != nil {
				return err
			}
		}

		return nil
	}

	// a mapping stores key and value alternately
	for i := 0; i+1 < len(n.Content); i += 2 {
		key, value := n.Content[i], n.Content[i+1]

		if key.Kind == yaml.ScalarNode && isStringTag(key.Tag) {
			if err := visit(key, path, true); err != nil {
				return err
			}
		}

		childPath := key.Value
		if path != "" {
			childPath = path + "." + key.Value
		}

		if _, skipped := skip[childPath]; skipped {
			continue
		}

		if err := walk(value, childPath, skip, visit); err != nil {
			return err
		}
	}

	return nil
}

func expandScalar(n *yaml.Node, isKey bool) error {
	name := placeholderName(n.Value)
	if name == "" {
		return nil
	}

	value, ok := os.LookupEnv(name)
	if !ok {
		return fmt.Errorf("%s at line %d:%d: %w", name, n.Line, n.Column, ErrUnsetEnv)
	}

	n.Value = value

	// an unquoted placeholder stands in for the whole value, so drop the tag the
	// parser inferred from the placeholder text and let yaml re-resolve it from
	// what came back. Without this the scalar stays !!str and cannot decode into
	// a number or a boolean. Only values that survive that round trip unchanged
	// qualify: quoted scalars, keys, and anything yaml would rewrite stay
	// strings, so an all-digit key cannot become an int and break the map around
	// it, and a password cannot be re-typed into something else entirely.
	if n.Style == 0 && !isKey && retypeSafe(value) {
		n.Tag = ""
		return nil
	}

	// the string tag on its own is not enough. The document is emitted by yaml.v3
	// and decoded by yaml.v2, and v3 emits a plain scalar whenever v3 itself would
	// read it back as a string — "yes", "no" and "0755" among them, all of which
	// v2 resolves under YAML 1.1 as a boolean or an octal. Quoting is what carries
	// the value across that hand-off intact.
	n.Tag = strTag
	n.Style = yaml.DoubleQuotedStyle

	return nil
}

func isStringTag(tag string) bool { return tag == "" || tag == strTag }
