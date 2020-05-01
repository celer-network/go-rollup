#!/bin/sh
dataDir="${1:-/tmp/celer-rollup-test/sidechaindata}"
keystore="${2:-env/keystore}"
geth --datadir "$dataDir" init env/sidechain_genesis.json
geth --networkid 883 --cache 256 --syncmode full --nousb --nodiscover --maxpeers 0 \
    --mine --allow-insecure-unlock --unlock ba756d65a1a03f07d205749f35e2406e4a8522ad \
    --etherbase ba756d65a1a03f07d205749f35e2406e4a8522ad \
    --password env/empty_password.txt \
    --netrestrict 127.0.0.1/8 --datadir "$dataDir" \
    --keystore "$keystore" --targetgaslimit 10000000 \
    --port 30304 \
    --rpc --rpcport 8547 --rpccorsdomain '*' \
    --rpcapi admin,debug,eth,miner,net,personal,txpool,web3 \
    --ws --wsport 8548 --wsorigins '*' \
    --wsapi admin,debug,eth,miner,net,personal,txpool,web3
