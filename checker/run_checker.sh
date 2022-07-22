#!/bin/sh -l

KEEP_CLONE=false

if [[ "${INPUT_DOWNLOAD_ARTIFACTS}" == "true" ]]; then
  KEEP_CLONE=true
fi

PROJECT_URL=${CI_MERGE_REQUEST_SOURCE_PROJECT_URL:-${CI_PROJECT_URL}}
PROJECT_ID=${CI_MERGE_REQUEST_PROJECT_ID:-${CI_PROJECT_ID}}
REF=${CI_COMMIT_TAG:-${CI_COMMIT_SHA}}


JSON_DATA=$(jq -n -c \
  --arg repo "$PROJECT_ID" \
  --arg ref "$REF" \
  --arg commands "$INPUT_COMMANDS" \
  --arg db_name "$INPUT_DBNAME" \
  --arg username "$GITLAB_USER_LOGIN" \
  --arg username_full "$GITLAB_USER_NAME" \
  --arg username_link "${CI_SERVER_URL}/$GITLAB_USER_LOGIN" \
  --arg branch "${CI_COMMIT_TAG:-${CI_COMMIT_REF_NAME}}" \
  --arg branch_link "${PROJECT_URL}/-/tree/${CI_COMMIT_TAG:-${CI_COMMIT_REF_NAME}}" \
  --arg commit "${CI_COMMIT_SHA}" \
  --arg commit_link "${PROJECT_URL}/-/commit/${CI_COMMIT_SHA}" \
  --arg request_link "${CI_MERGE_REQUEST_PROJECT_URL}/-/merge_requests/${CI_MERGE_REQUEST_IID}" \
  --arg diff_link "${CI_MERGE_REQUEST_PROJECT_URL}/-/merge_requests/${CI_MERGE_REQUEST_IID}/diffs" \
  --arg migration_envs "$INPUT_MIGRATION_ENVS" \
  --arg observation_interval "$INPUT_OBSERVATION_INTERVAL" \
  --arg max_lock_duration "$INPUT_MAX_LOCK_DURATION" \
  --arg max_duration "$INPUT_MAX_DURATION" \
  --argjson keep_clone "$KEEP_CLONE" \
  '{source: {repo: $repo, ref: $ref, branch: $branch, branch_link: $branch_link, commit: $commit, commit_link: $commit_link, request_link: $request_link, diff_link: $diff_link}, username: $username, username_full: $username_full, username_link: $username_link, db_name: $db_name, commands: $commands | rtrimstr("\n") | split("\n"), migration_envs: $migration_envs | rtrimstr("\n") | split("\n"), observation_config: { observation_interval: $observation_interval|tonumber, max_lock_duration: $max_lock_duration|tonumber, max_duration: $max_duration|tonumber}, keep_clone: $keep_clone}')

echo $JSON_DATA

# Remove when the GitLab source becomes supported
# exit 0

response_code=$(curl -k --show-error --silent --location --request POST "${DLMC_CI_ENDPOINT}/migration/run" --write-out "%{http_code}" \
--header "Verification-Token: ${DLMC_VERIFICATION_TOKEN}" \
--header 'Content-Type: application/json' \
--output response.json \
--data "${JSON_DATA}")

cat response.json

jq . response.json

if [[ $response_code -ne 200 ]]; then
  echo "Migration status code: ${response_code}"
  exit 1
fi

status=$(jq -r '.session.result.status' response.json)

if [[ $status != "passed" ]]; then
  echo "Migration status: ${status}"
  exit 1
fi

echo "::set-output name=response::$(cat response.json)"

clone_id=$(jq -r '.clone_id' response.json)
session_id=$(jq -r '.session.session_id' response.json)

if [[ ! $KEEP_CLONE ]]; then
  exit 0
fi

# Download artifacts
mkdir artifacts

download_artifacts() {
    artifact_code=$(curl -k --show-error --silent "${DLMC_CI_ENDPOINT}/artifact/download?artifact_type=$1&session_id=$2&clone_id=$3" --write-out "%{http_code}" \
         --header "Verification-Token: ${DLMC_VERIFICATION_TOKEN}" \
         --header 'Content-Type: application/json' \
         --output artifacts/$1)

    if [[ $artifact_code -ne 200 ]]; then
      echo "Downloading $1, invalid status code given: ${artifact_code}"
      return
    fi

    echo "Artifact \"$1\" has been downloaded to the artifacts directory"
}

cat response.json | jq -c -r '.session.artifacts[]' | while read artifact; do
    download_artifacts $artifact $session_id $clone_id
done

# Stop the running clone
response_code=$(curl --show-error --silent "${DLMC_CI_ENDPOINT}/artifact/stop?clone_id=${clone_id}" --write-out "%{http_code}" \
     --header "Verification-Token: ${DLMC_VERIFICATION_TOKEN}" \
     --header 'Content-Type: application/json')

if [[ $response_code -ne 200 ]]; then
  echo "Invalid status code given on destroy clone: ${artifact_code}"
fi
