/*
2020 © Postgres.ai
*/

package tools

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"iter"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	cerrdefs "github.com/containerd/errdefs"
	"github.com/moby/moby/api/types/container"
	imagetypes "github.com/moby/moby/api/types/image"
	"github.com/moby/moby/api/types/jsonstream"
	"github.com/moby/moby/api/types/mount"
	"github.com/moby/moby/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIfDirectoryEmpty(t *testing.T) {
	dirName, err := os.MkdirTemp("", "test")
	defer os.RemoveAll(dirName)

	require.NoError(t, err)

	// Check if the directory is empty.
	isEmpty, err := IsEmptyDirectory(dirName)
	require.NoError(t, err)
	assert.True(t, isEmpty)

	// Create a new file.
	_, err = os.CreateTemp(dirName, "testFile*")
	require.NoError(t, err)

	// Check if the directory is not empty.
	isEmpty, err = IsEmptyDirectory(dirName)
	require.NoError(t, err)
	assert.False(t, isEmpty)
}

func TestGetMountsFromMountPoints(t *testing.T) {
	testCases := []struct {
		name           string
		dataDir        string
		mountPoints    []container.MountPoint
		expectedPoints []mount.Mount
	}{
		{
			name:    "simple mount without transformation",
			dataDir: "/var/lib/dblab/clones/dblab_clone_6000/data",
			mountPoints: []container.MountPoint{{
				Type:        mount.TypeBind,
				Source:      "/var/lib/pgsql/data",
				Destination: "/var/lib/postgresql/data",
			}},
			expectedPoints: []mount.Mount{{
				Type:     mount.TypeBind,
				Source:   "/var/lib/pgsql/data",
				Target:   "/var/lib/postgresql/data",
				ReadOnly: true,
				BindOptions: &mount.BindOptions{
					Propagation: "",
				},
			}},
		},
		{
			name:    "mount with path transformation",
			dataDir: "/var/lib/dblab/clones/dblab_clone_6000/data",
			mountPoints: []container.MountPoint{{
				Type:        mount.TypeBind,
				Source:      "/var/lib/postgresql",
				Destination: "/var/lib/dblab",
			}},
			expectedPoints: []mount.Mount{{
				Type:     mount.TypeBind,
				Source:   "/var/lib/postgresql/clones/dblab_clone_6000/data",
				Target:   "/var/lib/dblab/clones/dblab_clone_6000/data",
				ReadOnly: true,
				BindOptions: &mount.BindOptions{
					Propagation: "",
				},
			}},
		},
		{
			name:    "deduplicate identical mounts",
			dataDir: "/var/lib/dblab/data",
			mountPoints: []container.MountPoint{
				{Type: mount.TypeBind, Source: "/host/dump", Destination: "/var/lib/dblab/dump"},
				{Type: mount.TypeBind, Source: "/host/dump", Destination: "/var/lib/dblab/dump"},
			},
			expectedPoints: []mount.Mount{{
				Type:     mount.TypeBind,
				Source:   "/host/dump",
				Target:   "/var/lib/dblab/dump",
				ReadOnly: true,
				BindOptions: &mount.BindOptions{
					Propagation: "",
				},
			}},
		},
		{
			name:    "deduplicate mounts with trailing slashes",
			dataDir: "/var/lib/dblab/data",
			mountPoints: []container.MountPoint{
				{Type: mount.TypeBind, Source: "/host/dump/", Destination: "/var/lib/dblab/dump"},
				{Type: mount.TypeBind, Source: "/host/dump", Destination: "/var/lib/dblab/dump/"},
			},
			expectedPoints: []mount.Mount{{
				Type:     mount.TypeBind,
				Source:   "/host/dump/",
				Target:   "/var/lib/dblab/dump",
				ReadOnly: true,
				BindOptions: &mount.BindOptions{
					Propagation: "",
				},
			}},
		},
		{
			name:    "volume mount uses name instead of path",
			dataDir: "/var/lib/dblab/data",
			mountPoints: []container.MountPoint{{
				Type:        mount.TypeVolume,
				Name:        "3749a7e336f27d8c1ce2a81c7b945954f7522ecc3a4be4a3855bf64473f63a89",
				Source:      "/var/lib/docker/volumes/3749a7e336f27d8c1ce2a81c7b945954f7522ecc3a4be4a3855bf64473f63a89/_data",
				Destination: "/var/lib/docker",
				RW:          false,
			}},
			expectedPoints: []mount.Mount{{
				Type:     mount.TypeVolume,
				Source:   "3749a7e336f27d8c1ce2a81c7b945954f7522ecc3a4be4a3855bf64473f63a89",
				Target:   "/var/lib/docker",
				ReadOnly: true,
			}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mounts := GetMountsFromMountPoints(tc.dataDir, tc.mountPoints)
			assert.Equal(t, tc.expectedPoints, mounts)
		})
	}
}

