package srv

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"gitlab.com/postgres-ai/database-lab/v3/internal/srv/config"
)

func TestLogLineFiltering(t *testing.T) {
	s := Server{Config: &config.Config{VerificationToken: "secretToken"}}
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
	}

	for _, tc := range testCases {
		filteredLine := s.filterLogLine(tc.input)

		assert.Equal(t, tc.output, filteredLine)
	}
}
