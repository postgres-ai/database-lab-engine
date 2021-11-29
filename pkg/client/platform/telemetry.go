/*
2021 © Postgres.ai
*/

package platform

import (
	"context"
	"errors"
	"fmt"

	"gitlab.com/postgres-ai/database-lab/v2/pkg/log"
)

// TelemetryEvent defines telemetry events.
type TelemetryEvent struct {
	InstanceID string      `json:"instance_id"`
	EventType  string      `json:"event_type"`
	Payload    interface{} `json:"event_data"`
}

// SendTelemetryEvent makes an HTTP request to send a telemetry event to the Platform.
func (p *Client) SendTelemetryEvent(ctx context.Context, request TelemetryEvent) (APIResponse, error) {
	respData := APIResponse{}

	log.Dbg("Send telemetry event", request)

	if err := p.doPost(ctx, "/rpc/telemetry_event", request, &respData); err != nil {
		return respData, fmt.Errorf("failed to post request: %w", err)
	}

	if respData.Code != "" || respData.Details != "" {
		log.Dbg(fmt.Sprintf("Unsuccessful response given. Request: %v", request))

		return respData, errors.New(respData.Details)
	}

	return respData, nil
}
