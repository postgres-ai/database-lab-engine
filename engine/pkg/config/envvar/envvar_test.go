package envvar

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestExpandStrict(t *testing.T) {
	t.Setenv("DBLAB_TOKEN", "secret-from-env")
	t.Setenv("EMPTY_VAR", "")

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr string
	}{
		{name: "empty string", input: "", want: ""},
		{name: "plain value", input: "plain-secret", want: "plain-secret"},
		{name: "braced placeholder", input: "${DBLAB_TOKEN}", want: "secret-from-env"},
		{name: "unbraced placeholder", input: "$DBLAB_TOKEN", want: "secret-from-env"},
		{name: "explicitly empty env var", input: "${EMPTY_VAR}", want: ""},
		{name: "unset variable", input: "${DBLAB_MISSING}", wantErr: `environment variable "DBLAB_MISSING" is not set`},
		{name: "regex backreference is a literal", input: "***$1", want: "***$1"},
		{name: "dollar inside a secret is a literal", input: "p@$$w0rd", want: "p@$$w0rd"},
		{name: "placeholder inside a longer value is a literal", input: "repo/tool:${TAG}", want: "repo/tool:${TAG}"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ExpandStrict(tc.input)
			if tc.wantErr != "" {
				require.EqualError(t, err, tc.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func expandDoc(t *testing.T, doc string) (map[string]interface{}, error) {
	t.Helper()

	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(doc), &root))

	if err := ExpandNode(&root); err != nil {
		return nil, err
	}

	out := map[string]interface{}{}
	require.NoError(t, root.Decode(&out))

	return out, nil
}

// valueTag expands a single-key document and reports the tag left on its value.
func valueTag(t *testing.T, doc string) string {
	t.Helper()

	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(doc), &root))
	require.NoError(t, ExpandNode(&root))

	return root.Content[0].Content[1].Tag
}

func TestExpandNode(t *testing.T) {
	t.Setenv("NODE_TOKEN", "secret-from-env")
	t.Setenv("NODE_EMPTY", "")

	t.Run("reaches arbitrarily nested values", func(t *testing.T) {
		out, err := expandDoc(t, "a:\n  b:\n    c: \"${NODE_TOKEN}\"\n")
		require.NoError(t, err)
		assert.Equal(t, "secret-from-env",
			out["a"].(map[string]interface{})["b"].(map[string]interface{})["c"])
	})

	t.Run("reaches values inside sequences", func(t *testing.T) {
		out, err := expandDoc(t, "hooks:\n  - secret: \"${NODE_TOKEN}\"\n")
		require.NoError(t, err)
		assert.Equal(t, "secret-from-env",
			out["hooks"].([]interface{})[0].(map[string]interface{})["secret"])
	})

	t.Run("explicitly empty variable resolves to empty", func(t *testing.T) {
		out, err := expandDoc(t, "a: \"${NODE_EMPTY}\"\n")
		require.NoError(t, err)
		assert.Equal(t, "", out["a"])
	})

	t.Run("unquoted placeholder is re-typed", func(t *testing.T) {
		t.Setenv("NODE_PORT", "2500")
		t.Setenv("NODE_FLAG", "true")

		out, err := expandDoc(t, "port: ${NODE_PORT}\nflag: ${NODE_FLAG}\n")
		require.NoError(t, err)
		assert.Equal(t, 2500, out["port"])
		assert.Equal(t, true, out["flag"])
	})

	t.Run("quoted placeholder stays a string", func(t *testing.T) {
		t.Setenv("NODE_NUMERIC", "2500")

		out, err := expandDoc(t, "port: \"${NODE_NUMERIC}\"\n")
		require.NoError(t, err)
		assert.Equal(t, "2500", out["port"])
	})

	t.Run("mapping keys stay strings", func(t *testing.T) {
		t.Setenv("NODE_KEY", "1234")

		out, err := expandDoc(t, "servers:\n  ${NODE_KEY}:\n    url: u\n")
		require.NoError(t, err)
		assert.Contains(t, out["servers"], "1234")
	})

	t.Run("unset variable reports name and position", func(t *testing.T) {
		_, err := expandDoc(t, "a: 1\nb: \"${NODE_DEFINITELY_MISSING}\"\n")
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrUnsetEnv))
		assert.Contains(t, err.Error(), "NODE_DEFINITELY_MISSING")
		assert.Contains(t, err.Error(), "line 2")
	})
}

func TestExpandNodeKeepsLiterals(t *testing.T) {
	t.Setenv("TAG", "v1.2.3")
	t.Setenv("VERSION", "v1.2.3")
	t.Setenv("BAD", "resolved")

	tests := []struct {
		name  string
		value string
	}{
		{name: "dollar in a password", value: "p@$w0rd"},
		{name: "doubled dollar in a password", value: "p@$$w0rd"},
		{name: "regex backreference", value: "***$1"},
		{name: "regex end anchor", value: `(\w+)@example\.com$`},
		{name: "braced placeholder in a longer value", value: "repo/tool:${TAG}"},
		{name: "bare placeholder in a longer value", value: "repo/tool:$VERSION"},
		{name: "two placeholders", value: "$TAG$VERSION"},
		{name: "unterminated brace", value: "${UNCLOSED"},
		{name: "empty braces", value: "${}"},
		{name: "dash in name", value: "${BAD-NAME}"},
		{name: "at sign in name", value: "${V@R}"},
		{name: "leading digit", value: "${1ABC}"},
		{name: "bare name with a dash", value: "$BAD-NAME"},
		{name: "lone dollar", value: "$"},
		{name: "shell substitution", value: "$(date)"},
		{name: "awk program", value: "{print $1}"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out, err := expandDoc(t, "a: '"+tc.value+"'\n")
			require.NoError(t, err)
			assert.Equal(t, tc.value, out["a"])
		})
	}
}

