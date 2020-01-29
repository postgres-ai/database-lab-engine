# TODO(anatoly): Describe how to run.
# Run with:
# sudo docker run --privileged --name test7 -d -t \
#   -v /var/run/docker.sock:/var/run/docker.sock \
#   -v /var/lib/dblab/test7:/var/lib/dblab/test7:rshared \
#   database-lab

FROM docker:19

# Install ZFS.
RUN apk update && apk add zfs

COPY ./bin/dblab-server ./bin
COPY ./api .
COPY ./web .
COPY ./configs .
COPY ./scripts .

CMD ./bin/dblab-server
