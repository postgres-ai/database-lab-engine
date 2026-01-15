/*
2025 © Postgres.ai
*/

package objbacker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
)

// VDEVState represents the state of an objbacker VDEV.
type VDEVState string

const (
	// VDEVStateUnknown indicates the VDEV state is unknown.
	VDEVStateUnknown VDEVState = "unknown"
	// VDEVStateOnline indicates the VDEV is online and healthy.
	VDEVStateOnline VDEVState = "online"
	// VDEVStateOffline indicates the VDEV is offline.
	VDEVStateOffline VDEVState = "offline"
	// VDEVStateDegraded indicates the VDEV is degraded.
	VDEVStateDegraded VDEVState = "degraded"
	// VDEVStateFaulted indicates the VDEV has faulted.
	VDEVStateFaulted VDEVState = "faulted"
)

// VDEVInfo contains information about an objbacker VDEV.
type VDEVInfo struct {
	ID            string            `json:"id"`
	PoolName      string            `json:"poolName"`
	State         VDEVState         `json:"state"`
	StorageType   StorageType       `json:"storageType"`
	Bucket        string            `json:"bucket"`
	Prefix        string            `json:"prefix"`
	BytesRead     uint64            `json:"bytesRead"`
	BytesWritten  uint64            `json:"bytesWritten"`
	OpsRead       uint64            `json:"opsRead"`
	OpsWrite      uint64            `json:"opsWrite"`
	CacheHitRatio float64           `json:"cacheHitRatio"`
	LastError     string            `json:"lastError,omitempty"`
	CreatedAt     time.Time         `json:"createdAt"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// Manager handles objbacker VDEV operations.
type Manager struct {
	config     Config
	mu         sync.RWMutex
	vdevs      map[string]*VDEVInfo
	daemonProc *os.Process
}

// NewManager creates a new objbacker Manager.
func NewManager(config Config) (*Manager, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	m := &Manager{
		config: config,
		vdevs:  make(map[string]*VDEVInfo),
	}

	return m, nil
}

// Config returns the current configuration.
func (m *Manager) Config() Config {
	return m.config
}

// IsEnabled returns true if objbacker is enabled.
func (m *Manager) IsEnabled() bool {
	return m.config.Enabled
}

// Initialize prepares the objbacker environment.
func (m *Manager) Initialize(ctx context.Context) error {
	if !m.config.Enabled {
		log.Dbg("objbacker: disabled, skipping initialization")
		return nil
	}

	log.Msg("objbacker: initializing")

	if err := m.ensureCacheDirectory(); err != nil {
		return errors.Wrap(err, "failed to create cache directory")
	}

	if err := m.checkDeviceAvailable(); err != nil {
		return errors.Wrap(err, "objbacker device not available")
	}

	if err := m.startDaemon(ctx); err != nil {
		return errors.Wrap(err, "failed to start objbacker daemon")
	}

	log.Msg("objbacker: initialization complete")
	return nil
}

// Shutdown gracefully shuts down objbacker.
func (m *Manager) Shutdown(ctx context.Context) error {
	if !m.config.Enabled {
		return nil
	}

	log.Msg("objbacker: shutting down")

	if err := m.stopDaemon(ctx); err != nil {
		log.Err("objbacker: error stopping daemon", err)
	}

	return nil
}

// CreateVDEV creates a new objbacker VDEV for use in a ZFS pool.
func (m *Manager) CreateVDEV(ctx context.Context, poolName string) (*VDEVInfo, error) {
	if !m.config.Enabled {
		return nil, errors.New("objbacker is not enabled")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	vdevID := fmt.Sprintf("objbacker_%s_%d", poolName, time.Now().UnixNano())

	vdev := &VDEVInfo{
		ID:          vdevID,
		PoolName:    poolName,
		State:       VDEVStateOffline,
		StorageType: m.config.StorageType,
		Bucket:      m.config.Bucket,
		Prefix:      m.config.ObjectPath(poolName),
		CreatedAt:   time.Now(),
		Metadata:    make(map[string]string),
	}

	if err := m.initializeVDEVStorage(ctx, vdev); err != nil {
		return nil, errors.Wrap(err, "failed to initialize VDEV storage")
	}

	vdev.State = VDEVStateOnline
	m.vdevs[vdevID] = vdev

	log.Msg("objbacker: created VDEV", vdevID, "pool:", poolName)

	return vdev, nil
}

// GetVDEV returns information about a specific VDEV.
func (m *Manager) GetVDEV(vdevID string) (*VDEVInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	vdev, ok := m.vdevs[vdevID]
	if !ok {
		return nil, fmt.Errorf("VDEV not found: %s", vdevID)
	}

	return vdev, nil
}

// ListVDEVs returns all managed VDEVs.
func (m *Manager) ListVDEVs() []*VDEVInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	vdevs := make([]*VDEVInfo, 0, len(m.vdevs))
	for _, v := range m.vdevs {
		vdevs = append(vdevs, v)
	}

	return vdevs
}

// DestroyVDEV removes an objbacker VDEV.
func (m *Manager) DestroyVDEV(ctx context.Context, vdevID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	vdev, ok := m.vdevs[vdevID]
	if !ok {
		return fmt.Errorf("VDEV not found: %s", vdevID)
	}

	if err := m.cleanupVDEVStorage(ctx, vdev); err != nil {
		log.Err("objbacker: error cleaning up VDEV storage", err)
	}

	delete(m.vdevs, vdevID)

	log.Msg("objbacker: destroyed VDEV", vdevID)

	return nil
}

// GetStats returns statistics for a VDEV.
func (m *Manager) GetStats(vdevID string) (*VDEVStats, error) {
	m.mu.RLock()
	vdev, ok := m.vdevs[vdevID]
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("VDEV not found: %s", vdevID)
	}

	stats := &VDEVStats{
		VDEVID:        vdevID,
		BytesRead:     vdev.BytesRead,
		BytesWritten:  vdev.BytesWritten,
		OpsRead:       vdev.OpsRead,
		OpsWrite:      vdev.OpsWrite,
		CacheHitRatio: vdev.CacheHitRatio,
		Timestamp:     time.Now(),
	}

	return stats, nil
}

// VDEVStats contains performance statistics for a VDEV.
type VDEVStats struct {
	VDEVID        string    `json:"vdevId"`
	BytesRead     uint64    `json:"bytesRead"`
	BytesWritten  uint64    `json:"bytesWritten"`
	OpsRead       uint64    `json:"opsRead"`
	OpsWrite      uint64    `json:"opsWrite"`
	CacheHitRatio float64   `json:"cacheHitRatio"`
	Timestamp     time.Time `json:"timestamp"`
}

// GenerateVDEVPath returns the device path for creating a ZFS pool.
func (m *Manager) GenerateVDEVPath(vdevID string) string {
	return fmt.Sprintf("%s:%s", m.config.DevicePath, vdevID)
}

func (m *Manager) ensureCacheDirectory() error {
	cachePath := m.config.Performance.LocalCachePath
	if cachePath == "" {
		return errors.New("cache path not configured")
	}

	if err := os.MkdirAll(cachePath, 0750); err != nil {
		return errors.Wrap(err, "failed to create cache directory")
	}

	return nil
}

func (m *Manager) checkDeviceAvailable() error {
	_, err := os.Stat(m.config.DevicePath)
	if os.IsNotExist(err) {
		return fmt.Errorf("objbacker device not found at %s - is the kernel module loaded?", m.config.DevicePath)
	}

	return err
}

func (m *Manager) startDaemon(ctx context.Context) error {
	daemonPath := m.findDaemonPath()
	if daemonPath == "" {
		log.Dbg("objbacker: daemon not found, assuming external management")
		return nil
	}

	args := m.buildDaemonArgs()

	cmd := exec.CommandContext(ctx, daemonPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return errors.Wrap(err, "failed to start objbacker daemon")
	}

	m.daemonProc = cmd.Process

	log.Msg("objbacker: daemon started, pid:", cmd.Process.Pid)

	return nil
}

func (m *Manager) stopDaemon(ctx context.Context) error {
	if m.daemonProc == nil {
		return nil
	}

	if err := m.daemonProc.Signal(os.Interrupt); err != nil {
		return errors.Wrap(err, "failed to signal daemon")
	}

	done := make(chan error, 1)

	go func() {
		_, err := m.daemonProc.Wait()
		done <- err
	}()

	select {
	case <-ctx.Done():
		if err := m.daemonProc.Kill(); err != nil {
			return errors.Wrap(err, "failed to kill daemon")
		}
	case err := <-done:
		if err != nil {
			return errors.Wrap(err, "daemon exited with error")
		}
	}

	return nil
}

func (m *Manager) findDaemonPath() string {
	candidates := []string{
		"/usr/local/bin/zfs_objbacker_daemon",
		"/usr/bin/zfs_objbacker_daemon",
		"/opt/objbacker/bin/daemon",
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

func (m *Manager) buildDaemonArgs() []string {
	args := []string{
		"--device", m.config.DevicePath,
		"--socket", m.config.DaemonSocketPath,
		"--storage-type", string(m.config.StorageType),
		"--bucket", m.config.Bucket,
		"--cache-path", m.config.Performance.LocalCachePath,
		"--cache-size", fmt.Sprintf("%d", m.config.Performance.LocalCacheSize),
		"--block-size", fmt.Sprintf("%d", m.config.Performance.BlockSize),
		"--max-concurrent-ops", fmt.Sprintf("%d", m.config.Performance.MaxConcurrentOps),
	}

	if m.config.Endpoint != "" {
		args = append(args, "--endpoint", m.config.Endpoint)
	}

	if m.config.Region != "" {
		args = append(args, "--region", m.config.Region)
	}

	if m.config.Prefix != "" {
		args = append(args, "--prefix", m.config.Prefix)
	}

	if m.config.Credentials.UseIAMRole {
		args = append(args, "--use-iam-role")
	} else if m.config.Credentials.CredentialsFile != "" {
		args = append(args, "--credentials-file", m.config.Credentials.CredentialsFile)
	}

	return args
}

func (m *Manager) initializeVDEVStorage(ctx context.Context, vdev *VDEVInfo) error {
	marker := &storageMarker{
		VDEVID:      vdev.ID,
		PoolName:    vdev.PoolName,
		StorageType: vdev.StorageType,
		CreatedAt:   vdev.CreatedAt,
		Version:     "1.0",
	}

	markerPath := filepath.Join(m.config.Performance.LocalCachePath, vdev.ID, "marker.json")

	if err := os.MkdirAll(filepath.Dir(markerPath), 0750); err != nil {
		return errors.Wrap(err, "failed to create VDEV cache directory")
	}

	data, err := json.Marshal(marker)
	if err != nil {
		return errors.Wrap(err, "failed to marshal marker")
	}

	if err := os.WriteFile(markerPath, data, 0640); err != nil {
		return errors.Wrap(err, "failed to write marker file")
	}

	return nil
}

func (m *Manager) cleanupVDEVStorage(ctx context.Context, vdev *VDEVInfo) error {
	vdevCachePath := filepath.Join(m.config.Performance.LocalCachePath, vdev.ID)

	if err := os.RemoveAll(vdevCachePath); err != nil {
		return errors.Wrap(err, "failed to remove VDEV cache directory")
	}

	return nil
}

type storageMarker struct {
	VDEVID      string      `json:"vdevId"`
	PoolName    string      `json:"poolName"`
	StorageType StorageType `json:"storageType"`
	CreatedAt   time.Time   `json:"createdAt"`
	Version     string      `json:"version"`
}

// BuildPoolCreateCommand returns the zpool create command arguments for an objbacker-backed pool.
func (m *Manager) BuildPoolCreateCommand(poolName string, vdevID string, opts PoolCreateOptions) []string {
	args := []string{"create"}

	if opts.MountPoint != "" {
		args = append(args, "-m", opts.MountPoint)
	}

	for k, v := range opts.Properties {
		args = append(args, "-o", fmt.Sprintf("%s=%s", k, v))
	}

	for k, v := range opts.FSProperties {
		args = append(args, "-O", fmt.Sprintf("%s=%s", k, v))
	}

	args = append(args, poolName)

	if m.config.Tiering.Enabled && opts.LocalVDEV != "" {
		args = append(args, "mirror", opts.LocalVDEV, m.GenerateVDEVPath(vdevID))
	} else {
		args = append(args, m.GenerateVDEVPath(vdevID))
	}

	if opts.CacheDevice != "" {
		args = append(args, "cache", opts.CacheDevice)
	}

	return args
}

// PoolCreateOptions contains options for creating an objbacker-backed pool.
type PoolCreateOptions struct {
	MountPoint   string
	Properties   map[string]string
	FSProperties map[string]string
	LocalVDEV    string
	CacheDevice  string
}

// DefaultPoolCreateOptions returns default options for pool creation.
func DefaultPoolCreateOptions() PoolCreateOptions {
	return PoolCreateOptions{
		Properties: map[string]string{
			"ashift":     "12",
			"autoexpand": "on",
		},
		FSProperties: map[string]string{
			"compression": "lz4",
			"atime":       "off",
			"recordsize":  "128K",
		},
	}
}

// EstimateCost estimates the monthly cost of storing data in object storage.
func (m *Manager) EstimateCost(dataSize uint64) *CostEstimate {
	var pricePerGBMonth float64
	var pricePerRequest float64

	switch m.config.StorageType {
	case StorageTypeS3:
		pricePerGBMonth = 0.023
		pricePerRequest = 0.0000004
	case StorageTypeGCS:
		pricePerGBMonth = 0.020
		pricePerRequest = 0.0000005
	case StorageTypeAzure:
		pricePerGBMonth = 0.018
		pricePerRequest = 0.0000004
	}

	gbSize := float64(dataSize) / (1024 * 1024 * 1024)
	storageCost := gbSize * pricePerGBMonth

	return &CostEstimate{
		DataSizeGB:       gbSize,
		StorageCostMonth: storageCost,
		PricePerGBMonth:  pricePerGBMonth,
		PricePerRequest:  pricePerRequest,
		ComparedToEBS:    gbSize * 0.10,
		SavingsPercent:   ((0.10 - pricePerGBMonth) / 0.10) * 100,
	}
}

// CostEstimate contains cost estimation data.
type CostEstimate struct {
	DataSizeGB       float64 `json:"dataSizeGb"`
	StorageCostMonth float64 `json:"storageCostMonth"`
	PricePerGBMonth  float64 `json:"pricePerGbMonth"`
	PricePerRequest  float64 `json:"pricePerRequest"`
	ComparedToEBS    float64 `json:"comparedToEbs"`
	SavingsPercent   float64 `json:"savingsPercent"`
}

// HealthCheck performs a health check on the objbacker system.
func (m *Manager) HealthCheck(ctx context.Context) (*HealthStatus, error) {
	status := &HealthStatus{
		Timestamp: time.Now(),
		Healthy:   true,
		Checks:    make(map[string]CheckResult),
	}

	if _, err := os.Stat(m.config.DevicePath); err != nil {
		status.Healthy = false
		status.Checks["device"] = CheckResult{
			Name:    "device",
			Passed:  false,
			Message: fmt.Sprintf("device not available: %v", err),
		}
	} else {
		status.Checks["device"] = CheckResult{
			Name:   "device",
			Passed: true,
		}
	}

	if _, err := os.Stat(m.config.Performance.LocalCachePath); err != nil {
		status.Healthy = false
		status.Checks["cache"] = CheckResult{
			Name:    "cache",
			Passed:  false,
			Message: fmt.Sprintf("cache directory not available: %v", err),
		}
	} else {
		status.Checks["cache"] = CheckResult{
			Name:   "cache",
			Passed: true,
		}
	}

	if m.daemonProc != nil {
		status.Checks["daemon"] = CheckResult{
			Name:   "daemon",
			Passed: true,
		}
	} else {
		status.Checks["daemon"] = CheckResult{
			Name:    "daemon",
			Passed:  false,
			Message: "daemon not running",
		}
		status.Healthy = false
	}

	return status, nil
}

// HealthStatus represents the health status of the objbacker system.
type HealthStatus struct {
	Timestamp time.Time              `json:"timestamp"`
	Healthy   bool                   `json:"healthy"`
	Checks    map[string]CheckResult `json:"checks"`
}

// CheckResult represents the result of a single health check.
type CheckResult struct {
	Name    string `json:"name"`
	Passed  bool   `json:"passed"`
	Message string `json:"message,omitempty"`
}

// sanitizePoolName removes invalid characters from pool names for use in object paths.
func sanitizePoolName(name string) string {
	return strings.ReplaceAll(name, "/", "_")
}
