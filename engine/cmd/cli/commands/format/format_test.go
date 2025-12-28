package format

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v2"
)

func TestConfig_IsJSON(t *testing.T) {
	tests := []struct {
		format string
		want   bool
	}{
		{FormatJSON, true},
		{FormatTable, false},
		{FormatWide, false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			cfg := Config{Format: tt.format}
			assert.Equal(t, tt.want, cfg.IsJSON())
		})
	}
}

func TestConfig_IsWide(t *testing.T) {
	tests := []struct {
		format string
		want   bool
	}{
		{FormatWide, true},
		{FormatTable, false},
		{FormatJSON, false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			cfg := Config{Format: tt.format}
			assert.Equal(t, tt.want, cfg.IsWide())
		})
	}
}

func TestConfig_IsTable(t *testing.T) {
	tests := []struct {
		format string
		want   bool
	}{
		{FormatTable, true},
		{"", true},
		{FormatJSON, false},
		{FormatWide, false},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			cfg := Config{Format: tt.format}
			assert.Equal(t, tt.want, cfg.IsTable())
		})
	}
}

func TestFromContext(t *testing.T) {
	app := &cli.App{
		Flags: []cli.Flag{OutputFlag, NoColorFlag},
		Action: func(c *cli.Context) error {
			cfg := FromContext(c)
			assert.Equal(t, FormatJSON, cfg.Format)
			assert.True(t, cfg.NoColor)
			return nil
		},
		Writer: &bytes.Buffer{},
	}

	err := app.Run([]string{"test", "--output", "json", "--no-color"})
	assert.NoError(t, err)
}

func TestFromContext_DefaultValues(t *testing.T) {
	app := &cli.App{
		Flags: []cli.Flag{OutputFlag, NoColorFlag},
		Action: func(c *cli.Context) error {
			cfg := FromContext(c)
			assert.Equal(t, FormatTable, cfg.Format)
			return nil
		},
		Writer: &bytes.Buffer{},
	}

	err := app.Run([]string{"test"})
	assert.NoError(t, err)
}

func TestFromContext_NoColorEnvVar(t *testing.T) {
	os.Setenv("NO_COLOR", "1")
	defer os.Unsetenv("NO_COLOR")

	app := &cli.App{
		Flags: []cli.Flag{OutputFlag, NoColorFlag},
		Action: func(c *cli.Context) error {
			cfg := FromContext(c)
			assert.True(t, cfg.NoColor)
			return nil
		},
		Writer: &bytes.Buffer{},
	}

	err := app.Run([]string{"test"})
	assert.NoError(t, err)
}
