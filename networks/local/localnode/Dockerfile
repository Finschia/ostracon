FROM golang:1.18-alpine

RUN apk update && \
    apk upgrade && \
    apk add --no-cache git make gcc libc-dev build-base curl jq bash file gmp-dev clang libtool autoconf automake

VOLUME [ "/ostracon" ]
WORKDIR /ostracon
EXPOSE 26656 26657
ENTRYPOINT ["/usr/bin/wrapper.sh"]
CMD ["node", "--proxy_app", "kvstore"]
STOPSIGNAL SIGTERM

COPY wrapper.sh /usr/bin/wrapper.sh
COPY config-template.toml /etc/ostracon/config-template.toml
