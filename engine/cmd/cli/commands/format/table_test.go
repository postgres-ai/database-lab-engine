package format

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewTable(t *testing.T) {
	buf := &bytes.Buffer{}
	table := NewTable(buf, false)

	assert.NotNil(t, table)
	assert.NotNil(t, table.Table)
}

func TestTable_SetHeaders(t *testing.T) {
	buf := &bytes.Buffer{}
	table := NewTable(buf, true)

	table.SetHeaders("COL1", "COL2", "COL3")
	table.Append([]string{"a", "b", "c"})
	table.Render()

	output := buf.String()
	assert.Contains(t, output, "COL1")
	assert.Contains(t, output, "COL2")
	assert.Contains(t, output, "COL3")
}

func TestTable_Render(t *testing.T) {
	buf := &bytes.Buffer{}
	table := NewTable(buf, true)

	table.SetHeaders("ID", "NAME")
	table.Append([]string{"1", "test1"})
	table.Append([]string{"2", "test2"})
	table.Render()

	output := buf.String()
	assert.Contains(t, output, "1")
	assert.Contains(t, output, "test1")
	assert.Contains(t, output, "2")
	assert.Contains(t, output, "test2")
}

func TestFormatStatus(t *testing.T) {
	tests := []struct {
		code    string
		noColor bool
		want    string
	}{
		{"OK", true, "OK"},
		{"READY", true, "READY"},
		{"CREATING", true, "CREATING"},
		{"WARNING", true, "WARNING"},
		{"FATAL", true, "FATAL"},
		{"UNKNOWN", true, "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			got := FormatStatus(tt.code, tt.noColor)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFormatStatus_WithColor(t *testing.T) {
	// color library auto-disables in non-TTY, so we just verify the function works
	got := FormatStatus("OK", false)
	assert.Contains(t, got, "OK")
}

func TestFormatTime(t *testing.T) {
	tests := []struct {
		name string
		time time.Time
		want string
	}{
		{"zero time", time.Time{}, "-"},
		{"recent time", time.Now().Add(-5 * time.Minute), "5 minutes ago"},
		{"hour ago", time.Now().Add(-1 * time.Hour), "1 hour ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatTime(tt.time)
			if tt.time.IsZero() {
				assert.Equal(t, tt.want, got)
			} else {
				assert.Contains(t, got, "ago")
			}
		})
	}
}

func TestFormatTimeAbs(t *testing.T) {
	tests := []struct {
		name string
		time time.Time
		want string
	}{
		{"zero time", time.Time{}, "-"},
		{"specific time", time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC), "2024-01-15 10:30:00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatTimeAbs(tt.time)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes uint64
		want  string
	}{
		{0, "-"},
		{1024, "1.0 kB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.1 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := FormatBytes(tt.bytes)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFormatBool(t *testing.T) {
	tests := []struct {
		value   bool
		noColor bool
		want    string
	}{
		{true, true, "yes"},
		{false, true, "-"},
		{false, false, "-"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := FormatBool(tt.value, tt.noColor)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFormatBool_WithColor(t *testing.T) {
	got := FormatBool(true, false)
	assert.Contains(t, got, "✓")
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input string
		max   int
		want  string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is a long string", 10, "this is..."},
		{"abc", 3, "abc"},
		{"abcd", 3, "abc"},
		{"ab", 5, "ab"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := Truncate(tt.input, tt.max)
			assert.Equal(t, tt.want, got)
			assert.LessOrEqual(t, len(got), tt.max)
		})
	}
}

func TestRepeat(t *testing.T) {
	colors := repeat(nil, 3)
	assert.Len(t, colors, 3)
}

func TestTable_NoColorHeaders(t *testing.T) {
	buf := &bytes.Buffer{}
	table := NewTable(buf, true)

	table.SetHeaders("A", "B")
	table.Append([]string{"1", "2"})
	table.Render()

	output := buf.String()
	assert.True(t, strings.Contains(output, "A"))
	assert.True(t, strings.Contains(output, "B"))
}
