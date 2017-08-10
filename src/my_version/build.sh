#!/bin/bash
set -x

export GOPATH="`pwd`/../.."

GIT_SHA=`git rev-parse --short HEAD`
VERSION="0.1"

go build -ldflags " -X main.Version=${VERSION} -X my_version/version.GitSHA=${GIT_SHA}" main.go
