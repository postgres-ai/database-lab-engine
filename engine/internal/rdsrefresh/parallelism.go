/*
2026 © PostgresAI
*/

package rdsrefresh

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
)

const (
	// rdsInstanceClassPrefix is stripped to derive the instance size.
	rdsInstanceClassPrefix = "db."

	// minParallelJobs is the minimum parallelism level.
	minParallelJobs = 1
)

// instanceSizeVCPUs maps AWS instance size suffixes to their typical vCPU count.
// this mapping is consistent across most instance families (m5, m6g, r5, r6g, c5, etc.).
// graviton and intel/amd variants of the same size have the same vCPU count.
var instanceSizeVCPUs = map[string]int{
	"micro":    1,
	"small":    1,
	"medium":   2,
	"large":    2,
	"xlarge":   4,
	"2xlarge":  8,
	"3xlarge":  12,
	"4xlarge":  16,
	"6xlarge":  24,
	"8xlarge":  32,
	"9xlarge":  36,
	"10xlarge": 40,
	"12xlarge": 48,
	"16xlarge": 64,
	"18xlarge": 72,
	"24xlarge": 96,
	"32xlarge": 128,
	"48xlarge": 192,
	"metal":    96,
}

// ParallelismConfig holds the computed parallelism levels for dump and restore.
type ParallelismConfig struct {
	DumpJobs    int
	RestoreJobs int
}

// ResolveParallelism determines the optimal parallelism levels for pg_dump and pg_restore.
// dump parallelism is based on the vCPU count of the RDS clone instance class.
// restore parallelism is based on the vCPU count of the local machine.
// local vCPU detection uses runtime.NumCPU(), which works on Linux
// (the target platform for DBLab Engine).
func ResolveParallelism(cfg *Config) (*ParallelismConfig, error) {
	dumpJobs, err := resolveRDSInstanceVCPUs(cfg.RDSClone.InstanceClass)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve RDS instance vCPUs: %w", err)
	}

	restoreJobs := resolveLocalVCPUs()

	log.Msg("auto-parallelism: dump jobs =", dumpJobs, "(RDS clone vCPUs), restore jobs =", restoreJobs, "(local vCPUs)")

	return &ParallelismConfig{
		DumpJobs:    dumpJobs,
		RestoreJobs: restoreJobs,
	}, nil
}

// resolveRDSInstanceVCPUs estimates the vCPU count for the given RDS instance class
// by parsing the instance size suffix (e.g. "xlarge" from "db.m5.xlarge").
// the mapping covers standard AWS size naming used across RDS instance families.
// if the size is not recognized, it attempts to parse a numeric multiplier prefix
// (e.g. "2xlarge" → 8 vCPUs).
func resolveRDSInstanceVCPUs(instanceClass string) (int, error) {
	size, err := extractInstanceSize(instanceClass)
	if err != nil {
		return 0, err
	}

	if vcpus, ok := instanceSizeVCPUs[size]; ok {
		return vcpus, nil
	}

	// handle unlisted NUMxlarge sizes by parsing the multiplier
	vcpus, err := parseXlargeMultiplier(size)
	if err != nil {
		return 0, fmt.Errorf("unknown instance size %q in class %q", size, instanceClass)
	}

	return vcpus, nil
}

// extractInstanceSize extracts the size component from an RDS instance class.
// for example, "db.m5.xlarge" → "xlarge", "db.r6g.2xlarge" → "2xlarge".
func extractInstanceSize(instanceClass string) (string, error) {
	if !strings.HasPrefix(instanceClass, rdsInstanceClassPrefix) {
		return "", fmt.Errorf("invalid RDS instance class %q: expected %q prefix", instanceClass, rdsInstanceClassPrefix)
	}

	withoutPrefix := strings.TrimPrefix(instanceClass, rdsInstanceClassPrefix)

	// format is "family.size", e.g. "m5.xlarge" or "r6g.2xlarge"
	parts := strings.SplitN(withoutPrefix, ".", 2)

	const expectedParts = 2
	if len(parts) != expectedParts || parts[1] == "" {
		return "", fmt.Errorf("invalid RDS instance class %q: expected format db.<family>.<size>", instanceClass)
	}

	return parts[1], nil
}

// parseXlargeMultiplier handles NUMxlarge patterns not in the static map.
// for example, "5xlarge" → 5 * 4 = 20 vCPUs.
func parseXlargeMultiplier(size string) (int, error) {
	idx := strings.Index(size, "xlarge")
	if idx <= 0 {
		return 0, fmt.Errorf("not an xlarge variant: %q", size)
	}

	multiplier, err := strconv.Atoi(size[:idx])
	if err != nil {
		return 0, fmt.Errorf("invalid multiplier in %q: %w", size, err)
	}

	const vcpusPerXlarge = 4

	return multiplier * vcpusPerXlarge, nil
}

// resolveLocalVCPUs returns the number of logical CPUs available on the local machine.
// uses runtime.NumCPU() which reads from /proc/cpuinfo on Linux
// (the target platform for DBLab Engine).
func resolveLocalVCPUs() int {
	cpus := runtime.NumCPU()
	if cpus < minParallelJobs {
		return minParallelJobs
	}

	return cpus
}
