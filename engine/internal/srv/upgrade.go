/*
2026 © Postgres.ai
*/

package srv

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/distribution/reference"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"

	"gitlab.com/postgres-ai/database-lab/v3/internal/cloning"
	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/probe"
	"gitlab.com/postgres-ai/database-lab/v3/internal/srv/api"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/client/dblabapi/types"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/log"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/models"
	"gitlab.com/postgres-ai/database-lab/v3/pkg/util"
)

// maxUpgradeMajorJump is how many majors back the upgrade image carries binaries for. It mirrors
// the OLD_MAJORS default of Dockerfile.pg-upgrade.
const maxUpgradeMajorJump = 4

// upgradeImageInput is everything needed to decide which image an upgraded clone runs.
type upgradeImageInput struct {
	// CurrentImage is the image the clone runs now.
	CurrentImage string
	// CurrentVersion is the major the clone runs now, empty when the engine has not recorded one.
	CurrentVersion string
	TargetVersion  int
	// RequestedImage is the caller's explicit override; it wins when set.
	RequestedImage string
	// AllowedRepositories restricts which repositories RequestedImage may name. An empty list
	// allows any, which is what an instance pulling its images from wherever its administrator
	// points it needs.
	AllowedRepositories []string
}

func (s *Server) upgradeClone(w http.ResponseWriter, r *http.Request) {
	cloneID := mux.Vars(r)["id"]

	if cloneID == "" {
		api.SendBadRequestError(w, r, "ID must not be empty")
		return
	}

	var upgradeRequest types.CloneUpgradeRequest

	if r.Body != http.NoBody {
		if err := json.NewDecoder(r.Body).Decode(&upgradeRequest); err != nil {
			api.SendError(w, r, errors.Wrap(err, "failed to parse request parameters"))
			return
		}
	}

	clone, err := s.Cloning.GetClone(cloneID)
	if err != nil {
		api.SendNotFoundError(w, r)
		return
	}

	// An observation session reads the clone's log directory and expects a live server, both of
	// which the upgrade takes away.
	if _, err := s.Observer.GetObservingClone(cloneID); err == nil {
		api.SendBadRequestError(w, r, "clone is under observation; stop the observation session before upgrading")
		return
	}

	// The target is a property of the instance, not of the request: the configured upgrade image
	// is built on the target major's binaries, so asking for any other version could only be
	// answered by failing.
	targetVersion, err := s.provisioner.UpgradeTargetVersion()
	if err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	if err := checkDefaultConfigDir(targetVersion); err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	// A clone that has never been upgraded carries no override, so the engine-wide image and
	// version are what it actually runs.
	targetImage, err := resolveUpgradeImage(upgradeImageInput{
		CurrentImage:        firstNonEmpty(clone.DockerImage, s.provisioner.ContainerOptions().DockerImage),
		CurrentVersion:      firstNonEmpty(clone.DBVersion, s.provisioner.DetectDBVersion()),
		TargetVersion:       targetVersion,
		RequestedImage:      upgradeRequest.DockerImage,
		AllowedRepositories: s.provisioner.UpgradeImageAllowList(),
	})
	if err != nil {
		api.SendBadRequestError(w, r, err.Error())
		return
	}

	if err := s.Cloning.UpgradeClone(cloneID, cloning.UpgradeRequest{
		TargetVersion: targetVersion,
		TargetImage:   targetImage,
	}); err != nil {
		api.SendError(w, r, errors.Wrap(err, "failed to upgrade clone"))
		return
	}

	log.Dbg(fmt.Sprintf("Clone ID=%s is being upgraded to PostgreSQL %d", cloneID, targetVersion))

	if err := api.WriteJSON(w, http.StatusOK, models.CloneUpgradePlan{
		TargetVersion: targetVersion,
		DockerImage:   targetImage,
	}); err != nil {
		api.SendError(w, r, err)
		return
	}
}

// cloneUpgradeStatus reports what the upgrade endpoint would accept right now. Everything it
// checks is instance-wide, so a client can both decide whether to offer the action and name the
// version it lands on without asking for an upgrade first.
func (s *Server) cloneUpgradeStatus() models.CloneUpgrade {
	targetVersion, err := s.provisioner.UpgradeTargetVersion()
	if err != nil {
		return models.CloneUpgrade{Reason: err.Error()}
	}

	if err := checkDefaultConfigDir(targetVersion); err != nil {
		return models.CloneUpgrade{Reason: err.Error()}
	}

	return models.CloneUpgrade{Available: true, TargetVersion: targetVersion}
}

