FROM golang:1.14-alpine
MAINTAINER Greg Szabo <greg@tendermint.com>

RUN apk update && \
    apk upgrade && \
    apk add --update --no-cache git make gcc libc-dev build-base curl jq file \
                                gmp-dev clang
VOLUME [ "/tendermint" ]
WORKDIR /tendermint
EXPOSE 26656 26657
ENTRYPOINT ["/usr/bin/wrapper.sh"]
CMD ["node", "--proxy_app", "kvstore"]
STOPSIGNAL SIGTERM

COPY wrapper.sh /usr/bin/wrapper.sh
COPY config-template.toml /etc/tendermint/config-template.toml
