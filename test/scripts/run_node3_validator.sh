#!/bin/sh
go run ../cmd/aggregator/main.go \
    -validatormode \
    -aggregatordb /tmp/celer-rollup-test/node3-aggregator-db \
    -validatordb /tmp/celer-rollup-test/node3-validator-db \
    -mainchainkeystore env/keystore/node3.json \
    -sidechainkeystore env/keystore/node3.json
