package embeddedui

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUIManagerOriginURL(t *testing.T) {
	testCases := []struct {
		host   string
		port   int
		result string
	}{
		{
			host:   "example.com",
			port:   2345,
			result: "http://example.com:2345",
		},
		{
			host:   "example.com",
			result: "http://example.com:0",
		},
		{
			host:   "container_host",
			port:   2345,
			result: "http://container_host:2345",
		},
		{
			host:   "172.168.1.1",
			port:   2345,
			result: "http://172.168.1.1:2345",
		},
		{
			host:   "0.0.0.0",
			port:   2345,
			result: "http://127.0.0.1:2345",
		},
		{
			host:   "",
			port:   2345,
			result: "http://127.0.0.1:2345",
		},
	}

	for _, tc := range testCases {
		uiManager := UIManager{
			cfg: Config{
				Enabled: true,
				Host:    tc.host,
				Port:    tc.port,
			},
		}

		require.True(t, uiManager.IsEnabled())
		require.Equal(t, tc.result, uiManager.OriginURL())

	}
}
