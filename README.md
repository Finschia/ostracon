# Ostracon

[![codecov](https://codecov.io/gh/line/ostracon/branch/main/graph/badge.svg?token=JFFuUevpzJ)](https://codecov.io/gh/line/ostracon)

This repository hosts `LINE Blockchain`.

**Node**: Requires [Go 1.15+](https://golang.org/dl/)

**Warnings**: Initial development is in progress, but there has not yet been a stable.

# Quick Start
## Docker
**Build Docker Image**
Build the linux binary in ./build
```sh
make build-linux              # build docker image
```
or
(optionally) Build ostracon/localnode image
```sh
make build-docker-localnode
```

**Run a testnet**
To start a 4 node testnet run:
```sh
make localnet-start
```

Before running it, don't forget to cleanup the old files:
```sh
# Clear the build folder
rm -rf ./build/node*
```

**visit with your browser**
* Node: http://localhost:26657/

## Local
**Build**
```
make build
make install
```

**Run**
```
ostracon init
ostracon node --proxy_app=kvstore                # Run a node
```

Before running it, don't forget to cleanup the old files:
```sh
# Clear the build folder
rm -rf ~/.ostracon
```

**visit with your browser**
* Node: http://localhost:26657/
