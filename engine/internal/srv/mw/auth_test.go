/*
2019 © Postgres.ai
*/

package mw

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test constants.
const (
	testVerificationToken   = "TestToken"
	testPlatformAccessToken = "PlatformAccessToken"
)

// MockPersonalTokenVerifier mocks personal verifier methods.
type MockPersonalTokenVerifier struct {
	isPersonalTokenEnabled bool
}

func (m MockPersonalTokenVerifier) IsAllowedToken(_ context.Context, token string) bool {
	return testPlatformAccessToken == token
}

func (m MockPersonalTokenVerifier) IsPersonalTokenEnabled() bool {
	return m.isPersonalTokenEnabled
}

func TestAccess(t *testing.T) {
	testCases := []struct {
		name                   string
		requestToken           string
		isPersonalTokenEnabled bool
		result                 bool
	}{
		{isPersonalTokenEnabled: false, requestToken: "", result: false, name: "empty RequestToken with disabled PersonalToken"},
		{isPersonalTokenEnabled: false, requestToken: "WrongToken", result: false, name: "wrong RequestToken with disabled PersonalToken"},
		{isPersonalTokenEnabled: false, requestToken: "TestToken", result: true, name: "correct RequestToken with disabled PersonalToken"},
		{isPersonalTokenEnabled: true, requestToken: "", result: false, name: "empty RequestToken with enabled PersonalToken"},
		{isPersonalTokenEnabled: true, requestToken: "WrongToken", result: false, name: "wrong RequestToken with enabled PersonalToken"},
		{isPersonalTokenEnabled: true, requestToken: "TestToken", result: true, name: "correct RequestToken with enabled PersonalToken"},
		{isPersonalTokenEnabled: true, requestToken: "PlatformAccessToken", result: true, name: "correct PersonalToken with enabled PersonalToken"},
	}

	mw := Auth{
		verificationToken: testVerificationToken,
	}

	for _, tc := range testCases {
		t.Log(tc.name)
		mw.personalTokenVerifier = MockPersonalTokenVerifier{isPersonalTokenEnabled: tc.result}

		isAllowed := mw.isAccessAllowed(context.Background(), tc.requestToken)
		assert.Equal(t, tc.result, isAllowed)
	}
}
