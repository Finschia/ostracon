FROM golang:1.22

# Grab deps (jq, hexdump, xxd, killall)
RUN apt-get update && \
  apt-get install -y --no-install-recommends \
  jq bsdmainutils vim-common psmisc netcat curl

# Setup ostracon repo
ENV REPO $GOPATH/src/github.com/Finschia/ostracon
ENV GOBIN $GOPATH/bin
WORKDIR $REPO

# Copy in the code
# TODO: rewrite to only copy Makefile & other files?
COPY . $REPO

# Install the vendored dependencies
# docker caching prevents reinstall on code change!
# XXX Should remove "make tools": https://github.com/Finschia/ostracon/commit/c6e0d20d4bf062921fcc1eb5b2399447a7d2226e#diff-76ed074a9305c04054cdebb9e9aad2d818052b07091de1f20cad0bbac34ffb52
#RUN make tools

# install ABCI CLI
RUN make install_abci

# install ostracon
RUN make install

RUN ostracon testnet \
  --config $REPO/test/docker/config-template.toml \
  --node-dir-prefix="mach" \
  --v=4 \
  --populate-persistent-peers=false \
  --o=$REPO/test/p2p/data

# Now copy in the code
# NOTE: this will overwrite whatever is in vendor/
COPY . $REPO

# expose the volume for debugging
VOLUME $REPO

EXPOSE 26656
EXPOSE 26657