func TestTrimPort(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "hostname with port", input: "localhost:5432", expected: "localhost"},
		{name: "hostname without port", input: "localhost", expected: "localhost"},
		{name: "ip with port", input: "192.168.1.1:5432", expected: "192.168.1.1"},
		{name: "ip without port", input: "192.168.1.1", expected: "192.168.1.1"},
		{name: "empty string", input: "", expected: ""},
		{name: "only colon", input: ":", expected: ""},
		{name: "hostname with multiple colons", input: "host:5432:extra", expected: "host"},
		{name: "fqdn with port", input: "db.example.com:5432", expected: "db.example.com"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, TrimPort(tc.input))
		})
	}
}

func TestGeneratePassword(t *testing.T) {
	pwd, err := GeneratePassword()
	require.NoError(t, err)
	assert.Len(t, pwd, passwordLength)

	t.Run("uniqueness", func(t *testing.T) {
		pwd2, err := GeneratePassword()
		require.NoError(t, err)
		assert.NotEqual(t, pwd, pwd2, "two generated passwords should differ")
	})

	t.Run("contains digits", func(t *testing.T) {
		digitCount := 0
		for _, ch := range pwd {
			if ch >= '0' && ch <= '9' {
				digitCount++
			}
		}
		assert.GreaterOrEqual(t, digitCount, passwordMinDigits)
	})
}

func TestErrHealthCheck_Error(t *testing.T) {
	testCases := []struct {
		name     string
		err      ErrHealthCheck
		expected string
	}{
		{name: "exit code 2", err: ErrHealthCheck{ExitCode: 2, Output: "no response"}, expected: "health check failed. Code: 2, Output: no response"},
		{name: "exit code 3", err: ErrHealthCheck{ExitCode: 3, Output: "invalid params"}, expected: "health check failed. Code: 3, Output: invalid params"},
		{name: "empty output", err: ErrHealthCheck{ExitCode: 1, Output: ""}, expected: "health check failed. Code: 1, Output: "},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.err.Error())
		})
	}
}

func TestCleanupDir(t *testing.T) {
	dirName, err := os.MkdirTemp("", "test_cleanup")
	require.NoError(t, err)
	defer os.RemoveAll(dirName)

	_, err = os.CreateTemp(dirName, "file1_*")
	require.NoError(t, err)

	err = os.Mkdir(filepath.Join(dirName, "subdir"), 0755)
	require.NoError(t, err)

	isEmpty, err := IsEmptyDirectory(dirName)
	require.NoError(t, err)
	assert.False(t, isEmpty)

	err = CleanupDir(dirName)
	require.NoError(t, err)

	isEmpty, err = IsEmptyDirectory(dirName)
	require.NoError(t, err)
	assert.True(t, isEmpty)

	t.Run("non-existent directory", func(t *testing.T) {
		err := CleanupDir("/nonexistent/path/12345")
		assert.Error(t, err)
	})
}

func TestTouchFile(t *testing.T) {
	dirName, err := os.MkdirTemp("", "test_touch")
	require.NoError(t, err)
	defer os.RemoveAll(dirName)

	filePath := filepath.Join(dirName, "newfile.txt")
	err = TouchFile(filePath)
	require.NoError(t, err)

	info, err := os.Stat(filePath)
	require.NoError(t, err)
	assert.Equal(t, int64(0), info.Size())

	t.Run("invalid path", func(t *testing.T) {
		err := TouchFile("/nonexistent/dir/file.txt")
		assert.Error(t, err)
	})
}

