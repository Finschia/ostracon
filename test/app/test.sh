#! /bin/bash
set -ex

#- kvstore over grpc, curl
#- counter over grpc, curl
#- counter over grpc, grpc

# TODO: install everything

export PATH="$GOBIN:$PATH"
export OSTHOME=$HOME/.ostracon

function kvstore_over_grpc(){
    rm -rf $OSTHOME
    ostracon init
    echo "Starting kvstore_over_grpc"
    abci-cli kvstore --abci grpc > /dev/null &
    pid_kvstore=$!
    ostracon node --abci grpc > ostracon.log &
    pid_ostracon=$!
    sleep 5

    echo "running test"
    bash test/app/kvstore_test.sh "KVStore over GRPC"

    kill -9 $pid_kvstore $pid_ostracon
}

# start ostraconi first
function kvstore_over_grpc_reorder(){
    rm -rf $OSTHOME
    ostracon init
    echo "Starting kvstore_over_grpc_reorder (ie. start ostracon first)"
    ostracon node --abci grpc > ostracon.log &
    pid_ostracon=$!
    sleep 2
    abci-cli kvstore --abci grpc > /dev/null &
    pid_kvstore=$!
    sleep 5

    echo "running test"
    bash test/app/kvstore_test.sh "KVStore over GRPC"

    kill -9 $pid_kvstore $pid_ostracon
}

function counter_over_grpc() {
    rm -rf $OSTHOME
    ostracon init
    echo "Starting counter_over_grpc"
    abci-cli counter --serial --abci grpc > /dev/null &
    pid_counter=$!
    ostracon node --abci grpc > ostracon.log &
    pid_ostracon=$!
    sleep 5

    echo "running test"
    bash test/app/counter_test.sh "Counter over GRPC"

    kill -9 $pid_counter $pid_ostracon
}

function counter_over_grpc_grpc() {
    rm -rf $OSTHOME
    ostracon init
    echo "Starting counter_over_grpc_grpc (ie. with grpc broadcast_tx)"
    abci-cli counter --serial --abci grpc > /dev/null &
    pid_counter=$!
    sleep 1
    GRPC_PORT=36656
    ostracon node --abci grpc --rpc.grpc_laddr tcp://localhost:$GRPC_PORT > ostracon.log &
    pid_ostracon=$!
    sleep 5

    echo "running test"
    GRPC_BROADCAST_TX=true bash test/app/counter_test.sh "Counter over GRPC via GRPC BroadcastTx"

    kill -9 $pid_counter $pid_ostracon
}

case "$1" in
    "kvstore_over_grpc")
    kvstore_over_grpc
    ;;
    "kvstore_over_grpc_reorder")
    kvstore_over_grpc_reorder
    ;;
    "counter_over_grpc")
    counter_over_grpc
    ;;
    "counter_over_grpc_grpc")
    counter_over_grpc_grpc
    ;;
*)
    echo "Running all"
    kvstore_over_grpc
    echo ""
    kvstore_over_grpc_reorder
    echo ""
    counter_over_grpc
    echo ""
    counter_over_grpc_grpc
esac