// resolveUpgradeImage decides which image the upgraded clone runs. An explicit request wins;
// otherwise only the major of the current tag is replaced, so the extension bundle and the glibc
// build carry over unchanged. Changing the glibc build across an upgrade would change collation
// behaviour, and pg_upgrade does not reindex.
func resolveUpgradeImage(in upgradeImageInput) (string, error) {
	// Every caller resolves the target from the instance configuration, which cannot yield this.
	// Checking anyway keeps the failure legible: without it a zero target is refused further down
	// by a message blaming the clone's own tag for not being a release tag.
	if in.TargetVersion <= 0 {
		return "", errors.New("the upgrade target must be a positive PostgreSQL major version")
	}

	if err := validateVersionJump(in.CurrentVersion, in.TargetVersion); err != nil {
		return "", err
	}

	if in.RequestedImage != "" {
		if err := validateRequestedImage(in.RequestedImage, in.TargetVersion, in.AllowedRepositories); err != nil {
			return "", err
		}

		return in.RequestedImage, nil
	}

	if in.CurrentImage == "" {
		return "", errors.New("cannot derive the target image because the clone has no recorded image; " +
			"pass dockerImage explicitly")
	}

	repository, tag := probe.SplitImageTag(in.CurrentImage)
	if tag == "" {
		return "", fmt.Errorf("cannot derive the target image from %q because it carries no tag; "+
			"pass dockerImage explicitly", in.CurrentImage)
	}

	// probe owns the release-tag grammar and is unit-tested against it; anything outside that
	// grammar (pre-releases, branch-named CI tags, dotted majors) would yield an image
	// reference that does not exist.
	targetTag, ok := probe.SubstituteTagMajor(tag, in.TargetVersion)
	if !ok {
		return "", fmt.Errorf("cannot derive the target image from %q because %q is not a release tag; "+
			"pass dockerImage explicitly", in.CurrentImage, tag)
	}

	return repository + ":" + targetTag, nil
}

// validateRequestedImage checks an explicit image override before the engine pulls it and runs a
// clone on it. The reference has to parse, which keeps anything that is not an image name out of
// the command the provisioner builds around it; its major, when the tag states one, has to be the
// major the cluster is being converted to; and it has to name an allowed repository whenever the
// instance lists any.
func validateRequestedImage(image string, targetVersion int, allowedRepositories []string) error {
	parsed, err := reference.ParseNormalizedNamed(image)
	if err != nil {
		return fmt.Errorf("dockerImage %q is not a valid image reference: %w", image, err)
	}

	// Booting a converted cluster on another major's binaries is exactly what the upgrade avoids
	// doing. A tag outside the release grammar states no major and is taken on trust.
	if major, ok := probe.MajorFromImage(image); ok && major != targetVersion {
		return fmt.Errorf("dockerImage %q runs PostgreSQL %d, but this instance upgrades to %d",
			image, major, targetVersion)
	}

	if len(allowedRepositories) == 0 {
		return nil
	}

	repository := reference.TrimNamed(parsed).Name()

	for _, allowed := range allowedRepositories {
		if repository == normalizeRepository(allowed) {
			return nil
		}
	}

	return fmt.Errorf("dockerImage %q is not listed in provision.upgradeImageAllowList", image)
}

// normalizeRepository drops any tag or digest and expands the implicit registry and namespace, so
// that a configured "postgresai/extended-postgres" matches a requested
// "docker.io/postgresai/extended-postgres:17". An entry that does not parse is compared verbatim
// and simply never matches a parsed reference.
func normalizeRepository(image string) string {
	parsed, err := reference.ParseNormalizedNamed(image)
	if err != nil {
		return image
	}

	return reference.TrimNamed(parsed).Name()
}

// firstNonEmpty returns the first value that is set.
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}

	return ""
}

// validateVersionJump rejects targets the upgrade image has no old binaries for. The current
// version is only known once the engine has recorded one, so an empty value skips the check and
// leaves it to the provisioner, which reads PG_VERSION from the data directory.
func validateVersionJump(currentVersion string, targetVersion int) error {
	if currentVersion == "" {
		return nil
	}

	current, err := strconv.Atoi(strings.SplitN(currentVersion, ".", 2)[0])
	if err != nil {
		return nil
	}

	if targetVersion <= current {
		return fmt.Errorf("target version %d must be greater than the current version %d", targetVersion, current)
	}

	if targetVersion-current > maxUpgradeMajorJump {
		return fmt.Errorf("cannot upgrade from %d to %d: the upgrade image carries binaries for at most %d preceding majors",
			current, targetVersion, maxUpgradeMajorJump)
	}

	return nil
}

// checkDefaultConfigDir makes sure the engine can configure the upgraded cluster. Without a
// matching directory the clone would convert successfully and then boot unreachable.
func checkDefaultConfigDir(targetVersion int) error {
	configPath, err := util.GetStandardConfigPath(path.Join("postgres", "default", strconv.Itoa(targetVersion)))
	if err != nil {
		return fmt.Errorf("failed to resolve the standard config path: %w", err)
	}

	info, err := os.Stat(configPath)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("PostgreSQL %d is not supported as an upgrade target: no default configuration shipped for it",
			targetVersion)
	}

	return nil
}
