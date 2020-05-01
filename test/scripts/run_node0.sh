#!/bin/sh
go run ../cmd/aggregator/main.go \
    -aggregatordb /tmp/celer-rollup-test/node0-aggregator-db \
    -validatordb /tmp/celer-rollup-test/node0-validator-db \
    -mainchainkeystore env/keystore/node0.json \
    -sidechainkeystore env/keystore/node0.json
