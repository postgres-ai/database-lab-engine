// Package envvar expands a config field whose value is solely a "${VAR}" or
// "$VAR" placeholder using the process environment. A value that merely
// contains a "$" (a password, a regex backreference, an unmatched brace) is a
// literal and is left unchanged.
package envvar

import (
	"os"
	"regexp"

	"github.com/pkg/errors"
)

// Field pairs a config field's display name with a pointer to its value.
type Field struct {
	Name string
	Ptr  *string
}

// placeholderRE matches a value that consists solely of a single "${VAR}" or
// "$VAR" reference, capturing the variable name. Anything else (including a
// value that merely embeds a "$") is treated as a literal so secrets that
// legitimately contain a "$" survive load without corruption.
var placeholderRE = regexp.MustCompile(`^\$\{([A-Za-z_][A-Za-z0-9_]*)\}$|^\$([A-Za-z_][A-Za-z0-9_]*)$`)

// ExpandStrict resolves s when it is exactly a single "${VAR}" or "$VAR"
// placeholder, returning the referenced environment variable's value. It
// returns an error if the referenced variable is unset, so a misconfigured
// placeholder fails loudly at startup instead of silently resolving to an
// empty string. Any other value, including one that merely contains a "$", is
// returned unchanged.
func ExpandStrict(s string) (string, error) {
	m := placeholderRE.FindStringSubmatch(s)
	if m == nil {
		return s, nil
	}

	name := m[1]
	if name == "" {
		name = m[2]
	}

	v, ok := os.LookupEnv(name)
	if !ok {
		return "", errors.Errorf("environment variable %q is not set", name)
	}

	return v, nil
}

// ExpandFields applies ExpandStrict to each field and writes the resolved
// value back through the pointer. Errors are wrapped with the field name so
// the caller can show which field failed to resolve.
func ExpandFields(fields []Field) error {
	for _, f := range fields {
		if f.Ptr == nil {
			continue
		}

		resolved, err := ExpandStrict(*f.Ptr)
		if err != nil {
			return errors.Wrapf(err, "config field %s", f.Name)
		}

		*f.Ptr = resolved
	}

	return nil
}
