#!/bin/sh
go run ../cmd/aggregator/main.go \
    -aggregatordb /tmp/celer-rollup-test/node1-aggregator-db \
    -validatordb /tmp/celer-rollup-test/node1-validator-db \
    -mainchainkeystore env/keystore/node1.json \
    -sidechainkeystore env/keystore/node1.json
