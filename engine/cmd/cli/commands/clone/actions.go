/*
2020 © Postgres.ai
*/

// Package clone provides clones management commands.
package clone

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"gitlab.com/postgres-ai/database-lab/v3/cmd/cli/commands"
	"gitlab.com/postgres-ai/database-lab/v3/internal/observer"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/client/dblabapi/types"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
)

// list runs a request to list clones of an instance.
func list(cliCtx *cli.Context) error {
	dblabClient, err := commands.ClientByCLIContext(cliCtx)
	if err != nil {
		return err
	}

	body, err := dblabClient.ListClonesRaw(cliCtx.Context)
	if err != nil {
		return err
	}

	defer func() { _ = body.Close() }()

	viewCloneList := &models.CloneListView{
		Cloning: models.CloningView{
			Clones: make([]*models.CloneView, 0),
		},
	}

	if err := json.NewDecoder(body).Decode(&viewCloneList); err != nil {
		return err
	}

	commandResponse, err := json.MarshalIndent(viewCloneList.Cloning.Clones, "", "    ")
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(cliCtx.App.Writer, string(commandResponse))

	return err
}

// status runs a request to get clone info.
func status(cliCtx *cli.Context) error {
	dblabClient, err := commands.ClientByCLIContext(cliCtx)
	if err != nil {
		return err
	}

	body, err := dblabClient.GetCloneRaw(cliCtx.Context, cliCtx.Args().First())
	if err != nil {
		return err
	}

	defer func() { _ = body.Close() }()

	var cloneView *models.CloneView

	if err := json.NewDecoder(body).Decode(&cloneView); err != nil {
		return err
	}

	commandResponse, err := json.MarshalIndent(cloneView, "", "    ")
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(cliCtx.App.Writer, string(commandResponse))

	return err
}

// create runs a request to create a new clone.
func create(cliCtx *cli.Context) error {
	dblabClient, err := commands.ClientByCLIContext(cliCtx)
	if err != nil {
		return err
	}

	cloneRequest := types.CloneCreateRequest{
		ID:        cliCtx.String("id"),
		Protected: cliCtx.Bool("protected"),
		DB: &types.DatabaseRequest{
			Username:   cliCtx.String("username"),
			Password:   cliCtx.String("password"),
			Restricted: cliCtx.Bool("restricted"),
			DBName:     cliCtx.String("db-name"),
		},
	}

	if cliCtx.IsSet("snapshot-id") {
		cloneRequest.Snapshot = &types.SnapshotCloneFieldRequest{ID: cliCtx.String("snapshot-id")}
	}

	cloneRequest.ExtraConf = splitFlags(cliCtx.StringSlice("extra-config"))

	var clone *models.Clone

	if cliCtx.Bool("async") {
		clone, err = dblabClient.CreateCloneAsync(cliCtx.Context, cloneRequest)
	} else {
		clone, err = dblabClient.CreateClone(cliCtx.Context, cloneRequest)
	}

	if err != nil {
		return err
	}

	viewClone, err := convertCloneView(clone)
	if err != nil {
		return err
	}

	commandResponse, err := json.MarshalIndent(viewClone, "", "    ")
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(cliCtx.App.Writer, string(commandResponse))

	return err
}

// update runs a request to update an existing clone.
func update(cliCtx *cli.Context) error {
	dblabClient, err := commands.ClientByCLIContext(cliCtx)
	if err != nil {
		return err
	}

	updateRequest := types.CloneUpdateRequest{
		Protected: cliCtx.Bool("protected"),
	}

	cloneID := cliCtx.Args().First()

	clone, err := dblabClient.UpdateClone(cliCtx.Context, cloneID, updateRequest)
	if err != nil {
		return err
	}

	viewClone, err := convertCloneView(clone)
	if err != nil {
		return err
	}

	commandResponse, err := json.MarshalIndent(viewClone, "", "    ")
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(cliCtx.App.Writer, string(commandResponse))

	return err
}

func convertCloneView(clone *models.Clone) (*models.CloneView, error) {
	data, err := json.Marshal(clone)
	if err != nil {
		return nil, err
	}

	var viewClone *models.CloneView
	if err = json.Unmarshal(data, &viewClone); err != nil {
		return nil, err
	}

	return viewClone, nil
}

// reset runs a request to reset clone.
func reset(cliCtx *cli.Context) error {
	dblabClient, err := commands.ClientByCLIContext(cliCtx)
	if err != nil {
		return err
	}

	cloneID := cliCtx.Args().First()
	resetOptions := types.ResetCloneRequest{
		Latest:     cliCtx.Bool(cloneResetLatestFlag),
		SnapshotID: cliCtx.String(cloneResetSnapshotIDFlag),
	}

	if cliCtx.Bool("async") {
		err = dblabClient.ResetCloneAsync(cliCtx.Context, cloneID, resetOptions)
	} else {
		err = dblabClient.ResetClone(cliCtx.Context, cloneID, resetOptions)
	}

	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(cliCtx.App.Writer, "The clone has been successfully reset: %s\n", cloneID)

	return err
}

// destroy runs a request to destroy clone.
func destroy(cliCtx *cli.Context) error {
	dblabClient, err := commands.ClientByCLIContext(cliCtx)
	if err != nil {
		return err
	}

	cloneID := cliCtx.Args().First()

	if cliCtx.Bool("async") {
		err = dblabClient.DestroyCloneAsync(cliCtx.Context, cloneID)
	} else {
		err = dblabClient.DestroyClone(cliCtx.Context, cloneID)
	}

	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(cliCtx.App.Writer, "The clone has been successfully destroyed: %s\n", cloneID)

	return err
}

