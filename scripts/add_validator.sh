#!/bin/sh
# This script generate new validator and add new validator to the chain
# warning:
#   1. if you input p2p port, abci, api and promethus port is set automatelly
#   2. if you set multi validator in one device, the `allow_duplicate_ip` of all validator's config should be true.
#
# Process
# 1. generate new validator with keys
# 2. get genesis.json from persistent_seed
# 3. change config.toml
#   - set port
#	- open prometheus port
#	- allow duplicate ip to perform multi in node same device.
# 4. send tx whitch add new validator
# 5. start new validator

if [ "$#" -ne 3 ]; then
	echo "generate new validator script"
	echo ""
	echo "usage: $0 [path] [p2p port] [persistent_peer]"
	echo ""
	echo "Example"
	echo "$0 ./single1 26677 c8029828535291adbd1782e8aad41c121019e163@localhost:26656"
	exit 2
fi

TENDERMINT="./build/tendermint"
# HOME_PATH="./single3"
HOME_PATH=$1
# P2P_PORT=26677
P2P_PORT=$2
# PERSISTENT_PEERS='c8029828535291adbd1782e8aad41c121019e163@localhost:26656'
PERSISTENT_PEERS=$3

ORIGIN_URL=$(echo $PERSISTENT_PEERS | cut -d '@' -f 2 | cut -d ':' -f 1 )":26657"
ABCI_PORT=$(( $P2P_PORT + 1 ))
API_PORT=$(( $P2P_PORT + 2 ))
PROMETHEUS_PORT=$(( $P2P_PORT + 3))

# generate new node with private key
${TENDERMINT} init --home=${HOME_PATH}
# show node id
echo "show node id: "
$TENDERMINT show_node_id --home=${HOME_PATH}

# cat $HOME_PATH/config/genesis.json
# get genesis and save genesis file
curl -s "$ORIGIN_URL/genesis" | jq ".result.genesis" > $HOME_PATH/config/genesis.json

# =======================
# change config ports
# proxy(abci)
# proxy_app = "tcp://127.0.0.1:26658"
# [rpc] - api
# laddr = "tcp://127.0.0.1:26657"
# [p2p]
# laddr = "tcp://0.0.0.0:26656"
# allow_duplicate_ip = true
# [instrumentation]
# prometheus = false
# prometheus_listen_addr = ":26660"
# add persistent_peers
# persistent_peers = "ID@localhost:26656"
sed -i'.back' -e "s/26658/$ABCI_PORT/" \
			  -e "s/26657/${API_PORT}/" \
			  -e "s/26656/${P2P_PORT}/" \
			  -e "/prometheus = false/s/false/true/" \
			  -e "/allow_duplicate_ip = false/s/false/true/" \
			  -e "/addr_book_strict = true/s/true/false/" \
			  -e "s/26660/${PROMETHEUS_PORT}/" \
			  $HOME_PATH/config/config.toml

sed -i'.back' -e "s/persistent_peers = \"\"/persistent_peers = \"${PERSISTENT_PEERS}\"/" \
        $HOME_PATH/config/config.toml

# show transaction for adding new validator
# get public key
echo "Get public address of new validator node"
PUBLIC_KEY=$(sudo bash -c "cat ${HOME_PATH}/config/priv_validator_key.json" | jq -r ".pub_key.value")

# send tx whitch add new validator
curl -G --data-urlencode 'tx="val:'${PUBLIC_KEY}'!10"' http://localhost:26657/broadcast_tx_sync

# run new validator
$TENDERMINT node --proxy_app=persistent_kvstore --home=$HOME_PATH
