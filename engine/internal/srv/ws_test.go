package srv

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/postgres-ai/database-lab/v3/internal/platform"
	"gitlab.com/postgres-ai/database-lab/v3/internal/srv/config"
)

func TestLogLineFiltering(t *testing.T) {
	pl, err := platform.New(context.Background(), platform.Config{AccessToken: "platformAccessToken"})
	require.NoError(t, err)

	s := Server{Config: &config.Config{VerificationToken: "secretToken"}, Platform: pl}
	s.initLogRegExp()

	testCases := []struct {
		input  []byte
		output []byte
	}{
		{
			input:  []byte(`verificationToken: "secretToken"`),
			output: []byte(`verificationToken: "********"`),
		},
		{
			input:  []byte(`password: "secret_token"`),
			output: []byte(`********`),
		},
		{
			input:  []byte(`password:"secret_token", host: "127.0.0.1"`),
			output: []byte(`******** host: "127.0.0.1"`),
		},
		{
			input:  []byte(`password:secret_token`),
			output: []byte(`********`),
		},
		{
			input:  []byte(`POSTGRES_PASSWORD=password`),
			output: []byte(`********`),
		},
		{
			input:  []byte(`PGPASSWORD=password`),
			output: []byte(`********`),
		},
		{
			input:  []byte(`accessToken:secret_token`),
			output: []byte(`********`),
		},
		{
			input:  []byte(`accessToken: "secret_token"`),
			output: []byte(`********`),
		},
		{
			input:  []byte(`AWS_SECRET_ACCESS_KEY:password`),
			output: []byte(`AWS_SECRET_********`),
		},
		{
			input:  []byte(`AWS_ACCESS_KEY_ID:password`),
			output: []byte(`AWS_********`),
		},
		{
			input:  []byte(`platform: "platformAccessToken"`),
			output: []byte(`platform: "********"`),
		},
	}

	for _, tc := range testCases {
		filteredLine := s.filterLogLine(tc.input)

		assert.Equal(t, tc.output, filteredLine)
	}
}