// startObservation runs a request to startObservation clone.
func startObservation(cliCtx *cli.Context) error {
	dblabClient, err := commands.ClientByCLIContext(cliCtx)
	if err != nil {
		return err
	}

	cloneID := cliCtx.Args().First()

	observationConfig := types.Config{
		ObservationInterval: cliCtx.Uint64("observation-interval"),
		MaxLockDuration:     cliCtx.Uint64("max-lock-duration"),
		MaxDuration:         cliCtx.Uint64("max-duration"),
	}

	start := types.StartObservationRequest{
		CloneID: cloneID,
		Config:  observationConfig,
		Tags:    splitFlags(cliCtx.StringSlice("tags")),
		DBName:  cliCtx.String("db-name"),
	}

	session, err := dblabClient.StartObservation(cliCtx.Context, start)
	if err != nil {
		return err
	}

	commandResponse, err := json.MarshalIndent(session, "", "    ")
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(cliCtx.App.Writer, string(commandResponse))

	return err
}

// stopObservation shows observing summary and check satisfaction of performance requirements.
func stopObservation(cliCtx *cli.Context) error {
	dblabClient, err := commands.ClientByCLIContext(cliCtx)
	if err != nil {
		return err
	}

	cloneID := cliCtx.Args().First()

	result, err := dblabClient.StopObservation(cliCtx.Context, types.StopObservationRequest{CloneID: cloneID})
	if err != nil {
		return err
	}

	commandResponse, err := json.MarshalIndent(result, "", "    ")
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(cliCtx.App.Writer, string(commandResponse))

	return err
}

// summaryObservation returns the observing summary artifact.
func summaryObservation(cliCtx *cli.Context) error {
	dblabClient, err := commands.ClientByCLIContext(cliCtx)
	if err != nil {
		return err
	}

	cloneID := cliCtx.String("clone-id")
	sessionID := cliCtx.String("session-id")

	result, err := dblabClient.SummaryObservation(cliCtx.Context, cloneID, sessionID)
	if err != nil {
		return err
	}

	commandResponse, err := json.MarshalIndent(result, "", "    ")
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(cliCtx.App.Writer, string(commandResponse))

	return err
}

func downloadArtifact(cliCtx *cli.Context) error {
	dblabClient, err := commands.ClientByCLIContext(cliCtx)
	if err != nil {
		return err
	}

	cloneID := cliCtx.String("clone-id")
	sessionID := cliCtx.String("session-id")
	artifactType := cliCtx.String("artifact-type")
	outputPath := cliCtx.String("output")

	body, err := dblabClient.DownloadArtifact(cliCtx.Context, cloneID, sessionID, artifactType)
	if err != nil {
		return err
	}

	defer func() {
		if err := body.Close(); err != nil {
			log.Err(err)
		}
	}()

	if outputPath == "" {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}

		outputPath = path.Join(wd, observer.BuildArtifactFilename(artifactType))
	}

	artifactFile, err := os.Create(outputPath)
	if err != nil {
		return errors.Wrapf(err, "failed to create file %s", outputPath)
	}

	defer func() { _ = artifactFile.Close() }()

	if _, err := io.Copy(artifactFile, body); err != nil {
		return err
	}

	_, err = fmt.Fprintf(cliCtx.App.Writer, "The file has been successfully downloaded: %s\n", outputPath)

	return err
}

func forward(cliCtx *cli.Context) error {
	remoteURL, err := url.Parse(cliCtx.String(commands.URLKey))
	if err != nil {
		return err
	}

	wg := &sync.WaitGroup{}

	port, err := retrieveClonePort(cliCtx, wg, remoteURL)
	if err != nil {
		return err
	}

	wg.Wait()

	log.Dbg(fmt.Sprintf("The clone port has been retrieved: %s", port))

	remoteURL.Host = commands.BuildHostname(remoteURL.Hostname(), port)

	tunnel, err := commands.BuildTunnel(cliCtx, remoteURL)
	if err != nil {
		return err
	}

	if err := tunnel.Open(); err != nil {
		return err
	}

	log.Msg(fmt.Sprintf("The clone is available by address: %s", tunnel.Endpoints.Local))

	if err := tunnel.Listen(cliCtx.Context); err != nil {
		return err
	}

	return nil
}

func retrieveClonePort(cliCtx *cli.Context, wg *sync.WaitGroup, remoteHost *url.URL) (string, error) {
	tunnel, err := commands.BuildTunnel(cliCtx, remoteHost)
	if err != nil {
		return "", err
	}

	if err := tunnel.Open(); err != nil {
		return "", err
	}

	const goroutineCount = 1

	wg.Add(goroutineCount)

	go func() {
		defer wg.Done()

		if err := tunnel.Listen(cliCtx.Context); err != nil {
			log.Fatal(err)
		}
	}()

	defer func() {
		log.Dbg("Stop tunnel to DBLab")

		if err := tunnel.Stop(); err != nil {
			log.Err(err)
		}
	}()

	log.Dbg("Retrieving clone port")

	dblabClient, err := commands.ClientByCLIContext(cliCtx)
	if err != nil {
		return "", err
	}

	clone, err := dblabClient.GetClone(cliCtx.Context, cliCtx.Args().First())
	if err != nil {
		return "", err
	}

	return clone.DB.Port, nil
}

func splitFlags(flags []string) map[string]string {
	const maxSplitParts = 2

	extraConfig := make(map[string]string, len(flags))

	if len(flags) == 0 {
		return extraConfig
	}

	for _, cfg := range flags {
		parsed := strings.SplitN(cfg, "=", maxSplitParts)
		extraConfig[parsed[0]] = parsed[1]
	}

	return extraConfig
}
