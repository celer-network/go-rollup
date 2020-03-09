#!/bin/sh
dataDir="${1:-/tmp/celer_rollup_test/mainchaindata}"
keystore="${2:-env/keystore}"
geth --datadir "$dataDir" init env/mainchain_genesis.json
geth --networkid 883 --cache 256 --syncmode full --nousb --nodiscover --maxpeers 0 \
    --mine --allow-insecure-unlock --unlock b5bb8b7f6f1883e0c01ffb8697024532e6f3238c \
    --etherbase b5bb8b7f6f1883e0c01ffb8697024532e6f3238c \
    --password env/empty_password.txt \
    --netrestrict 127.0.0.1/8 --datadir "$dataDir" \
    --keystore "$keystore" --targetgaslimit 10000000 \
    --port 30303 \
    --rpc --rpcport 8545 --rpccorsdomain '*' \
    --rpcapi admin,debug,eth,miner,net,personal,txpool,web3 \
    --ws --wsport 8546 --wsorigins '*' \
    --wsapi admin,debug,eth,miner,net,personal,txpool,web3
