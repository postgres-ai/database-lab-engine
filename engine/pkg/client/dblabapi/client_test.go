package dblabapi

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// roundTripFunc represents a mock type.
type roundTripFunc func(req *http.Request) *http.Response

// RoundTrip is a mock function.
func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

// NewTestClient returns a mock of *http.Client.
func NewTestClient(fn roundTripFunc) *http.Client {
	return &http.Client{
		Transport: fn,
	}
}

func TestNewClient(t *testing.T) {
	// The test case also checks if the client can be work with a no-ideal URL.
	c, err := NewClient(Options{
		Host:              "https://example.com//",
		VerificationToken: "testVerify",
		RequestTimeout:    30 * time.Second,
	})
	require.NoError(t, err)

	assert.IsType(t, &Client{}, c)
	assert.Equal(t, "https://example.com", c.url.String())
	assert.Equal(t, "testVerify", c.verificationToken)
	assert.Equal(t, 30*time.Second, c.requestTimeout)
	assert.IsType(t, &http.Client{}, c.client)
}

func TestClientURL(t *testing.T) {
	c, err := NewClient(Options{
		Host:              "https://example.com/",
		VerificationToken: "testVerify",
	})
	require.NoError(t, err)

	assert.Equal(t, "https://example.com/test-url", c.URL("test-url").String())
}

func TestClient_RequestTimeoutBoundsAHangingServer(t *testing.T) {
	release := make(chan struct{})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		<-release
		w.WriteHeader(http.StatusOK)
	}))
	// Close waits for the handler, so the handler has to be released first.
	defer server.Close()
	defer close(release)

	c, err := NewClient(Options{Host: server.URL, RequestTimeout: 200 * time.Millisecond})
	require.NoError(t, err)

	started := time.Now()
	_, err = c.Status(context.Background())
	require.Error(t, err)
	assert.Less(t, time.Since(started), 5*time.Second, "the request must fail on the timeout, not hang")
}

func TestClient_DownloadArtifactStreamsPastTheRequestTimeout(t *testing.T) {
	const chunks = 5

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		flusher, ok := w.(http.Flusher)
		require.True(t, ok)

		w.WriteHeader(http.StatusOK)
		flusher.Flush()

		for i := 0; i < chunks; i++ {
			time.Sleep(100 * time.Millisecond)
			_, _ = w.Write([]byte("chunk\n"))
			flusher.Flush()
		}
	}))
	defer server.Close()

	c, err := NewClient(Options{Host: server.URL, RequestTimeout: 200 * time.Millisecond})
	require.NoError(t, err)

	body, err := c.DownloadArtifact(context.Background(), "clone", "session", "artifact")
	require.NoError(t, err)

	defer body.Close()

	content, err := io.ReadAll(body)
	require.NoError(t, err, "the streamed body must not be cut by the request timeout")
	assert.Equal(t, strings.Repeat("chunk\n", chunks), string(content))
}

func TestClient_DownloadArtifactWaitsForHeadersWithinTheRequestTimeout(t *testing.T) {
	release := make(chan struct{})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		<-release
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	defer close(release)

	c, err := NewClient(Options{Host: server.URL, RequestTimeout: 200 * time.Millisecond})
	require.NoError(t, err)

	started := time.Now()
	_, err = c.DownloadArtifact(context.Background(), "clone", "session", "artifact")
	require.Error(t, err)
	assert.Less(t, time.Since(started), 5*time.Second)
}
