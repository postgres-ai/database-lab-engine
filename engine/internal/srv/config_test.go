package srv

import (
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

func TestCustomOptions(t *testing.T) {
	testCases := []struct {
		customOptions  []interface{}
		expectedResult error
	}{
		{customOptions: []interface{}{"--verbose"}, expectedResult: nil},
		{customOptions: []interface{}{"--exclude-scheme=test_scheme"}, expectedResult: nil},
		{customOptions: []interface{}{`--exclude-scheme="test_scheme"`}, expectedResult: nil},
		{customOptions: []interface{}{"--table=$(echo 'test')"}, expectedResult: errInvalidOption},
		{customOptions: []interface{}{"--table=test&table"}, expectedResult: errInvalidOption},
		{customOptions: []interface{}{5}, expectedResult: errInvalidOptionType},
	}

	for _, tc := range testCases {
		validationResult := validateCustomOptions(tc.customOptions)

		require.ErrorIs(t, validationResult, tc.expectedResult)
	}
}

func TestDetectConfigChanges(t *testing.T) {
	boolTrue := true
	boolFalse := false
	port1 := uint(6000)
	port2 := uint(6100)

	testCases := []struct {
		name                    string
		oldProj                 *models.ConfigProjection
		newProj                 *models.ConfigProjection
		expectedChangedSettings int
		expectedRestartSettings int
	}{
		{
			name:                    "no changes",
			oldProj:                 &models.ConfigProjection{Debug: &boolTrue},
			newProj:                 &models.ConfigProjection{Debug: &boolTrue},
			expectedChangedSettings: 0,
			expectedRestartSettings: 0,
		},
		{
			name:                    "debug changed",
			oldProj:                 &models.ConfigProjection{Debug: &boolTrue},
			newProj:                 &models.ConfigProjection{Debug: &boolFalse},
			expectedChangedSettings: 1,
			expectedRestartSettings: 0,
		},
		{
			name:                    "port pool changed - requires restart",
			oldProj:                 &models.ConfigProjection{PortPoolFrom: &port1},
			newProj:                 &models.ConfigProjection{PortPoolFrom: &port2},
			expectedChangedSettings: 1,
			expectedRestartSettings: 1,
		},
		{
			name:                    "multiple changes with restart",
			oldProj:                 &models.ConfigProjection{Debug: &boolTrue, PortPoolFrom: &port1},
			newProj:                 &models.ConfigProjection{Debug: &boolFalse, PortPoolFrom: &port2},
			expectedChangedSettings: 2,
			expectedRestartSettings: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			changedSettings, restartSettings := detectConfigChanges(tc.oldProj, tc.newProj)
			require.Equal(t, tc.expectedChangedSettings, len(changedSettings))
			require.Equal(t, tc.expectedRestartSettings, len(restartSettings))
		})
	}
}

func TestGenerateRestartWarnings(t *testing.T) {
	testCases := []struct {
		name             string
		restartSettings  []string
		expectedWarnings int
	}{
		{name: "no restart settings", restartSettings: []string{}, expectedWarnings: 0},
		{name: "one restart setting", restartSettings: []string{"server.port"}, expectedWarnings: 1},
		{name: "multiple restart settings", restartSettings: []string{"server.port", "provision.portPool.from"}, expectedWarnings: 2},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			warnings := generateRestartWarnings(tc.restartSettings)
			require.Equal(t, tc.expectedWarnings, len(warnings))
			for _, warning := range warnings {
				require.Equal(t, "restart", warning.Type)
				require.NotEmpty(t, warning.Message)
			}
		})
	}
}

