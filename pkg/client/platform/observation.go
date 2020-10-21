/*
2020 © Postgres.ai
*/

package platform

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/pkg/client/dblabapi/types"
	"gitlab.com/postgres-ai/database-lab/pkg/log"
	"gitlab.com/postgres-ai/database-lab/pkg/models"
)

// StartObservationRequest represents a start observation request.
type StartObservationRequest struct {
	InstanceID string            `json:"instance_id"`
	CloneID    string            `json:"clone_id"`
	StartedAt  string            `json:"started_at"`
	Config     types.Config      `json:"config"`
	Tags       map[string]string `json:"tags"`
}

// StartObservationResponse represents response of a start observation request.
type StartObservationResponse struct {
	APIResponse
	SessionID uint64 `json:"id"`
}

// StartObservationSession makes an HTTP request to notify Platform about start observation.
func (p *Client) StartObservationSession(ctx context.Context, request StartObservationRequest) (StartObservationResponse, error) {
	respData := StartObservationResponse{}

	log.Dbg("Start Observation session", request)

	if err := p.doPost(ctx, "/rpc/dblab_session_start", request, &respData); err != nil {
		return respData, errors.Wrap(err, "failed to post request")
	}

	if respData.Code != "" || respData.Details != "" {
		log.Dbg(fmt.Sprintf("Unsuccessful response given. Request: %v", request))

		return respData, errors.New(respData.Details)
	}

	log.Dbg("Start observation response", respData)

	return respData, nil
}

// StopObservationRequest represents a stop observation request.
type StopObservationRequest struct {
	SessionID  uint64                   `json:"id"`
	FinishedAt string                   `json:"finished_at"`
	Result     models.ObservationResult `json:"result"`
}

// StopObservationSession makes an HTTP request to notify Platform about stop observation and pass observation details.
func (p *Client) StopObservationSession(ctx context.Context, request StopObservationRequest) (APIResponse, error) {
	respData := APIResponse{}

	log.Dbg("Stop Observation session", request)

	if err := p.doPost(ctx, "/rpc/dblab_session_stop", request, &respData); err != nil {
		return respData, errors.Wrap(err, "failed to post request")
	}

	if respData.Code != "" || respData.Details != "" {
		log.Dbg(fmt.Sprintf("Unsuccessful response given. Request: %v", request))

		return respData, errors.New(respData.Details)
	}

	log.Dbg("Stop Observation response", respData)

	return respData, nil
}

// UploadObservationLogs makes an HTTP request to upload observation logs to Platform.
func (p *Client) UploadObservationLogs(ctx context.Context, data []byte, headers map[string]string) error {
	var respData APIResponse

	log.Dbg(headers)

	if err := p.doUpload(ctx, "/rpc/dblab_session_upload_log", data, headers, &respData); err != nil {
		return errors.Wrap(err, "failed to upload request")
	}

	if respData.Code != "" || respData.Details != "" {
		log.Dbg(fmt.Sprintf("Unsuccessful response given. Request len: %v", len(data)))

		return errors.New("failed to upload observation logs")
	}

	log.Dbg("Upload Observation response", respData)

	return nil
}