func TestExpandNodeKeepsDollarQuotedSQL(t *testing.T) {
	doc := "queryPreprocessing:\n" +
		"  inline: |\n" +
		"    DO $$ BEGIN\n" +
		"      PERFORM 1;\n" +
		"    END $$;\n"

	out, err := expandDoc(t, doc)
	require.NoError(t, err)

	inline := out["queryPreprocessing"].(map[string]interface{})["inline"]
	assert.Equal(t, "DO $$ BEGIN\n  PERFORM 1;\nEND $$;\n", inline)
}

func TestExpandNodeKeepsReplacementRules(t *testing.T) {
	t.Setenv("NODE_TOKEN", "resolved")

	doc := "observer:\n" +
		"  replacementRules:\n" +
		"    \"(\\\\w+)@example\\\\.com$\": \"***$1\"\n" +
		"    \"select \\\\d+\": \"***\"\n" +
		"other:\n" +
		"  value: \"${NODE_TOKEN}\"\n"

	out, err := expandDoc(t, doc)
	require.NoError(t, err)

	rules := out["observer"].(map[string]interface{})["replacementRules"].(map[string]interface{})
	assert.Equal(t, `***$1`, rules[`(\w+)@example\.com$`], "regex rule must survive untouched")
	assert.Equal(t, `***`, rules[`select \d+`])
	assert.Equal(t, "resolved", out["other"].(map[string]interface{})["value"])
}

func TestExpandNodeRetypesOnlyUnambiguousValues(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		retyped  bool
	}{
		{name: "port", envValue: "5432", retyped: true},
		{name: "zero", envValue: "0", retyped: true},
		{name: "negative", envValue: "-1", retyped: true},
		{name: "true", envValue: "true", retyped: true},
		{name: "false", envValue: "false", retyped: true},
		{name: "yaml 1.1 yes", envValue: "yes"},
		{name: "yaml 1.1 no", envValue: "no"},
		{name: "yaml 1.1 on", envValue: "on"},
		{name: "yaml 1.1 off", envValue: "off"},
		{name: "capitalized true", envValue: "True"},
		{name: "leading zero", envValue: "0755"},
		{name: "hex", envValue: "0x1F"},
		{name: "exponent", envValue: "1e5"},
		{name: "underscores", envValue: "1_000"},
		{name: "negative zero", envValue: "-0"},
		{name: "null", envValue: "null"},
		{name: "tilde", envValue: "~"},
		{name: "duration", envValue: "3m"},
		{name: "time", envValue: "1:30"},
		{name: "date", envValue: "2026-08-03"},
		{name: "ordinary password", envValue: "p@ssw0rd"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("RETYPE_VALUE", tc.envValue)

			tag := valueTag(t, "value: ${RETYPE_VALUE}\n")
			if tc.retyped {
				assert.Empty(t, tag, "value should be re-resolved by yaml")
				return
			}

			assert.Equal(t, strTag, tag, "value should stay a string so yaml cannot rewrite it")
		})
	}
}

func TestExpandNodeSkipsPaths(t *testing.T) {
	t.Setenv("user", "resolved-user")
	t.Setenv("NODE_TOKEN", "resolved")

	doc := "observer:\n  replacementRules:\n    \"(?P<user>\\\\w+)@example\\\\.com\": \"${user}\"\n    \"(?P<host>[a-z]+)$\": \"$host\"\n" +
		"  other: \"${NODE_TOKEN}\"\n"

	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(doc), &root))
	assert.Equal(t, []Reference{{Path: "observer.other", Name: "NODE_TOKEN"}}, References(&root, "observer.replacementRules"),
		"the skipped subtree is not a reference either")
	require.NoError(t, ExpandNode(&root, "observer.replacementRules"))

	out := map[string]interface{}{}
	require.NoError(t, root.Decode(&out))

	observer := out["observer"].(map[string]interface{})
	rules := observer["replacementRules"].(map[string]interface{})
	assert.Equal(t, "${user}", rules[`(?P<user>\w+)@example\.com`], "a named backreference is not an env reference")
	assert.Equal(t, "$host", rules[`(?P<host>[a-z]+)$`])
	assert.Equal(t, "resolved", observer["other"], "a sibling of the skipped subtree still expands")
}

func TestExpandNodeKeepsOutOfRangeIntegersAsStrings(t *testing.T) {
	for envValue, retyped := range map[string]bool{
		"9223372036854775807": true, "18446744073709551615": true, "-9223372036854775808": true,
		"18446744073709551616": false, "-9223372036854775809": false, "99999999999999999999": false,
	} {
		t.Setenv("RETYPE_VALUE", envValue)

		tag := valueTag(t, "value: ${RETYPE_VALUE}\n")
		if retyped {
			assert.Empty(t, tag, envValue)
			continue
		}

		assert.Equal(t, strTag, tag, "an integer yaml would turn into a float stays a string: "+envValue)
	}
}
