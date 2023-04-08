// Package billing manages the billing data for the instance.
package billing

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"time"

	"github.com/pbnjay/memory"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/pool"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/client/platform"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/config/global"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

// Billing manages the billing data.
type Billing struct {
	platform *platform.Client
	props    *global.EngineProps
	pm       *pool.Manager
}

// New creates a new Billing struct.
func New(platform *platform.Client, props *global.EngineProps, pm *pool.Manager) *Billing {
	return &Billing{platform: platform, props: props, pm: pm}
}

// RegisterInstance registers instance on the Platform.
func (b *Billing) RegisterInstance(ctx context.Context, systemMetrics models.System) error {
	if b.props.Infrastructure == global.AWSInfrastructure {
		// Because billing goes through AWS Marketplace.
		b.props.UpdateBilling(true)
	}

	if err := b.shouldSendPlatformRequests(); err != nil {
		return err
	}

	if err := b.platform.RegisterInstance(ctx, b.props); err != nil {
		return fmt.Errorf("cannot register instance: %w", err)
	}

	if _, err := b.platform.SendUsage(ctx, b.props, platform.InstanceUsage{
		InstanceID: b.props.InstanceID,
		EventData: platform.DataUsage{
			CPU:         systemMetrics.CPU,
			TotalMemory: systemMetrics.TotalMemory,
			DataSize:    systemMetrics.DataUsed,
		}}); err != nil {
		return fmt.Errorf("cannot send the initial usage event: %w", err)
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

			if _, err := b.platform.SendUsage(ctx, b.props, platform.InstanceUsage{
				InstanceID: b.props.InstanceID,
				EventData: platform.DataUsage{
					CPU:         system.CPU,
					TotalMemory: system.TotalMemory,
					DataSize:    poolStat.TotalUsed,
				},
			}); err != nil {
				log.Err("failed to send usage event:", err)
			}
		}
	}
}

func (b *Billing) shouldSendPlatformRequests() error {
	if b.props.Infrastructure == global.AWSInfrastructure {
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
