/*
2021 © Postgres.ai
*/

package dblabapi

import (
	"context"
	"encoding/json"
	"net/url"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/v3/internal/estimator"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
)

// EstimatorClient defines a client to perform timing estimation.
type EstimatorClient struct {
	conn    *websocket.Conn
	results chan estimator.Result
	ready   chan struct{}
}

// Close closes connection.
func (e *EstimatorClient) Close() error {
	return e.conn.Close()
}

// Wait waits for connection readiness.
func (e *EstimatorClient) Wait() chan struct{} {
	return e.ready
}

// ReadResult returns estimation results.
func (e *EstimatorClient) ReadResult() estimator.Result {
	return <-e.results
}

// SetReadBlocks sends a number of read blocks.
func (e *EstimatorClient) SetReadBlocks(readBlocks uint64) error {
	result := estimator.ReadBlocksEvent{
		EventType:  estimator.ReadBlocksType,
		ReadBlocks: readBlocks,
	}

	readBlocksData, err := json.Marshal(result)
	if err != nil {
		return err
	}

	return e.conn.WriteMessage(websocket.TextMessage, readBlocksData)
}

// Estimate creates connection for estimation session.
func (c *Client) Estimate(ctx context.Context, cloneID, pid string) (*EstimatorClient, error) {
	u := c.URL("/estimate")
	u.Scheme = "ws"

	values := url.Values{}
	values.Add("clone_id", cloneID)
	values.Add("pid", pid)
	u.RawQuery = values.Encode()

	log.Dbg("connecting to ", u.String())

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect")
	}

	est := &EstimatorClient{
		conn:    conn,
		results: make(chan estimator.Result, 1),
		ready:   make(chan struct{}, 1),
	}

	go func() {
		if err := est.receiveMessages(ctx); err != nil {
			log.Dbg("error while receive messages: ", err)
		}
	}()

	return est, nil
}

func (e *EstimatorClient) receiveMessages(ctx context.Context) error {
	for {
		if ctx.Err() != nil {
			log.Msg(ctx.Err())
			break
		}

		_, message, err := e.conn.ReadMessage()
		if err != nil {
			return err
		}

		event := estimator.Event{}
		if err := json.Unmarshal(message, &event); err != nil {
			return err
		}

		switch event.EventType {
		case estimator.ReadyEventType:
			e.ready <- struct{}{}

		case estimator.ResultEventType:
			result := estimator.ResultEvent{}
			if err := json.Unmarshal(message, &result); err != nil {
				log.Dbg("failed to read the result event: ", err)
				break
			}

			e.results <- result.Payload
			if err := e.conn.Close(); err != nil {
				log.Dbg("failed to close connection: ", err)
			}

			return nil
		}

		log.Dbg("received unknown event type: ", event.EventType, string(message))
	}

	return nil
}
