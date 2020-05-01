#!/bin/sh
go run ../tools/revertreason/main.go -ethrpc https://ropsten.infura.io/v3/b64e62bc284840a491fa39dedf88b6af -tx $1
