# We need to build in a Linux environment to support C libraries, e.g. RocksDB.
# We use Debian instead of Alpine, so that we can use binary database packages
# instead of spending time compiling them.
FROM golang:1.20

RUN apt-get -qq update -y && apt-get -qq upgrade -y >/dev/null
RUN apt-get -qq install -y libleveldb-dev make libc-dev libtool >/dev/null

# RocksDB 6.24.2+ is required to build with tm-db 0.6.7 (but RocksDB 7.x is not yet supported).
# librocksdb-dev installed by apt with golang:1.20 is 7.8.3-2, so we have to build it from the latest 6.x sources.
ARG ROCKSDB_VERSION=6.29.5
ARG ROCKSDB_FILE=rocksdb-v${ROCKSDB_VERSION}.tar.gz
ARG ROCKSDB_DIR=rocksdb-${ROCKSDB_VERSION}
RUN wget -O ${ROCKSDB_FILE} https://github.com/facebook/rocksdb/archive/v${ROCKSDB_VERSION}.tar.gz
RUN tar -zxvf ${ROCKSDB_FILE}
RUN cd ${ROCKSDB_DIR} && DISABLE_WARNING_AS_ERROR=1 make -j2 shared_lib && make install-shared
RUN cp /usr/local/lib/librocksdb.so* /usr/lib
RUN rm -rf ${ROCKSDB_FILE} ${ROCKSDB_DIR}

ENV OSTRACON_BUILD_OPTIONS badgerdb,boltdb,cleveldb,rocksdb
ENV CGO_LDFLAGS -lrocksdb
WORKDIR /src/ostracon

# Fetch dependencies separately (for layer caching)
COPY go.mod go.sum ./
RUN go mod download

# Build Ostracon and install into /usr/bin/ostracon
COPY . .
RUN make build && cp build/ostracon /usr/bin/ostracon
COPY test/e2e/docker/entrypoint* /usr/bin/

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
