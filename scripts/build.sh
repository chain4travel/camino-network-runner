#!/bin/bash

# Run with ./scripts/build.sh <optional_build_location>
echo "Building camino network runner..."

if [[ "$OSTYPE" != "linux"* ]]; then
    echo "camino-network-runner can be built on linux only. Current OS is $OSTYPE"
    exit 0
fi

if ! [[ "$0" =~ scripts/build.sh ]]; then
  echo "must be run from repository root"
  exit 1
fi

CAMINO_NETWORK_RUNNER_PATH=$( cd "$( dirname "${BASH_SOURCE[0]}" )"; cd .. && pwd )

VERSION=`cat $CAMINO_NETWORK_RUNNER_PATH/VERSION`

if [ $# -eq 0 ] ; then
    OUTPUT="bin"
else
    OUTPUT=$1
fi

cd $CAMINO_NETWORK_RUNNER_PATH
go build -v -ldflags="-X 'github.com/ava-labs/avalanche-network-runner/cmd.Version=$VERSION'" -o $OUTPUT/camino-network-runner 

# Exit build successfully if the binaries are created
if [[ -f "$CAMINO_NETWORK_RUNNER_PATH/$OUTPUT/camino-network-runner" ]]; then
    echo "Build Successful"
    exit 0
else
    echo "Build failure" >&2
    exit 1
fi