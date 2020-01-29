# TODO(anatoly): Describe how to run.
# Run with:
# sudo docker run --privileged --name test7 -d -t \
#   -v /var/run/docker.sock:/var/run/docker.sock \
#   -v /var/lib/dblab/test7:/var/lib/dblab/test7:rshared \
#   database-lab

FROM docker:19

# Install dependecies.
RUN apk update && apk add zfs bash

WORKDIR /home/dblab

COPY ./bin/dblab-server ./bin/dblabserver
COPY ./api ./api
COPY ./web ./web
COPY ./configs ./configs
COPY ./scripts ./scripts

CMD ./bin/dblabserver
