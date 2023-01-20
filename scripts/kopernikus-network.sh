#!/usr/bin/env bash

set -o errexit
set -o pipefail
set -e

if ! [[ "$0" =~ scripts/kopernikus-network.sh ]]; then
  echo "must be run from repository root"
  exit 1
fi

COMMAND='start'
CAMINO_EXEC_PATH='../camino-node/build/camino-node'
if [ $# == 1 ]; then
    COMMAND=$1
else
  if [ $# == 2 ]; then
    COMMAND=$1
    CAMINO_EXEC_PATH=$2
  fi
fi

echo CAMINO_EXEC_PATH: ${CAMINO_EXEC_PATH}
CAMINO_ROOT_PATH="${CAMINO_EXEC_PATH%/*/*}"

if [ "$COMMAND" = 'start' ]; then
  echo 'Starting network runner with kopernikus configuration...'
  ./bin/camino-network-runner server &>/dev/null &
  curl -X POST -k localhost:8081/v1/control/start -d '{"execPath":"'${CAMINO_EXEC_PATH}'","numNodes":2,"logLevel":"INFO","globalNodeConfig":"{\"http-host\":\"0.0.0.0\",\"network-id\":\"kopernikus\",\"genesis\":\"'${CAMINO_ROOT_PATH}'/dependencies/caminoethvm/caminogo/genesis/genesis_kopernikus.json\"}"}'
else
  echo 'Stopping network runner...'
  curl -X POST -k localhost:8081/v1/control/stop
fi
