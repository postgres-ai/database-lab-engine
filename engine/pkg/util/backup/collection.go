package backup

import (
	"fmt"
	iofs "io/fs"
	"os"
	"path/filepath"
	"sort"

	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools/fs"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util"
)

// Collection represents a collection of backups.
type Collection struct {
	Filename string
	backups  []*backup
	perm     iofs.FileMode
}

// NewBackupCollection finds a collection of backups.
func NewBackupCollection(filename string) (*Collection, error) {
	filename = filepath.Clean(filename)

	stat, err := os.Stat(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	if stat.IsDir() {
		return nil, fmt.Errorf("file is a directory")
	}

	files, err := filepath.Glob(filename + "*")
	if err != nil {
		return nil, fmt.Errorf("failed to glob file: %w", err)
	}

	backups := make([]*backup, 0, len(files)-1)

	for _, filePath := range files {
		if filepath.Ext(filePath) != backupFileExtension {
			continue
		}

		timestamp, err := getFileTimestamp(filePath)
		if err != nil {
			continue
		}

		backup := &backup{
			Filename: filePath,
			Time:     timestamp,
		}
		backups = append(backups, backup)
	}

	c := &Collection{
		Filename: filename,
		backups:  backups,
		perm:     stat.Mode(),
	}

	c.sort()

	return c, nil
}

// Rotate rotates the backups.
func (c *Collection) Rotate(content []byte) error {
	err := c.Backup()
	if err != nil {
		return fmt.Errorf("failed to backup: %w", err)
	}

	err = os.WriteFile(c.Filename, content, c.perm)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// Backup create backup of a file.
func (c *Collection) Backup() error {
	nowTime := now()

	last := &backup{
		Filename: c.Filename + "." +
			nowTime.Format(util.DataStateAtFormat) +
			backupFileExtension,
		Time: nowTime,
	}

	err := fs.CopyFile(c.Filename, last.Filename)
	if err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	c.backups = append(c.backups, last)
	c.sort()

	return nil
}

// EnsureMaxBackups ensures that there are no more than maxBackups backups.
func (c *Collection) EnsureMaxBackups(count int) error {
	if count < 0 {
		return fmt.Errorf("count must be positive")
	}

	c.sort()
	backupsCount := len(c.backups)
	removeCount := backupsCount - count

	if removeCount <= 0 {
		return nil
	}

	for i := 0; i < removeCount; i++ {
		err := os.Remove(c.backups[i].Filename)
		if err != nil {
			return fmt.Errorf("failed to remove file: %w", err)
		}
	}

	c.backups = make([]*backup, count)
	copy(c.backups, c.backups[removeCount:])

	return nil
}

func (c *Collection) sort() {
	sort.Slice(c.backups, func(i, j int) bool {
		return c.backups[i].Time.Before(c.backups[j].Time)
	})
}