func TestValidatePortPoolSettings(t *testing.T) {
	validFrom := uint(6000)
	validTo := uint(6100)
	invalidFrom := uint(0)
	invalidTo := uint(5000)

	testCases := []struct {
		name      string
		proj      *models.ConfigProjection
		expectErr bool
	}{
		{name: "valid port pool", proj: &models.ConfigProjection{PortPoolFrom: &validFrom, PortPoolTo: &validTo}, expectErr: false},
		{name: "nil port pool", proj: &models.ConfigProjection{}, expectErr: false},
		{name: "invalid from port", proj: &models.ConfigProjection{PortPoolFrom: &invalidFrom, PortPoolTo: &validTo}, expectErr: true},
		{name: "from greater than to", proj: &models.ConfigProjection{PortPoolFrom: &validTo, PortPoolTo: &invalidTo}, expectErr: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validatePortPoolSettings(tc.proj)
			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateDiagnosticSettings(t *testing.T) {
	validDays := 30
	negativeDays := -1

	testCases := []struct {
		name      string
		proj      *models.ConfigProjection
		expectErr bool
	}{
		{name: "valid retention days", proj: &models.ConfigProjection{LogsRetentionDays: &validDays}, expectErr: false},
		{name: "nil retention days", proj: &models.ConfigProjection{}, expectErr: false},
		{name: "negative retention days", proj: &models.ConfigProjection{LogsRetentionDays: &negativeDays}, expectErr: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateDiagnosticSettings(tc.proj)
			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateEmbeddedUISettings(t *testing.T) {
	validPort := 2345
	invalidPortLow := 0
	invalidPortHigh := 70000

	testCases := []struct {
		name      string
		proj      *models.ConfigProjection
		expectErr bool
	}{
		{name: "valid port", proj: &models.ConfigProjection{EmbeddedUIPort: &validPort}, expectErr: false},
		{name: "nil port", proj: &models.ConfigProjection{}, expectErr: false},
		{name: "port too low", proj: &models.ConfigProjection{EmbeddedUIPort: &invalidPortLow}, expectErr: true},
		{name: "port too high", proj: &models.ConfigProjection{EmbeddedUIPort: &invalidPortHigh}, expectErr: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateEmbeddedUISettings(tc.proj)
			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateWebhookSettings(t *testing.T) {
	testCases := []struct {
		name      string
		proj      *models.ConfigProjection
		expectErr bool
	}{
		{name: "no webhooks", proj: &models.ConfigProjection{}, expectErr: false},
		{
			name: "valid webhook",
			proj: &models.ConfigProjection{
				WebhooksHooks: []models.WebhookHookProjection{{URL: "http://example.com", Trigger: []string{"clone.created"}}},
			},
			expectErr: false,
		},
		{
			name: "empty url",
			proj: &models.ConfigProjection{
				WebhooksHooks: []models.WebhookHookProjection{{URL: "", Trigger: []string{"clone.created"}}},
			},
			expectErr: true,
		},
		{
			name: "empty trigger",
			proj: &models.ConfigProjection{
				WebhooksHooks: []models.WebhookHookProjection{{URL: "http://example.com", Trigger: []string{}}},
			},
			expectErr: true,
		},
		{
			name: "invalid trigger",
			proj: &models.ConfigProjection{
				WebhooksHooks: []models.WebhookHookProjection{{URL: "http://example.com", Trigger: []string{"invalid.trigger"}}},
			},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateWebhookSettings(tc.proj)
			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestHasRetrievalSettings(t *testing.T) {
	testCases := []struct {
		name     string
		objMap   map[string]interface{}
		expected bool
	}{
		{name: "empty map", objMap: map[string]interface{}{}, expected: false},
		{name: "has debug only", objMap: map[string]interface{}{"debug": true}, expected: false},
		{name: "has host", objMap: map[string]interface{}{"host": "localhost"}, expected: true},
		{name: "has timetable", objMap: map[string]interface{}{"timetable": "0 0 * * *"}, expected: true},
		{name: "has databases", objMap: map[string]interface{}{"databases": []string{"db1"}}, expected: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := hasRetrievalSettings(tc.objMap)
			require.Equal(t, tc.expected, result)
		})
	}
}