func TestDiscoverDataStateAt(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
		hasError bool
	}{
		{
			name:     "valid archive header",
			input:    "-- some line\n;; Archive created at 2023-01-15 10:30:00 UTC\n-- other line",
			expected: "20230115103000",
		},
		{
			name:     "no archive header",
			input:    "-- some line\n-- other line",
			hasError: true,
		},
		{
			name:     "empty input",
			input:    "",
			hasError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := DiscoverDataStateAt(bytes.NewBufferString(tc.input))
			if tc.hasError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestParseDataStateAt(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
		hasError bool
	}{
		{name: "valid label", input: "Archive created at 2023-06-20 14:30:00 UTC", expected: "20230620143000"},
		{name: "no matching prefix", input: "some other text", expected: ""},
		{name: "invalid date format", input: "Archive created at not-a-date", hasError: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := parseDataStateAt(tc.input)
			if tc.hasError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsEmptyDirectory_NonExistent(t *testing.T) {
	_, err := IsEmptyDirectory("/nonexistent/path/12345")
	assert.Error(t, err)
}

type inspectFn func(ctx context.Context, imageID string, opts ...client.ImageInspectOption) (client.ImageInspectResult, error)
type pullFn func(ctx context.Context, refStr string, options client.ImagePullOptions) (client.ImagePullResponse, error)

// fakePullResponse implements client.ImagePullResponse over a plain reader.
type fakePullResponse struct {
	io.ReadCloser
}

func (f fakePullResponse) JSONMessages(context.Context) iter.Seq2[jsonstream.Message, error] {
	return func(func(jsonstream.Message, error) bool) {}
}

func (f fakePullResponse) Wait(context.Context) error { return nil }

// fakePuller implements imagePuller for testing PullImage in isolation.
type fakePuller struct {
	inspect    inspectFn
	pull       pullFn
	pullCalled bool
}

func (f *fakePuller) ImageInspect(ctx context.Context, imageID string, opts ...client.ImageInspectOption) (client.ImageInspectResult, error) {
	if f.inspect != nil {
		return f.inspect(ctx, imageID, opts...)
	}
	return client.ImageInspectResult{}, errors.New("inspect not implemented")
}

func (f *fakePuller) ImagePull(ctx context.Context, refStr string, options client.ImagePullOptions) (client.ImagePullResponse, error) {
	f.pullCalled = true
	if f.pull != nil {
		return f.pull(ctx, refStr, options)
	}
	return nil, errors.New("pull not implemented")
}

func TestPullImage(t *testing.T) {
	notFound := fmt.Errorf("no such image: %w", cerrdefs.ErrNotFound)
	wrappedNotFound := fmt.Errorf("inspect failed: %w", notFound)

	localImage := func(context.Context, string, ...client.ImageInspectOption) (client.ImageInspectResult, error) {
		return client.ImageInspectResult{InspectResponse: imagetypes.InspectResponse{ID: "sha256:abc"}}, nil
	}
	notFoundInspect := func(context.Context, string, ...client.ImageInspectOption) (client.ImageInspectResult, error) {
		return client.ImageInspectResult{}, notFound
	}
	wrappedNotFoundInspect := func(context.Context, string, ...client.ImageInspectOption) (client.ImageInspectResult, error) {
		return client.ImageInspectResult{}, wrappedNotFound
	}
	daemonDownInspect := func(context.Context, string, ...client.ImageInspectOption) (client.ImageInspectResult, error) {
		return client.ImageInspectResult{}, errors.New("daemon down")
	}

	emptyStream := func(context.Context, string, client.ImagePullOptions) (client.ImagePullResponse, error) {
		return fakePullResponse{io.NopCloser(strings.NewReader(""))}, nil
	}
	errorStream := func(context.Context, string, client.ImagePullOptions) (client.ImagePullResponse, error) {
		body := `{"errorDetail":{"message":"manifest for postgresai/extended-postgres:99-bogus not found: manifest unknown"},"error":"manifest unknown"}`
		return fakePullResponse{io.NopCloser(strings.NewReader(body))}, nil
	}

	testCases := []struct {
		name            string
		inspect         inspectFn
		pull            pullFn
		wantErr         bool
		wantErrContains string
		wantPullCalled  bool
	}{
		{name: "image already local skips pull", inspect: localImage},
		{name: "not found triggers pull", inspect: notFoundInspect, pull: emptyStream, wantPullCalled: true},
		{name: "wrapped not found triggers pull", inspect: wrappedNotFoundInspect, pull: emptyStream, wantPullCalled: true},
		{name: "non-not-found inspect error propagates", inspect: daemonDownInspect, wantErr: true, wantErrContains: "failed to inspect image"},
		{name: "pull stream error propagates", inspect: notFoundInspect, pull: errorStream, wantErr: true, wantErrContains: "failed to pull image", wantPullCalled: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			puller := &fakePuller{inspect: tc.inspect, pull: tc.pull}

			err := PullImage(context.Background(), puller, "postgresai/extended-postgres:test")

			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErrContains)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tc.wantPullCalled, puller.pullCalled)
		})
	}
}

type panicReader struct{}

func (panicReader) Read([]byte) (int, error) { panic("broken stream") }

func TestProcessAttachResponse_ReturnsErrorOnPanic(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := processAttachResponse(ctx, panicReader{})

	assert.ErrorContains(t, err, "panic in")
}

func TestProcessAttachResponse_CopyEndsAfterCancel(t *testing.T) {
	reader, writer := io.Pipe()
	defer func() { _ = writer.Close() }()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := processAttachResponse(ctx, reader)
	require.ErrorIs(t, err, context.Canceled)

	assert.Eventually(t, func() bool {
		stacks := make([]byte, 1<<20)

		return !strings.Contains(string(stacks[:runtime.Stack(stacks, true)]), "tools.processAttachResponse")
	}, 5*time.Second, 10*time.Millisecond)
}
