// Package billing manages the billing data for the instance.
package billing

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/pbnjay/memory"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/pool"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/client/platform"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/config/global"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

const errorsSoftLimit = 2

// Billing manages the billing data.
type Billing struct {
	platform  *platform.Client
	props     *global.EngineProps
	pm        *pool.Manager
	mu        *sync.Mutex
	softFails int
}

// New creates a new Billing struct.
func New(platform *platform.Client, props *global.EngineProps, pm *pool.Manager) *Billing {
	return &Billing{platform: platform, props: props, pm: pm, mu: &sync.Mutex{}}
}

// Reload updates platform client.
func (b *Billing) Reload(platformSvc *platform.Client) {
	b.platform = platformSvc
}

func (b *Billing) increaseFailureCounter() int {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.softFails++

	return b.softFails
}

func (b *Billing) softLimitCounter() int {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.softFails
}

func (b *Billing) isSoftLimitExceeded() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.softFails > errorsSoftLimit
}

func (b *Billing) resetSoftFailureCounter() {
	b.mu.Lock()

	b.softFails = 0

	b.mu.Unlock()
}

// RegisterInstance registers instance on the Platform.
func (b *Billing) RegisterInstance(ctx context.Context, systemMetrics models.System) error {
	if b.props.IsAWS() {
		// Because billing goes through AWS Marketplace.
		b.props.UpdateBilling(true)
	}

	if err := b.shouldSendPlatformRequests(); err != nil {
		return err
	}

	if err := b.platform.RegisterInstance(ctx, b.props); err != nil {
		return fmt.Errorf("cannot register instance: %w", err)
	}

	// To check billing state immediately.
	if err := b.SendUsage(ctx, systemMetrics); err != nil {
		return err
	}

	return nil
}

// CollectUsage periodically collects usage statistics of the instance.
func (b *Billing) CollectUsage(ctx context.Context, system models.System) {
	if err := b.shouldSendPlatformRequests(); err != nil {
		log.Msg("Skip collecting usage:", err.Error())
		return
	}

	ticker := time.NewTicker(time.Hour)

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			poolStat := b.pm.CollectPoolStat()

			if err := b.SendUsage(ctx, models.System{
				CPU:         system.CPU,
				TotalMemory: system.TotalMemory,
				DataUsed:    poolStat.TotalUsed,
			}); err != nil {
				log.Err("collecting usage:", err)
			}
		}
	}
}

// SendUsage sends usage events.
func (b *Billing) SendUsage(ctx context.Context, systemMetrics models.System) error {
	respData, err := b.platform.SendUsage(ctx, b.props, platform.InstanceUsage{
		InstanceID: b.props.InstanceID,
		EventData: platform.DataUsage{
			CPU:         systemMetrics.CPU,
			TotalMemory: systemMetrics.TotalMemory,
			DataSize:    systemMetrics.DataUsed,
		}})

	if err != nil {
		b.increaseFailureCounter()

		if b.isSoftLimitExceeded() {
			log.Msg("Billing error threshold surpassed. Certain features have been temporarily disabled.")
			b.props.UpdateBilling(false)
		}

		return fmt.Errorf("cannot send usage event: %w. Attempts: %d", err, b.softLimitCounter())
	}

	if b.props.BillingActive != respData.BillingActive {
		b.props.UpdateBilling(respData.BillingActive)

		log.Dbg("Instance state updated. Billing is active:", respData.BillingActive)
	}

	if b.props.BillingActive {
		b.resetSoftFailureCounter()
	}

	return nil
}

func (b *Billing) shouldSendPlatformRequests() error {
	if b.props.IsAWS() {
		return errors.New("DLE infrastructure is AWS Marketplace")
	}

	if b.props.GetEdition() != global.StandardEdition {
		return errors.New("DLE edition is not Standard")
	}

	if !b.platform.HasOrgKey() {
		return errors.New("organization key is empty")
	}

	return nil
}

// GetSystemMetrics collects system metrics significant for billing purposes.
func GetSystemMetrics(pm *pool.Manager) models.System {
	poolStat := pm.CollectPoolStat()

	systemMetrics := models.System{
		CPU:         runtime.NumCPU(),
		TotalMemory: memory.TotalMemory(),
		DataUsed:    poolStat.TotalUsed,
	}

	return systemMetrics
}
