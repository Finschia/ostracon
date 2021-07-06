# stage 1 Generate Ostracon Binary
FROM golang:1.15-alpine as builder
RUN apk update && \
    apk upgrade && \
    apk add --no-cache git make gcc libc-dev build-base curl jq bash file gmp-dev clang
COPY / /ostracon
WORKDIR /ostracon
RUN make build-linux

# stage 2
FROM golang:1.15-alpine
LABEL maintainer="hello@blockchain.line.me"

# Ostracon will be looking for the genesis file in /ostracon/config/genesis.json
# (unless you change `genesis_file` in config.toml). You can put your config.toml and
# private validator file into /ostracon/config.
#
# The /ostracon/data dir is used by ostracon to store state.
ENV TMHOME /ostracon

# OS environment setup
# Set user right away for determinism, create directory for persistence and give our user ownership
# jq and curl used for extracting `pub_key` from private validator while
# deploying ostracon with Kubernetes. It is nice to have bash so the users
# could execute bash commands.
RUN apk update && \
    apk upgrade && \
    apk add --no-cache git make gcc libc-dev build-base curl jq bash file gmp-dev clang && \
    addgroup ostracon && \
    adduser -S -G ostracon ostracon -h "$TMHOME"

# Run the container with ostracon by default. (UID=100, GID=1000)
USER ostracon

WORKDIR $TMHOME

# p2p, rpc and prometheus port
EXPOSE 26656 26657 26660

STOPSIGNAL SIGTERM

COPY --from=builder /ostracon/build/ostracon /usr/bin/ostracon

# You can overwrite these before the first run to influence
# config.json and genesis.json. Additionally, you can override
# CMD to add parameters to `ostracon node`.
ENV PROXY_APP=kvstore MONIKER=dockernode CHAIN_ID=dockerchain

COPY ./DOCKER/docker-entrypoint.sh /usr/local/bin/

ENTRYPOINT ["docker-entrypoint.sh"]
CMD ["node"]

# Expose the data directory as a volume since there's mutable state in there
VOLUME [ "$TMHOME" ]

