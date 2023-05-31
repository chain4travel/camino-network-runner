#!/bin/bash

# Run with ./scripts/build.sh <optional_build_location>

if ! [[ "$0" =~ scripts/build.sh ]]; then
  echo "must be run from repository root"
  exit 1
fi

VERSION=`cat VERSION`

if [ $# -eq 0 ] ; then
    OUTPUT="bin"
else
    OUTPUT=$1
fi

go build -v -ldflags="-X 'github.com/ava-labs/avalanche-network-runner/cmd.Version=$VERSION'" -o $OUTPUT/camino-network-runner
