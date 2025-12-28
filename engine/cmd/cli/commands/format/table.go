package format

import (
	"io"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
)

// Table wraps tablewriter with convenient defaults.
type Table struct {
	*tablewriter.Table
	noColor bool
}

// NewTable creates a new table with CLI-friendly defaults.
func NewTable(w io.Writer, noColor bool) *Table {
	t := tablewriter.NewWriter(w)

	t.SetAutoWrapText(false)
	t.SetAutoFormatHeaders(true)
	t.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	t.SetAlignment(tablewriter.ALIGN_LEFT)
	t.SetCenterSeparator("")
	t.SetColumnSeparator("")
	t.SetRowSeparator("")
	t.SetHeaderLine(false)
	t.SetBorder(false)
	t.SetTablePadding("  ")
	t.SetNoWhiteSpace(true)

	return &Table{Table: t, noColor: noColor}
}

// SetHeaders sets table headers with optional coloring.
func (t *Table) SetHeaders(headers ...string) {
	if !t.noColor {
		t.Table.SetHeaderColor(
			repeat(tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiWhiteColor}, len(headers))...,
		)
	}
	t.Table.SetHeader(headers)
}

// repeat creates a slice of identical color configurations.
func repeat(c tablewriter.Colors, n int) []tablewriter.Colors {
	result := make([]tablewriter.Colors, n)
	for i := range result {
		result[i] = c
	}
	return result
}

// Status formatting helpers.
var (
	StatusOK      = color.New(color.FgGreen).SprintFunc()
	StatusWarning = color.New(color.FgYellow).SprintFunc()
	StatusError   = color.New(color.FgRed).SprintFunc()
	StatusPending = color.New(color.FgCyan).SprintFunc()
	Highlight     = color.New(color.FgHiWhite, color.Bold).SprintFunc()
	Dim           = color.New(color.FgHiBlack).SprintFunc()
)

// FormatStatus returns a colored status string.
func FormatStatus(code string, noColor bool) string {
	if noColor {
		return code
	}

	switch code {
	case "OK", "READY", "RUNNING":
		return StatusOK(code)
	case "CREATING", "RESETTING", "PENDING", "REFRESHING":
		return StatusPending(code)
	case "WARNING", "DELETING":
		return StatusWarning(code)
	case "FATAL", "ERROR", "FAILED":
		return StatusError(code)
	default:
		return code
	}
}

// FormatTime returns a human-readable relative time.
func FormatTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return humanize.Time(t)
}

// FormatTimeAbs returns an absolute time string.
func FormatTimeAbs(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format("2006-01-02 15:04:05")
}

// FormatBytes returns a human-readable byte size.
func FormatBytes(bytes uint64) string {
	if bytes == 0 {
		return "-"
	}
	return humanize.Bytes(bytes)
}

// FormatBool returns a checkmark or dash for boolean values.
func FormatBool(b bool, noColor bool) string {
	if b {
		if noColor {
			return "yes"
		}
		return StatusOK("✓")
	}
	return "-"
}

// Truncate shortens a string to max length, adding ellipsis if needed.
func Truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}
