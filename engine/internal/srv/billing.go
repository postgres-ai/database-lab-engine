package srv

import (
	"context"
	"fmt"
	"net/http"

	"github.com/AlekSi/pointer"

	"gitlab.com/postgres-ai/database-lab/v3/internal/billing"
	"gitlab.com/postgres-ai/database-lab/v3/internal/srv/api"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/client/platform"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
	"gitlab.com/postgres-ai/database-lab/v3/version"
)

func (s *Server) billingStatus(w http.ResponseWriter, r *http.Request) {
	usageResponse, err := s.billingUsage(r.Context())
	if err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	if usageResponse.Code != "" {
		api.SendBadRequestError(w, r, fmt.Sprintf("Error code %s, message: %s", usageResponse.Code, usageResponse.Message))
		return
	}

	if err := api.WriteJSON(w, http.StatusOK, usageResponse.BillingResponse); err != nil {
		api.SendError(w, r, err)
		return
	}
}

func (s *Server) activate(w http.ResponseWriter, r *http.Request) {
	if _, err := s.billingUsage(r.Context()); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	engine := models.Engine{
		Version:                   version.GetVersion(),
		Edition:                   s.engProps.GetEdition(),
		BillingActive:             pointer.ToBool(s.engProps.BillingActive),
		InstanceID:                s.engProps.InstanceID,
		StartedAt:                 s.startedAt,
		Telemetry:                 pointer.ToBool(s.Platform.IsTelemetryEnabled()),
		DisableConfigModification: pointer.ToBool(s.Config.DisableConfigModification),
	}

	if err := api.WriteJSON(w, http.StatusOK, engine); err != nil {
		api.SendError(w, r, err)
		return
	}
}

func (s *Server) billingUsage(ctx context.Context) (*platform.EditionResponse, error) {
	systemMetrics := billing.GetSystemMetrics(s.pm)

	instanceUsage := platform.InstanceUsage{
		InstanceID: s.engProps.InstanceID,
		EventData: platform.DataUsage{
			CPU:         systemMetrics.CPU,
			TotalMemory: systemMetrics.TotalMemory,
			DataSize:    systemMetrics.DataUsed,
		},
	}

	return s.Platform.Client.SendUsage(ctx, s.engProps, instanceUsage)
}
