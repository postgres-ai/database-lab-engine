/*
2021 © Postgres.ai
*/

// Package config contains configuration options of HTTP server.
package config

// Config provides configuration management via DLE API
type Config struct {
	VerificationToken         string `yaml:"verificationToken" json:"-"`
	Host                      string `yaml:"host"`
	Port                      uint   `yaml:"port"`
	DisableConfigModification bool   `yaml:"disableConfigModification" json:"-"`
}

// Retention configures background auto-deletion of unused branches and snapshots.
// All fields are optional; a zero value disables the corresponding behavior, so an absent
// retention section leaves auto-deletion off.
type Retention struct {
	// UnusedSnapshotMinutes auto-deletes an unused snapshot after this many minutes; 0 disables it.
	UnusedSnapshotMinutes uint `yaml:"unusedSnapshotMinutes"`
	// UnusedBranchMinutes auto-deletes an unused branch after this many minutes; 0 disables it.
	UnusedBranchMinutes uint `yaml:"unusedBranchMinutes"`
	// CheckIntervalMinutes is the sweep cadence; 0 falls back to the default interval.
	CheckIntervalMinutes uint `yaml:"checkIntervalMinutes"`
	// ProtectionMaxDurationMinutes caps timed protection of branches/snapshots; 0 means no cap.
	ProtectionMaxDurationMinutes uint `yaml:"protectionMaxDurationMinutes"`
	// MaxDeletionsPerTick bounds how many entities the sweeper deletes per cycle.
	MaxDeletionsPerTick uint `yaml:"maxDeletionsPerTick"`
}
