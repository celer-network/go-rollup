#!/bin/sh
go run ../cmd/aggregator/main.go \
    -aggregatordb /tmp/celer_rollup_test/aggregator2Db \
    -validatordb /tmp/celer_rollup_test/validator2Db \
    -mainchainkeystore env/keystore/node2.json \
    -sidechainkeystore env/keystore/node2.json
