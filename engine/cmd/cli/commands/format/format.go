// Package format provides output formatting utilities for the CLI.
package format

import (
	"io"
	"os"

	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
	"github.com/urfave/cli/v2"
)

// Output format types.
const (
	FormatTable = "table"
	FormatJSON  = "json"
	FormatWide  = "wide"
)

// Global flags for output formatting.
var (
	OutputFlag = &cli.StringFlag{
		Name:    "output",
		Aliases: []string{"o"},
		Usage:   "output format: table, json, wide",
		Value:   FormatTable,
		EnvVars: []string{"DBLAB_OUTPUT"},
	}
	NoColorFlag = &cli.BoolFlag{
		Name:    "no-color",
		Usage:   "disable colored output",
		EnvVars: []string{"NO_COLOR", "DBLAB_NO_COLOR"},
	}
)

// Config holds output formatting configuration.
type Config struct {
	Format  string
	NoColor bool
	IsTTY   bool
	Writer  io.Writer
}

// FromContext extracts formatting configuration from CLI context.
func FromContext(c *cli.Context) Config {
	writer := c.App.Writer
	if writer == nil {
		writer = os.Stdout
	}

	isTTY := false
	if f, ok := writer.(*os.File); ok {
		isTTY = isatty.IsTerminal(f.Fd()) || isatty.IsCygwinTerminal(f.Fd())
	}

	noColor := c.Bool("no-color") || os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" || !isTTY

	if noColor {
		color.NoColor = true
	}

	return Config{
		Format:  c.String("output"),
		NoColor: noColor,
		IsTTY:   isTTY,
		Writer:  writer,
	}
}

// IsJSON returns true if JSON output is requested.
func (c Config) IsJSON() bool {
	return c.Format == FormatJSON
}

// IsWide returns true if wide output is requested.
func (c Config) IsWide() bool {
	return c.Format == FormatWide
}

// IsTable returns true if table output is requested.
func (c Config) IsTable() bool {
	return c.Format == FormatTable || c.Format == ""
}
