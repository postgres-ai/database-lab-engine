/*
2020 © Postgres.ai
*/

// Package observer provides clone monitoring.
package observer

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"time"

	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/pkg/log"
	"gitlab.com/postgres-ai/database-lab/pkg/models"
	"gitlab.com/postgres-ai/database-lab/pkg/util"
)

const (
	defaultIntervalSeconds        = 10
	defaultMaxLockDurationSeconds = 10
	defaultMaxDurationSeconds     = 60 * 60 // 1 hour.

	stateFilePath = "/tmp/dblab-observe-state.json"
)

// Config defines configuration options for observer.
type Config struct {
	Follow                 bool   `json:"follow"`
	IntervalSeconds        uint64 `json:"intervalSeconds"`
	MaxLockDurationSeconds uint64 `json:"maxLockDurationSeconds"`
	MaxDurationSeconds     uint64 `json:"maxDurationSeconds"`
	SSLMode                string `json:"sslmode"`
}

// Observer defines monitoring service.
type Observer struct {
	StartedAt      time.Time     `json:"startedAt"`
	Elapsed        time.Duration `json:"elapsed"`
	CounterTotal   uint64        `json:"counterTotal"`
	CounterWarning uint64        `json:"counterWarning"`
	CounterSuccess uint64        `json:"counterSuccess"`
	Config         Config        `json:"config"`

	writer io.Writer
}

// NewObserver creates Observer instance.
func NewObserver(config Config, writer io.Writer) *Observer {
	if config.IntervalSeconds == 0 {
		config.IntervalSeconds = defaultIntervalSeconds
	}

	if config.MaxLockDurationSeconds == 0 {
		config.MaxLockDurationSeconds = defaultMaxLockDurationSeconds
	}

	if config.MaxDurationSeconds == 0 {
		config.MaxDurationSeconds = defaultMaxDurationSeconds
	}

	return &Observer{
		Config: config,
		writer: writer,
	}
}

// Start runs clone monitoring.
func (obs *Observer) Start(clone *models.Clone) error {
	log.Dbg("Start observing...")

	db, err := initConnection(clone, obs.Config.SSLMode)
	if err != nil {
		return errors.Wrap(err, "cannot connect to database")
	}

	obs.StartedAt = time.Now()

	for {
		now := time.Now()
		obs.Elapsed = time.Since(obs.StartedAt)

		var output strings.Builder

		output.WriteString(fmt.Sprintf("[%s] Database Lab Observer:\n", util.FormatTime(now)))
		output.WriteString(fmt.Sprintf("  Elapsed: %s\n", util.DurationToString(obs.Elapsed)))
		output.WriteString("  Dangerous locks:\n")

		dangerousLocks, err := runQuery(db, buildLocksMetricQuery(obs.Config.MaxLockDurationSeconds))
		if err != nil {
			return errors.Wrap(err, "cannot query metrics")
		}

		obs.CounterTotal++

		if len(dangerousLocks) > 0 {
			obs.CounterWarning++
		} else {
			dangerousLocks = "    Not observed\n"
			obs.CounterSuccess++
		}

		output.WriteString(dangerousLocks)

		output.WriteString("  Observed intervals:\n")
		output.WriteString(fmt.Sprintf("    Successful: %d\n", obs.CounterSuccess))
		output.WriteString(fmt.Sprintf("    With dangerous locks: %d\n", obs.CounterWarning))

		_, err = fmt.Fprintln(obs.writer, output.String())
		if err != nil {
			return errors.Wrap(err, "cannot print")
		}

		err = obs.SaveObserverState()
		if err != nil {
			return errors.Wrap(err, "cannot save observer state")
		}

		if !obs.Config.Follow {
			break
		}

		time.Sleep(time.Duration(obs.Config.IntervalSeconds) * time.Second)
	}

	return nil
}

// SaveObserverState saves observer state to the disk.
func (obs *Observer) SaveObserverState() error {
	bytes, err := json.MarshalIndent(obs, "", "  ")
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(stateFilePath, bytes, 0644)
	if err != nil {
		return err
	}

	return nil
}

// LoadObserverState loads observer state from the disk.
func (obs *Observer) LoadObserverState() error {
	bytes, err := ioutil.ReadFile(stateFilePath)
	if err != nil {
		return err
	}

	err = json.Unmarshal(bytes, &obs)
	if err != nil {
		return err
	}

	return nil
}

// PrintSummary prints monitoring summary.
func (obs *Observer) PrintSummary() error {
	maxDuration := time.Duration(obs.Config.MaxDurationSeconds) * time.Second

	var summary strings.Builder

	summary.WriteString("Summary:\n")
	summary.WriteString(formatSummaryItem(fmt.Sprintf("Duration: %s", util.DurationToString(obs.Elapsed))))
	summary.WriteString(formatSummaryItem(fmt.Sprintf("Intervals with dangerous locks: %d", obs.CounterWarning)))
	summary.WriteString(formatSummaryItem(fmt.Sprintf("Total number of observed intervals: %d", obs.CounterTotal)))
	summary.WriteString("\nPerformance checklist:\n")
	summary.WriteString(formatChecklistItem(fmt.Sprintf("Duration < %s", util.DurationToString(maxDuration)), obs.CheckDuration()))
	summary.WriteString(formatChecklistItem("No dangerous locks", obs.CheckLocks()))

	_, err := fmt.Fprint(obs.writer, summary.String())
	if err != nil {
		return errors.Wrap(err, "cannot print")
	}

	return nil
}

// CheckPerformanceRequirements checks monitoring data and returns an error if any of performance requires was not satisfied.
func (obs *Observer) CheckPerformanceRequirements() error {
	if obs.CheckDuration() || obs.CheckLocks() {
		return errors.New("performance requirements not satisfied")
	}

	return nil
}

// CheckDuration checks duration of the operation.
func (obs *Observer) CheckDuration() bool {
	return obs.Elapsed < time.Duration(obs.Config.MaxDurationSeconds)*time.Second
}

// CheckLocks checks long-lasting locks during the operation.
func (obs *Observer) CheckLocks() bool {
	return obs.CounterWarning == 0
}

func formatSummaryItem(str string) string {
	return "  " + str + "\n"
}

func formatChecklistItem(str string, state bool) string {
	stateStr := colorizeRed("FAILED")

	if state {
		stateStr = colorizeGreen("PASSED")
	}

	return "  " + str + ": " + stateStr + "\n"
}

func colorizeRed(str string) string {
	return fmt.Sprintf("\033[1;31m%s\033[0m", str)
}

func colorizeGreen(str string) string {
	return fmt.Sprintf("\033[1;32m%s\033[0m", str)
}
