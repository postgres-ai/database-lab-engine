/*
2021 © Postgres.ai
*/

package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLevelByAlertType(t *testing.T) {
	testCases := []struct {
		alertType AlertType
		level     AlertLevel
	}{
		{
			alertType: "refresh_failed",
			level:     ErrorLevel,
		},
		{
			alertType: "refresh_skipped",
			level:     WarningLevel,
		},
		{
			alertType: "unknown_fail",
			level:     UnknownLevel,
		},
	}

	for _, tc := range testCases {
		level := AlertLevelByType(tc.alertType)
		assert.Equal(t, tc.level, level)
	}
}
