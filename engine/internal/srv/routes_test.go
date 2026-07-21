package srv

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/postgres-ai/database-lab/v3/internal/platform"
	"gitlab.com/postgres-ai/database-lab/v3/internal/srv/mw"
)

func TestOwnerFromEmail(t *testing.T) {
	longLocal := strings.Repeat("a", maxOwnerLabelLength+1)

	testCases := []struct {
		name    string
		email   string
		want    string
		wantErr bool
	}{
		{name: "full email", email: "jsmith@acme.io", want: "jsmith@acme.io"},
		{name: "case preserved", email: "JSmith@Acme.io", want: "JSmith@Acme.io"},
		{name: "dotted local part", email: "a.b@acme.io", want: "a.b@acme.io"},
		{name: "underscore and hyphen", email: "j_t-x@acme.io", want: "j_t-x@acme.io"},
		{name: "display name stripped", email: `"John Smith" <jsmith@acme.io>`, want: "jsmith@acme.io"},
		{name: "same local part distinct domains", email: "jsmith@other.io", want: "jsmith@other.io"},
		{name: "no domain rejected", email: "jsmith", wantErr: true},
		{name: "plus tag rejected", email: "j+t@acme.io", wantErr: true},
		{name: "space rejected", email: "bad name@acme.io", wantErr: true},
		{name: "empty local rejected", email: "@acme.io", wantErr: true},
		{name: "too long rejected", email: longLocal + "@acme.io", wantErr: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ownerFromEmail(tc.email)
			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestOwnerFromContext_NoIdentity(t *testing.T) {
	assert.Empty(t, ownerFromContext(context.Background()))
}

func TestOwnerFromContext_Identity(t *testing.T) {
	testCases := []struct {
		name  string
		email string
		want  string
	}{
		{name: "valid email yields owner label", email: "jsmith@acme.io", want: "jsmith@acme.io"},
		{name: "unlabelable email falls back to unlabeled", email: "j+t@acme.io", want: ""},
		{name: "empty email falls back to unlabeled", email: "", want: ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := mw.WithUserIdentity(context.Background(), platform.UserIdentity{Email: tc.email})
			assert.Equal(t, tc.want, ownerFromContext(ctx))
		})
	}
}
