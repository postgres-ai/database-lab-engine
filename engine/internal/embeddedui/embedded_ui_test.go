package embeddedui

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/config/global"
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

func TestGetEmbeddedUIName(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		instanceID string
		expected   string
	}{
		{instanceID: "test123", expected: "dblab_embedded_ui_test123"},
		{instanceID: "", expected: "dblab_embedded_ui_"},
		{instanceID: "my-instance", expected: "dblab_embedded_ui_my-instance"},
		{instanceID: "a_b_c", expected: "dblab_embedded_ui_a_b_c"},
	}

	for _, tc := range testCases {
		t.Run(tc.instanceID, func(t *testing.T) {
			result := getEmbeddedUIName(tc.instanceID)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestUIManager_IsEnabled(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		enabled  bool
		expected bool
	}{
		{name: "enabled", enabled: true, expected: true},
		{name: "disabled", enabled: false, expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ui := &UIManager{cfg: Config{Enabled: tc.enabled}}
			assert.Equal(t, tc.expected, ui.IsEnabled())
		})
	}
}

func TestUIManager_GetHost(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		host     string
		expected string
	}{
		{name: "specific host", host: "192.168.1.1", expected: "192.168.1.1"},
		{name: "empty host", host: "", expected: ""},
		{name: "localhost", host: "127.0.0.1", expected: "127.0.0.1"},
		{name: "domain host", host: "example.com", expected: "example.com"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ui := &UIManager{cfg: Config{Host: tc.host}}
			assert.Equal(t, tc.expected, ui.GetHost())
		})
	}
}

func TestUIManager_IsConfigChanged(t *testing.T) {
	t.Parallel()

	baseCfg := Config{Enabled: true, DockerImage: "nginx:1.23", Host: "0.0.0.0", Port: 2345}

	testCases := []struct {
		name    string
		newCfg  Config
		changed bool
	}{
		{name: "identical config", newCfg: Config{Enabled: true, DockerImage: "nginx:1.23", Host: "0.0.0.0", Port: 2345}, changed: false},
		{name: "enabled changed", newCfg: Config{Enabled: false, DockerImage: "nginx:1.23", Host: "0.0.0.0", Port: 2345}, changed: true},
		{name: "image changed", newCfg: Config{Enabled: true, DockerImage: "nginx:1.24", Host: "0.0.0.0", Port: 2345}, changed: true},
		{name: "host changed", newCfg: Config{Enabled: true, DockerImage: "nginx:1.23", Host: "127.0.0.1", Port: 2345}, changed: true},
		{name: "port changed", newCfg: Config{Enabled: true, DockerImage: "nginx:1.23", Host: "0.0.0.0", Port: 9999}, changed: true},
		{name: "all changed", newCfg: Config{Enabled: false, DockerImage: "alpine:3", Host: "10.0.0.1", Port: 8080}, changed: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.changed, isConfigChanged(baseCfg, tc.newCfg))
		})
	}
}

func TestNew(t *testing.T) {
	t.Parallel()

	cfg := Config{Enabled: true, DockerImage: "nginx:1.23", Host: "0.0.0.0", Port: 2345}
	engProps := global.EngineProps{InstanceID: "test-id", ContainerName: "test-container"}

	ui := New(cfg, engProps, nil, nil)

	require.NotNil(t, ui)
	assert.Equal(t, cfg, ui.cfg)
	assert.Equal(t, engProps, ui.engProps)
	assert.Nil(t, ui.runner)
	assert.Nil(t, ui.docker)
}

func TestUIManager_OriginURL_IPv6(t *testing.T) {
	t.Parallel()

	ui := &UIManager{cfg: Config{Host: "::1", Port: 8080}}
	result := ui.OriginURL()
	assert.Equal(t, "http://[::1]:8080", result)
}

func TestUIManager_OriginURL_HighPort(t *testing.T) {
	t.Parallel()

	ui := &UIManager{cfg: Config{Host: "10.0.0.1", Port: 65535}}
	assert.Equal(t, "http://10.0.0.1:65535", ui.OriginURL())
}

func TestReloadRacesWithConfigReaders(t *testing.T) {
	t.Parallel()

	cfg := Config{Enabled: true, DockerImage: "nginx:1.23", Host: "0.0.0.0", Port: 2345}
	ui := &UIManager{cfg: cfg}

	const iterations = 200

	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		defer wg.Done()

		for i := 0; i < iterations; i++ {
			// Reload writes ui.cfg before it decides what to do. An unchanged config stops it
			// there, which keeps the test off the container paths that need a Docker daemon.
			assert.NoError(t, ui.Reload(context.Background(), cfg))
		}
	}()

	go func() {
		defer wg.Done()

		for i := 0; i < iterations; i++ {
			_ = ui.IsEnabled()
			_ = ui.GetHost()
			_ = ui.OriginURL()
			_ = ui.Config()
		}
	}()

	wg.Wait()

	assert.Equal(t, cfg, ui.Config())
}
