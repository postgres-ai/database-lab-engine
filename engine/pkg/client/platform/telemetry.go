/*
2021 © Postgres.ai
*/

package platform

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"path"
	"time"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/config/global"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
)

// TelemetryEvent defines telemetry events.
type TelemetryEvent struct {
	InstanceID string      `json:"instance_id"`
	EventType  string      `json:"event_type"`
	Payload    interface{} `json:"event_data"`
}

// RegisterInstanceRequest defines request to register instance on Platform.
type RegisterInstanceRequest struct {
	InstanceID  string `json:"selfassigned_instance_id"`
	ProjectName string `json:"project_name"`
}

// InstanceUsage defines details of the instance and its usage stats.
type InstanceUsage struct {
	InstanceID string    `json:"selfassigned_instance_id"`
	EventData  DataUsage `json:"event_data"`
}

// DataUsage defines event usage data.
type DataUsage struct {
	CPU         int    `json:"cpu"`
	TotalMemory uint64 `json:"total_memory"`
	DataSize    uint64 `json:"data_size"`
}

// SendTelemetryEvent makes an HTTP request to send a telemetry event to the Platform.
func (p *Client) SendTelemetryEvent(ctx context.Context, request TelemetryEvent) (APIResponse, error) {
	respData := APIResponse{}

	log.Dbg("Send telemetry event", request)

	if !p.isURLDefined() {
		return APIResponse{}, fmt.Errorf("platform URL is not defined")
	}

	if err := p.doPost(ctx, "/rpc/telemetry_event", request, &respData); err != nil {
		return respData, fmt.Errorf("failed to post request: %w", err)
	}

	if respData.Code != "" || respData.Details != "" {
		log.Dbg(fmt.Sprintf("Unsuccessful response given. Request: %v", request))

		return respData, errors.New(respData.Details)
	}

	return respData, nil
}

// RegisterResponse contains information about the DLE registration.
type RegisterResponse struct {
	APIResponse
	ID         int        `json:"id"`
	ProjectID  int        `json:"project_id"`
	InstanceID string     `json:"selfassigned_instance_id"`
	Plan       string     `json:"plan"`
	CreatedAt  *time.Time `json:"created_at,omitempty"`
}

// EditionResponse contains information about the DLE edition.
type EditionResponse struct {
	APIResponse
	BillingResponse
}

// BillingResponse contains billing status.
type BillingResponse struct {
	Result        string `json:"result"`
	BillingActive bool   `json:"billing_active"`
	Org           Org    `json:"recognized_org"`
}

// Org contains organization details.
type Org struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Alias       string `json:"alias"`
	BillingPage string `json:"billing_page,omitempty"`
	//nolint:misspell
	PrivilegedUntil time.Time `json:"priveleged_until"`
}

// RegisterInstance registers the instance.
func (p *Client) RegisterInstance(ctx context.Context, props *global.EngineProps) error {
	if !p.HasOrgKey() {
		return errors.New("no organization key provided to register instance")
	}

	request := RegisterInstanceRequest{
		InstanceID:  props.InstanceID,
		ProjectName: p.projectName,
	}

	var respData RegisterResponse

	log.Dbg("Register instance", request)

	if err := p.doPost(ctx, "/rpc/dblab_instance_register", request, &respData); err != nil {
		return fmt.Errorf("failed to post request: %w. Response: %v", err, respData)
	}

	log.Dbg("Instance has been registered:", respData)

	return nil
}

const (
	consolePath = "console"
	billingPath = "billing"
)

// SendUsage sends usage statistics of the instance.
func (p *Client) SendUsage(ctx context.Context, props *global.EngineProps, usage InstanceUsage) (*EditionResponse, error) {
	if !p.HasOrgKey() {
		return nil, errors.New("no organization key provided")
	}

	var respData EditionResponse

	log.Dbg("Send usage event", usage)

	if err := p.doPost(ctx, "/rpc/telemetry_usage", usage, &respData); err != nil {
		return nil, fmt.Errorf("failed to post telemetry request: %w", err)
	}

	if props.BillingActive != respData.BillingActive {
		props.UpdateBilling(respData.BillingActive)

		log.Dbg("Instance state updated. Billing is active:", respData.BillingActive)
	}

	respData.Org.BillingPage = (&url.URL{
		Scheme: p.url.Scheme,
		Host:   p.url.Host,
		Path:   path.Join(consolePath, respData.Org.Alias, billingPath),
	}).String()

	log.Dbg("Usage event response", respData)

	return &respData, nil
}

func (p *Client) isURLDefined() bool {
	return p.url != nil
}

// HasOrgKey checks if orgKey is set.
func (p *Client) HasOrgKey() bool {
	return p.orgKey != ""
}
