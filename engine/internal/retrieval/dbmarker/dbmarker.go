/*
2020 © Postgres.ai
*/

// Package dbmarker provides a tool for marking database data.
package dbmarker

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

const (
	configDir      = ".dblab"
	configFilename = "dbmarker"

	refsDir      = "refs"
	branchesDir  = "branch"
	snapshotsDir = "snapshot"
	headFile     = "HEAD"
	logsFile     = "logs"
	mainBranch   = "main"

	// LogicalDataType defines a logical data type.
	LogicalDataType = "logical"

	// PhysicalDataType defines a physical data type.
	PhysicalDataType = "physical"
)

// Marker marks database data depends on a retrieval process.
type Marker struct {
	dataPath string
}

// NewMarker creates a new DBMarker.
func NewMarker(dataPath string) *Marker {
	return &Marker{
		dataPath: dataPath,
	}
}

// Config describes marked data.
type Config struct {
	DataStateAt string `yaml:"dataStateAt"`
	DataType    string `yaml:"dataType"`
}

// Head describes content of HEAD file.
type Head struct {
	Ref string `yaml:"ref"`
}

// SnapshotInfo describes snapshot info.
type SnapshotInfo struct {
	ID        string
	Parent    string
	CreatedAt string
	StateAt   string
}

// Init inits DB marker for the data directory.
func (m *Marker) initDBLabDirectory() error {
	dirname := path.Join(m.dataPath, configDir)
	if err := os.MkdirAll(dirname, 0755); err != nil {
		return errors.Wrapf(err, "cannot create a DBMarker directory %s", dirname)
	}

	return nil
}

// CreateConfig creates a new DBMarker config file.
func (m *Marker) CreateConfig() error {
	if err := m.initDBLabDirectory(); err != nil {
		return errors.Wrap(err, "failed to init DBMarker")
	}

	dbMarkerFile, err := os.OpenFile(m.buildFileName(configFilename), os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return err
	}

	defer func() { _ = dbMarkerFile.Close() }()

	return nil
}

// GetConfig provides a loaded DBMarker config.
func (m *Marker) GetConfig() (*Config, error) {
	configData, err := os.ReadFile(m.buildFileName(configFilename))
	if err != nil {
		return nil, err
	}

	cfg := &Config{}

	if len(configData) == 0 {
		return cfg, nil
	}

	if err := yaml.Unmarshal(configData, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// SaveConfig stores a DBMarker config.
func (m *Marker) SaveConfig(cfg *Config) error {
	configData, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(m.buildFileName(configFilename), configData, 0600)
}

// buildFileName builds a DBMarker filename.
func (m *Marker) buildFileName(filename string) string {
	return path.Join(m.dataPath, configDir, filename)
}

// InitBranching creates structures for data branching.
func (m *Marker) InitBranching() error {
	branchesDir := m.buildBranchesPath()
	if err := os.MkdirAll(branchesDir, 0755); err != nil {
		return fmt.Errorf("cannot create branches directory %s: %w", branchesDir, err)
	}

	snapshotsDir := m.buildSnapshotsPath()
	if err := os.MkdirAll(snapshotsDir, 0755); err != nil {
		return fmt.Errorf("cannot create snapshots directory %s: %w", snapshotsDir, err)
	}

	f, err := os.Create(m.buildFileName(headFile))
	if err != nil {
		return fmt.Errorf("cannot create HEAD file: %w", err)
	}

	_ = f.Close()

	return nil
}

// InitMainBranch creates a new main branch.
func (m *Marker) InitMainBranch(infos []SnapshotInfo) error {
	var head Head

	mainDir := m.buildBranchName(mainBranch)
	if err := os.MkdirAll(mainDir, 0755); err != nil {
		return fmt.Errorf("cannot create branches directory %s: %w", mainDir, err)
	}

	var bb bytes.Buffer

	for _, info := range infos {
		if err := m.storeSnapshotInfo(info); err != nil {
			return err
		}

		head.Ref = buildSnapshotRef(info.ID)
		log := strings.Join([]string{info.Parent, info.ID, info.CreatedAt, info.StateAt}, " ") + "\n"
		bb.WriteString(log)
	}

	if err := os.WriteFile(m.buildBranchArtifactPath(mainBranch, logsFile), bb.Bytes(), 0755); err != nil {
		return fmt.Errorf("cannot store file with HEAD metadata: %w", err)
	}

	headData, err := yaml.Marshal(head)
	if err != nil {
		return fmt.Errorf("cannot prepare HEAD metadata: %w", err)
	}

	if err := os.WriteFile(m.buildFileName(headFile), headData, 0755); err != nil {
		return fmt.Errorf("cannot store file with HEAD metadata: %w", err)
	}

	if err := os.WriteFile(m.buildBranchArtifactPath(mainBranch, headFile), headData, 0755); err != nil {
		return fmt.Errorf("cannot store file with HEAD metadata: %w", err)
	}

	return nil
}

func (m *Marker) storeSnapshotInfo(info SnapshotInfo) error {
	snapshotName := m.buildSnapshotName(info.ID)

	data, err := yaml.Marshal(info)
	if err != nil {
		return fmt.Errorf("cannot prepare snapshot metadata %s: %w", snapshotName, err)
	}

	if err := os.WriteFile(snapshotName, data, 0755); err != nil {
		return fmt.Errorf("cannot store file with snapshot metadata %s: %w", snapshotName, err)
	}

	return nil
}

// CreateBranch creates a new DLE data branch.
func (m *Marker) CreateBranch(branch, base string) error {
	dirname := m.buildBranchName(branch)
	if err := os.MkdirAll(dirname, 0755); err != nil {
		return fmt.Errorf("cannot create branches directory %s: %w", dirname, err)
	}

	headPath := m.buildBranchArtifactPath(base, headFile)

	readData, err := os.ReadFile(headPath)
	if err != nil {
		return fmt.Errorf("cannot read file %s: %w", headPath, err)
	}

	branchPath := m.buildBranchArtifactPath(branch, headFile)

	if err := os.WriteFile(branchPath, readData, 0755); err != nil {
		return fmt.Errorf("cannot write file %s: %w", branchPath, err)
	}

	return nil
}

// ListBranches returns branch list.
func (m *Marker) ListBranches() ([]string, error) {
	branches := []string{}

	dirs, err := os.ReadDir(m.buildBranchesPath())
	if err != nil {
		return nil, fmt.Errorf("failed to read repository: %w", err)
	}

	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}

		branches = append(branches, dir.Name())
	}

	return branches, nil
}

