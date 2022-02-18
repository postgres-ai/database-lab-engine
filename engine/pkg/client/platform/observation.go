/*
2020 © Postgres.ai
*/

package platform

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/v3/pkg/client/dblabapi/types"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
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

// UploadArtifactResponse represents response of a uploading artifact request.
type UploadArtifactResponse struct {
	DBLabSessionID int       `json:"dblab_session_id"`
	ArtifactTime   time.Time `json:"artifact_time"`
	ArtifactType   string    `json:"artifact_type"`
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
func (p *Client) UploadObservationLogs(ctx context.Context, data []byte, sessionID string) error {
	headers := map[string]string{
		"Prefer":            "params=multiple-objects",
		"Content-Type":      "text/csv",
		"X-PGAI-Session-ID": sessionID,
		"X-PGAI-Part":       "1", // TODO (akartasov): Support chunks.
	}

	var respData APIResponse

	if err := p.doUpload(ctx, "/rpc/dblab_session_upload_log", data, headers, newUploadParser(&respData)); err != nil {
		return errors.Wrap(err, "failed to upload request")
	}

	if respData.Code != "" || respData.Details != "" {
		log.Dbg(fmt.Sprintf("Unsuccessful response given. Code: %v. Details: %v", respData.Code, respData.Details))

		return errors.New("failed to upload observation logs")
	}

	return nil
}

// UploadObservationArtifact makes an HTTP request to upload an observation artifact to Platform.
func (p *Client) UploadObservationArtifact(ctx context.Context, data []byte, sessionID, artifactType string) error {
	headers := map[string]string{
		"Prefer":               "params=single-object",
		"Content-Type":         "application/json",
		"X-PGAI-Session-ID":    sessionID,
		"X-PGAI-Artifact-Type": artifactType,
	}

	log.Dbg("Uploading artifact: ", artifactType)

	respData := UploadArtifactResponse{}

	if err := p.doUpload(ctx, "/rpc/dblab_session_upload_artifact", data, headers, newJSONParser(&respData)); err != nil {
		return errors.Wrap(err, "failed to upload")
	}

	log.Dbg("Upload artifacts response: ", respData)

	return nil
}
