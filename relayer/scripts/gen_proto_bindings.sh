#!/bin/sh
protoc -Iproto --go_out=plugins=grpc:. proto/relayer.proto --go_opt=paths=source_relative
