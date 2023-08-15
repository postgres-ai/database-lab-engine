package log

import (
	"regexp"
	"strings"
	"unicode"
)

const (
	minTokenLength = 8
	replacingMask  = "********"
)

var filter = newFiltering([]string{})

// Filtering represents a struct to filter secrets in logs output.
type Filtering struct {
	re *regexp.Regexp
}

// GetFilter gets an instance of Log Filtering.
func GetFilter() *Filtering {
	return filter
}

func newFiltering(tokens []string) *Filtering {
	f := &Filtering{}
	f.ReloadLogRegExp(tokens)

	return f
}

// ReloadLogRegExp updates secrets configuration.
func (f *Filtering) ReloadLogRegExp(secretStings []string) {
	secretPatterns := []string{
		"password:\\s?(\\S+)",
		"POSTGRES_PASSWORD=(\\S+)",
		"PGPASSWORD=(\\S+)",
		"accessToken:\\s?(\\S+)",
		"orgKey:\\s?(\\S+)",
		"ACCESS_KEY(_ID)?:\\s?(\\S+)",
		"secret:\\s?(\\S+)",
	}

	for _, secret := range secretStings {
		if len(secret) >= minTokenLength && !containsSpace(secret) {
			secretPatterns = append(secretPatterns, secret)
		}
	}

	f.re = regexp.MustCompile("(?i)" + strings.Join(secretPatterns, "|"))
}

// ReplaceAll replaces all secrets in the input line.
func (f *Filtering) ReplaceAll(input []byte) []byte {
	return f.re.ReplaceAll(input, []byte(replacingMask))
}

func containsSpace(s string) bool {
	for _, v := range s {
		if unicode.IsSpace(v) {
			return true
		}
	}

	return false
}
