# We need to build in a Linux environment to support C libraries, e.g. RocksDB.
# We use Debian instead of Alpine, so that we can use binary database packages
# instead of spending time compiling them.
# -> There is currently no librocksdb-dev v6.17.x that is necessary by line/gorocksdb.
# So we download rocksdb and build it.

FROM golang:1.17

RUN apt-get -qq update -y && apt-get -qq upgrade -y >/dev/null
RUN apt-get -qq install -y libleveldb-dev make libc-dev libtool >/dev/null
WORKDIR /
RUN wget -O rocksdb-v6.20.3.tar.gz https://github.com/facebook/rocksdb/archive/v6.20.3.tar.gz
RUN tar -zxvf rocksdb-v6.20.3.tar.gz
RUN cd rocksdb-6.20.3 && make -j2 shared_lib && make install-shared
RUN cp /usr/local/lib/librocksdb.so* /usr/lib

# Set up build directory /src/ostracon
ENV OSTRACON_BUILD_OPTIONS badgerdb,boltdb,cleveldb,rocksdb
ENV LIBSODIUM 1
WORKDIR /src/ostracon

# Fetch dependencies separately (for layer caching)
COPY go.mod go.sum ./
RUN go mod download

# Build Ostracon and install into /usr/bin/ostracon
COPY . .
RUN make build && cp build/ostracon /usr/bin/ostracon
COPY test/e2e/docker/entrypoint* /usr/bin/
RUN cd test/e2e && make maverick && cp build/maverick /usr/bin/maverick
RUN cd test/e2e && make node && cp build/node /usr/bin/app

# Set up runtime directory. We don't use a separate runtime image since we need
# e.g. leveldb and rocksdb which are already installed in the build image.
WORKDIR /ostracon
VOLUME /ostracon
ENV OCHOME=/ostracon

EXPOSE 26656 26657 26660 6060
ENTRYPOINT ["/usr/bin/entrypoint"]
CMD ["node"]
STOPSIGNAL SIGTERM
