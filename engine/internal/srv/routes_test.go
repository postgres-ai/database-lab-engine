package srv

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOwnerFromEmail(t *testing.T) {
	longLocal := strings.Repeat("a", maxOwnerLabelLength+1)

	testCases := []struct {
		name    string
		email   string
		want    string
		wantErr bool
	}{
		{name: "simple local part", email: "jsmith@acme.io", want: "jsmith"},
		{name: "case preserved", email: "JSmith@Acme.io", want: "JSmith"},
		{name: "dotted local part", email: "a.b@acme.io", want: "a.b"},
		{name: "underscore and hyphen", email: "j_t-x@acme.io", want: "j_t-x"},
		{name: "display name form", email: `"John Smith" <jsmith@acme.io>`, want: "jsmith"},
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
	owner, err := ownerFromContext(context.Background())
	require.NoError(t, err)
	assert.Empty(t, owner)
}
