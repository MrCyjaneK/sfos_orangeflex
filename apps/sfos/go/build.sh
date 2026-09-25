#!/bin/sh
# Native c-shared library. Invoked from qmake during rpmbuild (Go comes from
# .sfosbuild/image/10-golang.sh).
set -eu
cd "$(dirname "$0")"
mkdir -p ../lib
export CGO_ENABLED=1
export GOFLAGS="${GOFLAGS:--buildvcs=false}"
go build -buildmode=c-shared -trimpath -o ../lib/liborangeflex.so .
