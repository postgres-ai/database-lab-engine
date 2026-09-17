package webhooks

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
)

const testTimeout = 5 * time.Second

type receivedRequest struct {
	token string
	event BasicEvent
}

func newRecordingServer(t *testing.T) (*httptest.Server, <-chan receivedRequest) {
	t.Helper()

	received := make(chan receivedRequest, 16)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var event BasicEvent

		require.NoError(t, json.NewDecoder(r.Body).Decode(&event))

		select {
		case received <- receivedRequest{token: r.Header.Get(DLEWebhookTokenHeader), event: event}:
		default:
		}

		_, _ = w.Write([]byte("ok"))
	}))

	t.Cleanup(srv.Close)

	return srv, received
}

func waitForRequest(t *testing.T, received <-chan receivedRequest) receivedRequest {
	t.Helper()

	select {
	case req := <-received:
		return req
	case <-time.After(testTimeout):
		t.Fatal("webhook request has not arrived")
	}

	return receivedRequest{}
}

func TestReloadRegistersOnlyValidHooks(t *testing.T) {
	cfg := &Config{Hooks: []Hook{
		{URL: "https://example.com/hook", Trigger: []string{CloneCreatedEvent, CloneDeleteEvent}},
		{URL: "not-a-url", Trigger: []string{CloneCreatedEvent}},
		{URL: "/relative/path", Trigger: []string{CloneResetEvent}},
	}}

	s := NewService(cfg, nil)

	assert.Len(t, s.hooksFor(CloneCreatedEvent), 1)
	assert.Len(t, s.hooksFor(CloneDeleteEvent), 1)
	assert.Empty(t, s.hooksFor(CloneResetEvent))

	s.Reload(&Config{})

	assert.Empty(t, s.hooksFor(CloneCreatedEvent))
	assert.Empty(t, s.hooksFor(CloneDeleteEvent))
}

func TestHooksForReturnsCopy(t *testing.T) {
	s := NewService(&Config{Hooks: []Hook{{URL: "https://example.com/a", Trigger: []string{CloneCreatedEvent}}}}, nil)

	hooks := s.hooksFor(CloneCreatedEvent)
	require.Len(t, hooks, 1)

	hooks[0].URL = "https://example.com/changed"

	assert.Equal(t, "https://example.com/a", s.hooksFor(CloneCreatedEvent)[0].URL)
}

func TestRunDispatchesHooks(t *testing.T) {
	srv, received := newRecordingServer(t)

	eventCh := make(chan EventTyper)
	cfg := &Config{Hooks: []Hook{{URL: srv.URL, Secret: "secret", Trigger: []string{CloneCreatedEvent}}}}
	s := NewService(cfg, eventCh)

	done := make(chan struct{})

	go func() {
		defer close(done)

		s.Run(context.Background())
	}()

	eventCh <- BasicEvent{EventType: CloneCreatedEvent, EntityID: "clone-1"}
	eventCh <- BasicEvent{EventType: CloneDeleteEvent, EntityID: "clone-2"}

	req := waitForRequest(t, received)
	assert.Equal(t, "secret", req.token)
	assert.Equal(t, BasicEvent{EventType: CloneCreatedEvent, EntityID: "clone-1"}, req.event)

	close(eventCh)
	<-done

	select {
	case req := <-received:
		t.Fatalf("unexpected webhook request for event %q", req.event.EventType)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestTriggerWebhookHandlesNon2xx(t *testing.T) {
	var closedBody atomic.Bool

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	}))
	t.Cleanup(srv.Close)

	s := NewService(&Config{}, nil)
	s.client.Transport = roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		resp, err := http.DefaultTransport.RoundTrip(r)
		if err != nil {
			return nil, err
		}

		resp.Body = &closeTrackingBody{ReadCloser: resp.Body, closed: &closedBody}

		return resp, nil
	})

	logBuf := &bytes.Buffer{}

	log.SetOutput(logBuf)
	log.SetDebug(false)
	t.Cleanup(func() {
		log.ResetOutput()
		log.SetDebug(true)
	})

	s.triggerWebhook(context.Background(), Hook{URL: srv.URL}, BasicEvent{EventType: CloneCreatedEvent})

	assert.True(t, closedBody.Load(), "response body must be closed")
	assert.Contains(t, logBuf.String(), "responded with status 500")
	assert.NotContains(t, logBuf.String(), "boom", "response body must stay out of non-debug logs")
}

func TestNewClientBoundsEveryStage(t *testing.T) {
	client := newClient()

	assert.Equal(t, requestTimeout, client.Timeout)

	transport, ok := client.Transport.(*http.Transport)
	require.True(t, ok)

	assert.Equal(t, requestTimeout, transport.ResponseHeaderTimeout)
	assert.Equal(t, tlsHandshakeTimeout, transport.TLSHandshakeTimeout)
	assert.Equal(t, idleConnTimeout, transport.IdleConnTimeout)
}

func TestMakeRequestTimesOut(t *testing.T) {
	release := make(chan struct{})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-release:
		case <-r.Context().Done():
		}
	}))
	t.Cleanup(func() {
		close(release)
		srv.Close()
	})

	s := NewService(&Config{}, nil)
	s.client.Timeout = 50 * time.Millisecond

	start := time.Now()

	resp, err := s.makeRequest(context.Background(), Hook{URL: srv.URL}, BasicEvent{EventType: CloneCreatedEvent})
	if resp != nil {
		_ = resp.Body.Close()
	}

	require.Error(t, err)
	assert.Less(t, time.Since(start), testTimeout)

	var netErr net.Error

	require.True(t, errors.As(err, &netErr))
	assert.True(t, netErr.Timeout())
}

func TestReloadConcurrentWithRun(t *testing.T) {
	srv, received := newRecordingServer(t)

	eventCh := make(chan EventTyper)
	cfg := &Config{Hooks: []Hook{{URL: srv.URL, Trigger: []string{CloneCreatedEvent}}}}
	s := NewService(cfg, eventCh)

	var wg sync.WaitGroup

	wg.Add(1)

	go func() {
		defer wg.Done()

		s.Run(context.Background())
	}()

	const iterations = 200

	wg.Add(1)

	go func() {
		defer wg.Done()

		for i := 0; i < iterations; i++ {
			s.Reload(cfg)
		}
	}()

	for i := 0; i < iterations; i++ {
		eventCh <- BasicEvent{EventType: CloneCreatedEvent, EntityID: "clone"}
	}

	close(eventCh)
	wg.Wait()

	waitForRequest(t, received)
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

type closeTrackingBody struct {
	io.ReadCloser
	closed *atomic.Bool
}

func (b *closeTrackingBody) Close() error {
	b.closed.Store(true)

	return b.ReadCloser.Close()
}
