/*
2026 © Postgres.ai
*/

package srv

import (
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	srvCfg "gitlab.com/postgres-ai/database-lab/v3/internal/srv/config"
)

func TestRouter_BranchAndSnapshotRoutes(t *testing.T) {
	const (
		logRoute      = "/branch/{branchName}/log"
		commitRoute   = "/branch/snapshot/{id:.*}"
		branchRoute   = "/branch/{branchName}"
		snapshotRoute = "/snapshot/{id:.*}"
	)

	router := (&Server{Config: &srvCfg.Config{}}).newRouter()
	branch := func(name string) map[string]string { return map[string]string{"branchName": name} }
	id := func(id string) map[string]string { return map[string]string{"id": id} }

	testCases := []struct {
		name   string
		method string
		path   string
		tmpl   string
		vars   map[string]string
	}{
		{name: "log of a branch named snapshot", method: "GET", path: "/branch/snapshot/log", tmpl: logRoute, vars: branch("snapshot")},
		{name: "log of a regular branch", method: "GET", path: "/branch/dev/log", tmpl: logRoute, vars: branch("dev")},
		{name: "commit by raw id", method: "GET", path: "/branch/snapshot/p/main@s", tmpl: commitRoute, vars: id("p/main@s")},
		{name: "commit by encoded id", method: "GET", path: "/branch/snapshot/p%2Fmain%40s", tmpl: commitRoute, vars: id("p/main@s")},
		{name: "snapshot by encoded id", method: "GET", path: "/snapshot/p%2Fmain%40s", tmpl: snapshotRoute, vars: id("p/main@s")},
		{name: "delete branch named snapshot", method: "DELETE", path: "/branch/snapshot", tmpl: branchRoute, vars: branch("snapshot")},
		{name: "create a commit", method: "POST", path: "/branch/snapshot", tmpl: "/branch/snapshot", vars: map[string]string{}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var match mux.RouteMatch

			require.True(t, router.Match(httptest.NewRequest(tc.method, tc.path, nil), &match))
			require.NoError(t, match.MatchErr)

			template, err := match.Route.GetPathTemplate()
			require.NoError(t, err)
			assert.Equal(t, tc.tmpl, template)
			assert.Equal(t, tc.vars, match.Vars)
		})
	}
}
