#!/bin/bash

set -euo pipefail

docker_file=${DOCKER_FILE:-""}
tags=${TAG:-""}

# Docker login for GCP.
echo $GCP_SERVICE_ACCOUNT | base64 -d > ./key.json
docker login -u _json_key --password-stdin https://gcr.io < ./key.json

tags_build=""
tags_push=""

IFS=',' read -ra ADDR string <<EOF
$tags
EOF

for tag in "${ADDR[@]}"; do
  tags_build="${tags_build} --tag ${tag}"
  tags_push="${tags_push}${tag}\n"
done

# Set envs before building the container image, because env vars
# will not be available in user's browser.
source "./web/packages/platform/deploy/configs/${NAMESPACE}.sh"

set -x
docker build \
      --build-arg ARG_REACT_APP_API_SERVER="${REACT_APP_API_SERVER}" \
      --build-arg ARG_PUBLIC_URL="${PUBLIC_URL}" \
      --build-arg ARG_REACT_APP_SIGNIN_URL="${REACT_APP_SIGNIN_URL}" \
      --build-arg ARG_REACT_APP_WS_SERVER="${REACT_APP_WS_SERVER}" \
      --build-arg ARG_REACT_APP_EXPLAIN_DEPESZ_SERVER="${REACT_APP_EXPLAIN_DEPESZ_SERVER}" \
      --build-arg ARG_REACT_APP_EXPLAIN_PEV2_SERVER="${REACT_APP_EXPLAIN_PEV2_SERVER}" \
      --build-arg ARG_REACT_APP_STRIPE_API_KEY="${REACT_APP_STRIPE_API_KEY}" \
      --build-arg ARG_REACT_APP_AUTH_URL="${REACT_APP_AUTH_URL}" \
      --build-arg ARG_REACT_APP_ROOT_URL="${REACT_APP_ROOT_URL}" \
      --build-arg ARG_REACT_APP_SENTRY_DSN="${REACT_APP_SENTRY_DSN}" \
      $tags_build --file ./web/packages/platform/Dockerfile .
set +x

echo -e "$tags_push" | while read -r tag; do
  [ -z "$tag" ] && continue
  set -x
  docker push $tag
  set +x
done
