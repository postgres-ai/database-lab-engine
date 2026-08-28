#!/bin/bash

set -euo pipefail

docker_file=${DOCKER_FILE:-"Dockerfile"}
tags=${TAGS:-""}
# Extra "--build-arg name=value" pairs, used by images that are parameterized at build time
# (Dockerfile.pg-upgrade takes the extended-postgres base it derives from).
build_args=${BUILD_ARGS:-""}

registry_user=${REGISTRY_USER:-"${CI_REGISTRY_USER}"}
registry_password=${REGISTRY_PASSWORD:-"${CI_REGISTRY_PASSWORD}"}
registry=${REGISTRY:-"${CI_REGISTRY}"}

docker login --username $registry_user --password "${registry_password}" $registry

tags_build=""
tags_push=""

IFS=',' read -ra ADDR string <<EOF
$tags
EOF

for tag in "${ADDR[@]}"; do
  tags_build="${tags_build} --tag $(echo $tag | tr '[:upper:]' '[:lower:]')"
  tags_push="${tags_push}$(echo $tag | tr '[:upper:]' '[:lower:]')\n"
done

set -x
docker build $tags_build $build_args --file $docker_file .
set +x

# Smoke-test the trimmed runtime images:
#   1. Docker CLI loads and runs (`docker --version`).
#   2. ZFS-bearing images still ship a working `zfs` binary. We use
#      `zfs --help` instead of `zfs --version` because the latter exits
#      non-zero when the kernel module is absent (the CI builder is DinD,
#      which has none) — `--help` only exercises the userspace binary.
#   3. None of the daemon / runtime / compose binaries that ship in the
#      full `docker:*` image (and are NOT needed in a CLI-only runtime)
#      have crept back in. The forbidden list is daemon-side only —
#      the `docker` CLI itself is required and stays.
case "$docker_file" in
  Dockerfile.dblab-server|Dockerfile.dblab-server-debug|Dockerfile.dblab-cli|Dockerfile.ci-checker)
    if [ "${#ADDR[@]}" -eq 0 ] || [ -z "${ADDR[0]:-}" ]; then
      echo "ERROR: smoke test cannot run, TAGS is empty" >&2
      exit 1
    fi
    smoke_image=$(printf '%s' "${ADDR[0]}" | tr '[:upper:]' '[:lower:]')
    set -x
    docker run --rm "$smoke_image" docker --version
    set +x
    case "$docker_file" in
      Dockerfile.dblab-server|Dockerfile.dblab-server-debug)
        set -x
        docker run --rm "$smoke_image" zfs --help >/dev/null
        set +x
        ;;
    esac
    for daemon in dockerd containerd ctr docker-buildx docker-compose; do
      # Pass the daemon name via env so it is never concatenated into the
      # inner shell string. `|| true` swallows `command -v`'s exit-1 for
      # "not found"; a real `docker run` failure (image won't start, etc.)
      # still aborts the build via `set -e` on the assignment itself.
      found=$(docker run --rm -e D="$daemon" "$smoke_image" sh -c 'command -v "$D" || true')
      if [ -n "$found" ]; then
        echo "ERROR: forbidden runtime binary '$daemon' is present in $smoke_image" >&2
        exit 1
      fi
    done
    ;;
  Dockerfile.pg-upgrade)
    # The upgrade image is only useful if it carries the target-major pg_upgrade, at least one
    # older server bindir to point --old-bindir at, and a runnable entrypoint.
    if [ "${#ADDR[@]}" -eq 0 ] || [ -z "${ADDR[0]:-}" ]; then
      echo "ERROR: smoke test cannot run, TAGS is empty" >&2
      exit 1
    fi
    smoke_image=$(printf '%s' "${ADDR[0]}" | tr '[:upper:]' '[:lower:]')
    set -x
    docker run --rm --entrypoint sh "$smoke_image" -c '
      set -eu
      test -x /usr/local/bin/pg_upgrade_clone.sh
      test -x "/usr/lib/postgresql/${PG_MAJOR}/bin/pg_upgrade"
      ls -d /usr/lib/postgresql/*/bin/pg_ctl | grep -qv "/${PG_MAJOR}/bin/"
    '
    set +x
    ;;
esac

echo -e "$tags_push" | while read -r tag; do
  [ -z "$tag" ] && continue
  set -x
  docker push $tag
  set +x
done
