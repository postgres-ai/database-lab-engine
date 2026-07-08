package branch

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

func TestBranchProtectionAnnotation(t *testing.T) {
	future := models.NewLocalTime(time.Date(2099, 1, 2, 12, 0, 0, 0, time.UTC))
	past := models.NewLocalTime(time.Date(2000, 1, 2, 12, 0, 0, 0, time.UTC))

	testCases := []struct {
		name   string
		branch models.BranchView
		want   string
	}{
		{name: "unprotected", branch: models.BranchView{Name: "dev"}, want: ""},
		{name: "protected forever", branch: models.BranchView{Name: "dev", Protected: true}, want: "[protected]"},
		{name: "protected until future", branch: models.BranchView{Name: "dev", Protected: true, ProtectedTill: future},
			want: "[protected until 2099-01-02T12:00:00Z]"},
		{name: "expired protection shows nothing", branch: models.BranchView{Name: "dev", Protected: true, ProtectedTill: past},
			want: ""},
		{name: "scheduled deletion", branch: models.BranchView{Name: "dev", DeleteAt: future},
			want: "[auto-delete at 2099-01-02T12:00:00Z]"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, branchProtectionAnnotation(tc.branch))
		})
	}
}

func TestDedupBranchViews(t *testing.T) {
	branches := []models.BranchView{
		{Name: "dev", Protected: true},
		{Name: "dev", Protected: false},
		{Name: "main"},
	}

	views := dedupBranchViews(branches)
	assert.Len(t, views, 2)
	assert.True(t, views["dev"].Protected, "the first occurrence of a branch name is kept")
}

func TestFormatBranchList(t *testing.T) {
	till := models.NewLocalTime(time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC))

	branches := []models.BranchView{
		{Name: "main"},
		{Name: "dev", Protected: true},
		{Name: "feature", DeleteAt: till},
	}

	out := formatBranchList("main", branches)

	assert.Contains(t, out, "* ", "the current branch is marked with a star")
	assert.Contains(t, out, "  dev  [protected]")
	assert.Contains(t, out, "[auto-delete at 2026-06-20T12:00:00Z]")
	assert.Less(t, strings.Index(out, "dev"), strings.Index(out, "feature"), "branches are listed in sorted order")
}
