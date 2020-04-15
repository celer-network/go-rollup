#!/bin/sh
go run ../cmd/aggregator/main.go \
    -fraudtransfer \
    -aggregatordb /tmp/celer_rollup_test/aggregator1Db \
    -validatordb /tmp/celer_rollup_test/validator1Db \
    -mainchainkeystore env/keystore/node1.json \
    -sidechainkeystore env/keystore/node1.json