// GetSnapshotID returns snapshot pointer for branch.
func (m *Marker) GetSnapshotID(branch string) (string, error) {
	headPath := m.buildBranchArtifactPath(branch, headFile)

	readData, err := os.ReadFile(headPath)
	if err != nil {
		return "", fmt.Errorf("cannot read file %s: %w", headPath, err)
	}

	h := &Head{}
	if err := yaml.Unmarshal(readData, &h); err != nil {
		return "", fmt.Errorf("cannot read reference: %w", err)
	}

	snapshotsPath := m.buildPathFromRef(h.Ref)

	snapshotData, err := os.ReadFile(snapshotsPath)
	if err != nil {
		return "", fmt.Errorf("cannot read file %s: %w", snapshotsPath, err)
	}

	snInfo := &SnapshotInfo{}

	if err := yaml.Unmarshal(snapshotData, &snInfo); err != nil {
		return "", fmt.Errorf("cannot read reference: %w", err)
	}

	return snInfo.ID, nil
}

// SaveSnapshotRef stores snapshot reference for branch.
func (m *Marker) SaveSnapshotRef(branch, snapshotID string) error {
	h, err := m.getBranchHead(branch)
	if err != nil {
		return err
	}

	h.Ref = buildSnapshotRef(snapshotID)

	if err := m.writeBranchHead(h, branch); err != nil {
		return "", fmt.Errorf("cannot write branch head: %w", err)
	}

	return nil
}

func (m *Marker) getBranchHead(branch string) (*Head, error) {
	headPath := m.buildBranchArtifactPath(branch, headFile)

	readData, err := os.ReadFile(headPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read file %s: %w", headPath, err)
	}

	h := &Head{}
	if err := yaml.Unmarshal(readData, &h); err != nil {
		return nil, fmt.Errorf("cannot read reference: %w", err)
	}

	return h, nil
}

func (m *Marker) writeBranchHead(h *Head, branch string) error {
	headPath := m.buildBranchArtifactPath(branch, headFile)

	writeData, err := yaml.Marshal(h)
	if err != nil {
		return fmt.Errorf("cannot marshal structure: %w", err)
	}

	if err := os.WriteFile(headPath, writeData, 0755); err != nil {
		return fmt.Errorf("cannot write file %s: %w", headPath, err)
	}

	return nil
}

// buildBranchesPath builds path of branches dir.
func (m *Marker) buildBranchesPath() string {
	return path.Join(m.dataPath, configDir, refsDir, branchesDir)
}

// buildBranchName builds a branch name.
func (m *Marker) buildBranchName(branch string) string {
	return path.Join(m.buildBranchesPath(), branch)
}

// buildBranchArtifactPath builds a branch artifact name.
func (m *Marker) buildBranchArtifactPath(branch, artifact string) string {
	return path.Join(m.buildBranchName(branch), artifact)
}

// buildSnapshotsPath builds path of snapshots dir.
func (m *Marker) buildSnapshotsPath() string {
	return path.Join(m.dataPath, configDir, refsDir, snapshotsDir)
}

// buildSnapshotName builds a snapshot file name.
func (m *Marker) buildSnapshotName(snapshotID string) string {
	return path.Join(m.buildSnapshotsPath(), snapshotID)
}

// buildSnapshotRef builds snapshot ref.
func buildSnapshotRef(snapshotID string) string {
	return path.Join(refsDir, snapshotsDir, snapshotID)
}

// buildPathFromRef builds path from ref.
func (m *Marker) buildPathFromRef(ref string) string {
	return path.Join(m.dataPath, configDir, ref)
}
