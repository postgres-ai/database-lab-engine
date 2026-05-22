// Package envvar expands "${VAR}" and "$VAR" placeholders in selected config
// fields using the process environment.
package envvar

import (
	"os"

	"github.com/pkg/errors"
)

// Field pairs a config field's display name with a pointer to its value.
type Field struct {
	Name string
	Ptr  *string
}

// ExpandStrict expands "${VAR}" and "$VAR" references in s. It returns an
// error if any referenced variable is unset, so misconfigured tokens fail
// loudly at startup instead of silently resolving to an empty string.
func ExpandStrict(s string) (string, error) {
	if s == "" {
		return "", nil
	}

	var missing string

	resolved := os.Expand(s, func(name string) string {
		if missing != "" {
			return ""
		}

		v, ok := os.LookupEnv(name)
		if !ok {
			missing = name
			return ""
		}

		return v
	})

	if missing != "" {
		return "", errors.Errorf("environment variable %q is not set", missing)
	}

	return resolved, nil
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
