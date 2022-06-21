# Ostracon

[![codecov](https://codecov.io/gh/line/ostracon/branch/main/graph/badge.svg?token=JFFuUevpzJ)](https://codecov.io/gh/line/ostracon)

Ostracon is forked from Tendermint Core at 2021-03-15.

**Node**: Requires [Go 1.18+](https://golang.org/dl/)

**Warnings**: Initial development is in progress, but there has not yet been a stable.

# Quick Start

## git clone
```shell
git clone https://github.com/line/ostracon.git
# or
git clone git@github.com:line/ostracon.git
```

### git clone with recursive if you want to use libsodium
```shell
git clone --recursive https://github.com/line/ostracon.git
# or
git clone --recursive git@github.com:line/ostracon.git
```

### git submodule if you forget to clone with submodule
```shell
git submodule update --init --recursive
```

## Local Standalone
**Build**
 ```sh
 make build     # go help build
 make install   # go help install
 ```

**Run**
 ```sh
 ostracon init
 ostracon node --proxy_app=kvstore                # Run a node
 ```

Before running it, don't forget to cleanup the old files:
 ```sh
 # Clear the build folder
 rm -rf ~/.ostracon
 ```

**Visit with your browser**
* Node: http://localhost:26657/

## Localnet(4 nodes) with Docker
**Build Docker Image**

(optionally) Build the linux binary for localnode in ./build
 ```sh
 make build-localnode
 ```
(optionally) Build ostracon/localnode image
 ```sh
 make build-localnode-docker
 ```

**Run localnet**

To start 4 nodes
 ```sh
 make localnet-start
 ```

Before running it, don't forget to cleanup the old files
 ```sh
 rm -rf ./build/node*
 ```

**Visit with your browser**
* Node: http://localhost:26657/

## Linux Docker
**Build Docker Image**

Build the linux binary
 ```sh
 make build-linux-docker
 ```

**Run a linux docker node**

To start a linux node
 ```sh
 make standalone-linux-docker
 ```

**Visit with your browser**
* Node: http://localhost:26657/
