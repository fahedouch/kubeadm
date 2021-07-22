#!/bin/bash

set -o nounset
set -o pipefail
set -o errexit

SEP="###############################################################################"
echo $SEP
echo -e "* running `basename "$0"`..."
echo $SEP

set -x

# cleanup go.* and tmp dir on exit
cleanup(){
    rm -rf "${TMP:?}" go.mod go.sum
}
trap 'cleanup' EXIT
# create portable tempdir
TMP="$(mktemp -d)"

# install curl if missing
if ! `curl --version > /dev/null`; then
	apt-get update
	apt-get install -y curl
fi

# install go if missing
if ! `go version > /dev/null`; then
	curl -sSL https://dl.google.com/go/go1.16.linux-amd64.tar.gz -o "/${TMP}/go.tar.gz"
	tar -C /usr/local -xzf "/${TMP}/go.tar.gz"
	export PATH="$PATH":/usr/local/go/bin
	rm "/${TMP}/go.tar.gz"
fi

# api-machinery requires gcc
#   go: extracting k8s.io/apimachinery v0.17.3
#   runtime/cgo
#   exec: "gcc": executable file not found in $PATH
if ! `gcc -v > /dev/null`; then
	apt-get install -y gcc
fi

LPATH=`dirname "$0"`
cd "$LPATH"

# use go modules. this forces using the latest k8s.io/apimachinery package.
go mod init verify-manifest-lists

# add module requirements and sums (required in go 1.16)
go mod tidy

# run unit tests
go test -v ./verify_manifest_lists.go ./verify_manifest_lists_test.go

# run main test
go run ./verify_manifest_lists.go
