# Run with config.yml mount:
# sudo docker run --privileged --name dblab_server --detach \
#   --label dblab-control \
#   --volume /var/run/docker.sock:/var/run/docker.sock \
#   --volume /var/lib/dblab:/var/lib/dblab:rshared \
#   --volume /home/user/configs/config.yml:/home/dblab/configs/config.yml \
#   --publish 3000:3000 \
#   dblab-server
#
# or run with envs options:
# sudo docker run --privileged --name dblab_server --detach \
#   --label dblab-control \
#   --volume /var/run/docker.sock:/var/run/docker.sock \
#   --volume /var/lib/dblab:/var/lib/dblab:rshared \
#   --env VERIFICATION_TOKEN=token \
#   --env MOUNT_DIR=/var/lib/dblab/clones \
#   --env UNIX_SOCKET_DIR=/var/lib/dblab/sockets \
#   --env DOCKER_IMAGE=postgres:12-alpine \
#   --publish 3000:3000 \
#   dblab-server

FROM docker:19

# Install dependecies.
RUN apk update && apk add zfs bash

WORKDIR /home/dblab

COPY ./bin/dblab-server ./bin/dblab-server
COPY ./api ./api
COPY ./web ./web
COPY ./configs ./configs
COPY ./scripts ./scripts

CMD ./bin/dblab-server
