package srv

import (
	"bufio"
	"encoding/base64"
	"io"
	"net/http"

	"github.com/ahmetalpbalkan/dlog"
	"github.com/docker/docker/api/types/container"
	"github.com/gorilla/websocket"

	"gitlab.com/postgres-ai/database-lab/v3/internal/srv/api"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util/engine"
)

// websocketAuth generates a one-time token to access web-socket handlers.
func (s *Server) websocketAuth(w http.ResponseWriter, r *http.Request) {
	tokenString, err := s.wsService.tokenKeeper.IssueToken()
	if err != nil {
		api.SendError(w, r, err)
		return
	}

	if err := api.WriteJSON(w, http.StatusOK, models.WSToken{Token: tokenString}); err != nil {
		api.SendError(w, r, err)
		return
	}
}

// instanceLogs provides logs entries of the Database Lab Engine instance via web-sockets connection.
func (s *Server) instanceLogs(w http.ResponseWriter, r *http.Request) {
	s.wsService.upgrader.CheckOrigin = func(r *http.Request) bool {
		requestOrigin := r.Header.Get("Origin")

		var uiURL string

		if s.wsService.uiManager.IsEnabled() {
			if s.wsService.uiManager.GetHost() == "" || s.wsService.uiManager.GetHost() == engine.DefaultListenerHost {
				return true
			}

			uiURL = s.wsService.uiManager.OriginURL()
		}

		platformURL := s.Platform.OriginURL()

		log.Dbg("Request Origin", requestOrigin)
		log.Dbg("UI URL", uiURL)
		log.Dbg("Platform URL", platformURL)

		return (uiURL != "" && uiURL == requestOrigin) || (platformURL != "" && platformURL == requestOrigin)
	}

	conn, err := s.wsService.upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Err(err)

		return
	}

	defer func() {
		if err := conn.Close(); err != nil {
			log.Dbg("Failed to close connection", err)
		}
	}()

	readCloser, err := s.docker.ContainerLogs(r.Context(), s.engProps.ContainerName, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Since:      logsSinceInterval,
		Follow:     true,
	})
	if err != nil {
		log.Err("failed to get container logs", err)

		if writingErr := conn.WriteMessage(websocket.TextMessage, []byte(err.Error())); writingErr != nil {
			log.Dbg("Failed to report about error", err)
		}

		return
	}

	defer func() {
		if err := readCloser.Close(); err != nil {
			log.Dbg("Failed to close reader of logs", err)
		}
	}()

	sc := bufio.NewScanner(dlog.NewReader(readCloser))
	for sc.Scan() {
		encodedLine := base64.StdEncoding.EncodeToString(s.filterLogLine(sc.Bytes()))

		err := conn.WriteMessage(websocket.TextMessage, []byte(encodedLine))
		if err != nil && err != io.EOF {
			log.Err(err)
			break
		}
	}

	if sc.Err() != nil {
		log.Dbg(err)

		return
	}
}

func (s *Server) filterLogLine(inputLine []byte) []byte {
	return s.filtering.ReplaceAll(inputLine)
}
