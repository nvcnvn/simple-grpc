#!/bin/sh
# for regenerate protoc files when need
protoc -I=./grpc -I=$GOPATH/src -I=$GOPATH/src/github.com/gogo/protobuf/protobuf --gogoslick_out=plugins=grpc:./grpc ./grpc/service.proto