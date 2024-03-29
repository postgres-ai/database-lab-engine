#!/bin/bash

set -euo pipefail

docker_file=${DOCKER_FILE:-""}
tags=${TAGS:-""}

registry_user=${REGISTRY_USER:-"${CI_REGISTRY_USER}"}
registry_password=${REGISTRY_PASSWORD:-"${CI_REGISTRY_PASSWORD}"}
registry=${REGISTRY:-"${CI_REGISTRY}"}

echo "${registry_password}" | docker login --username $registry_user --password-stdin $registry

tags_build=""
tags_push=""

IFS=',' read -ra ADDR string <<EOF
$tags
EOF

for tag in "${ADDR[@]}"; do
  tags_build="${tags_build} --tag ${tag}"
  tags_push="${tags_push}${tag}\n"
done

set -x
DOCKER_BUILDKIT=1 docker build --build-arg API_URL_PREFIX=/api --build-arg WS_URL_PREFIX=/ws $tags_build --file ./ui/packages/ce/Dockerfile .
set +x

echo -e "$tags_push" | while read -r tag; do
  [ -z "$tag" ] && continue
  set -x
  docker push $tag
  set +x
done
