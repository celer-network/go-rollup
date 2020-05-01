#!/bin/sh
go run ../cmd/aggregator/main.go \
    -fraudtransfer \
    -aggregatordb /tmp/celer-rollup-test/node2-aggregator-db \
    -validatordb /tmp/celer-rollup-test/node2-validator-db \
    -mainchainkeystore env/keystore/node2.json \
    -sidechainkeystore env/keystore/node2.json
