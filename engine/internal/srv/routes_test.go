package srv

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/postgres-ai/database-lab/v3/internal/platform"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval"
	"gitlab.com/postgres-ai/database-lab/v3/internal/srv/mw"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
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

func TestObservation_NullBodyIsBadRequest(t *testing.T) {
	pl, err := platform.New(context.Background(), platform.Config{}, "instanceID")
	require.NoError(t, err)

	s := &Server{Platform: pl}

	testCases := []struct {
		name    string
		path    string
		handler http.HandlerFunc
	}{
		{name: "start observation", path: "/observation/start", handler: s.startObservation},
		{name: "stop observation", path: "/observation/stop", handler: s.stopObservation},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader("null"))
			rec := httptest.NewRecorder()

			require.NotPanics(t, func() { tc.handler(rec, req) })
			assert.Equal(t, http.StatusBadRequest, rec.Code)

			var apiErr models.Error
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &apiErr))
			assert.Equal(t, models.ErrCodeBadRequest, apiErr.Code)
			assert.Equal(t, errEmptyRequestBody, apiErr.Message)
		})
	}
}

func TestRefresh_PrecheckDoesNotConsumeTheSlot(t *testing.T) {
	s := &Server{Retrieval: &retrieval.Retrieval{}}

	// A refresh already holds the slot, so the handler must reject the request.
	require.NoError(t, s.Retrieval.State.TryStartRefresh())

	rec := httptest.NewRecorder()
	s.refresh(rec, httptest.NewRequest(http.MethodPost, "/full-refresh", nil))

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var apiErr models.Error
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &apiErr))
	assert.Equal(t, retrieval.ErrRefreshInProgress.Error(), apiErr.Message)

	// The rejected request left the slot alone: releasing it lets the running refresh finish
	// and the next one claim it.
	s.Retrieval.State.FinishRefresh()
	assert.NoError(t, s.Retrieval.State.TryStartRefresh())
}

func TestRefresh_ConcurrentRequestsLeaveTheSlotIntact(t *testing.T) {
	s := &Server{Retrieval: &retrieval.Retrieval{}}

	// A refresh already holds the slot, so none of the concurrent requests may start one.
	require.NoError(t, s.Retrieval.State.TryStartRefresh())

	const requests = 16

	codes := make([]int, requests)

	var wg sync.WaitGroup

	wg.Add(requests)

	for i := range codes {
		go func() {
			defer wg.Done()

			rec := httptest.NewRecorder()
			s.refresh(rec, httptest.NewRequest(http.MethodPost, "/full-refresh", nil))
			codes[i] = rec.Code
		}()
	}

	wg.Wait()

	for i, code := range codes {
		assert.Equal(t, http.StatusBadRequest, code, "request %d must be rejected", i)
	}

	// The rejected requests neither claimed nor released the slot: one release frees it.
	s.Retrieval.State.FinishRefresh()
	assert.NoError(t, s.Retrieval.State.CanStartRefresh())
}
