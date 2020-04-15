#!/bin/sh
go run ../tools/revertreason/main.go -ethrpc http://127.0.0.1:8545 -tx $1
