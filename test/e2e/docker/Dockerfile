# We need to build in a Linux environment to support C libraries, e.g. RocksDB.
# We use Debian instead of Alpine, so that we can use binary database packages
# instead of spending time compiling them.
FROM golang:1.15

RUN apt-get -qq update -y && apt-get -qq upgrade -y >/dev/null
RUN apt-get -qq install -y libleveldb-dev make libc-dev libtool >/dev/null

ARG SRCDIR=/src/ostracon

# There is currently no librocksdb-dev v6.17.3 or higher that is necessary by line/gorocksdb.
# So we download rocksdb and build it.
# See:
#    - line/gorocksdb: https://github.com/line/gorocksdb/pull/3
#    - line/tm-db: https://github.com/line/tm-db/blob/main/tools/Dockerfile
ARG ROCKSDB_VERSION=6.20.3
ARG ROCKSDB_FILE=rocksdb-v${ROCKSDB_VERSION}.tar.gz
ARG ROCKSDB_DIR=rocksdb-${ROCKSDB_VERSION}
RUN wget -O ${ROCKSDB_FILE} https://github.com/facebook/rocksdb/archive/v${ROCKSDB_VERSION}.tar.gz
RUN tar -zxvf ${ROCKSDB_FILE}
RUN cd ${ROCKSDB_DIR} && make -j2 shared_lib && make install-shared
RUN cp /usr/local/lib/librocksdb.so* /usr/lib
RUN rm -rf ${ROCKSDB_FILE} ${ROCKSDB_DIR}

# Build/Install libsodium separately (for layer caching)
ARG VRF_ROOT=crypto/vrf/internal/vrf
ARG LIBSODIUM_ROOT=${VRF_ROOT}/libsodium
ARG LIBSODIUM_OS=${SRCDIR}/${VRF_ROOT}/sodium/linux_amd64
COPY ${LIBSODIUM_ROOT} ${LIBSODIUM_ROOT}
RUN cd ${LIBSODIUM_ROOT} && \
    ./autogen.sh && \
    ./configure --disable-shared --prefix="${LIBSODIUM_OS}" && \
    make && \
    make install
RUN rm -rf ${LIBSODIUM_ROOT}

ENV OSTRACON_BUILD_OPTIONS badgerdb,boltdb,cleveldb,rocksdb
ENV CGO_LDFLAGS -lrocksdb
ENV LIBSODIUM 1

# Fetch dependencies separately (for layer caching)
COPY go.mod go.sum ${SRCDIR}
RUN cd ${SRCDIR} && go mod download

# Build Ostracon and install into /usr/bin/ostracon
COPY . ${SRCDIR}
COPY test/e2e/docker/entrypoint* /usr/bin/
RUN cd ${SRCDIR} && make build && cp build/ostracon /usr/bin/ostracon
RUN cd ${SRCDIR}/test/e2e && make maverick && cp build/maverick /usr/bin/maverick
RUN cd ${SRCDIR}/test/e2e && make node && cp build/node /usr/bin/app

# Set up runtime directory. We don't use a separate runtime image since we need
# e.g. leveldb and rocksdb which are already installed in the build image.
WORKDIR /ostracon
VOLUME /ostracon
ENV OCHOME=/ostracon

EXPOSE 26656 26657 26660 6060
ENTRYPOINT ["/usr/bin/entrypoint"]
CMD ["node"]
STOPSIGNAL SIGTERM
