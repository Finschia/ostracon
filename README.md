# Ostracon

[![codecov](https://codecov.io/gh/line/ostracon/branch/main/graph/badge.svg?token=JFFuUevpzJ)](https://codecov.io/gh/line/ostracon)

Ostracon is forked from Tendermint Core at 2021-03-15.

**Node**: Requires [Go 1.15+](https://golang.org/dl/)

**Warnings**: Initial development is in progress, but there has not yet been a stable.

# Quick Start
## Docker
**Build Docker Image**
Build the linux binary in ./build
 ```sh
 make build-linux
 ```
(optionally) Build ostracon/localnode image
 ```sh
 make build-docker-localnode
 ```

**Run a testnet**
To start a 4 node testnet run
 ```sh
 make localnet-start
 ```

Before running it, don't forget to cleanup the old files
 ```sh
 rm -rf ./build/node*
 ```

**visit with your browser**
* Node: http://localhost:26657/

## Local
**Build**
 ```
 make build     # go help build
 make install   # go help install
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
