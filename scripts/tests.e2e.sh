#!/usr/bin/env bash
set -e

export RUN_E2E="true"
# e.g.,
# ./scripts/tests.e2e.sh 0.4.1-rc1 0.4.1-rc1
if ! [[ "$0" =~ scripts/tests.e2e.sh ]]; then
  echo "must be run from repository root"
  exit 255
fi

DEFAULT_VERSION_1=0.4.10-rc3
DEFAULT_VERSION_2=0.4.10-rc3

if [ $# == 0 ]; then
    VERSION_1=$DEFAULT_VERSION_1
    VERSION_2=$DEFAULT_VERSION_2
else
    VERSION_1=$1
    if [[ -z "${VERSION_1}" ]]; then
      echo "Missing version argument!"
      echo "Usage: ${0} [VERSION_1] [VERSION_2]" >> /dev/stderr
      exit 255
    fi
    VERSION_2=$2
    if [[ -z "${VERSION_2}" ]]; then
      echo "Missing version argument!"
      echo "Usage: ${0} [VERSION_1] [VERSION_2]" >> /dev/stderr
      exit 255
    fi
fi

echo "Running e2e tests with:"
echo VERSION_1: ${VERSION_1}
echo VERSION_2: ${VERSION_2}

TEMP_PATH=/tmp

if [ ! -f ${TEMP_PATH}/camino-node-v${VERSION_1}/camino-node ]
then
    ############################
    # download camino-node
    # https://github.com/chain4travel/camino-node/releases
    GOARCH=$(go env GOARCH)
    GOOS=$(go env GOOS)
    DOWNLOAD_URL=https://github.com/chain4travel/camino-node/releases/download/v${VERSION_1}/camino-node-linux-${GOARCH}-v${VERSION_1}.tar.gz
    DOWNLOAD_PATH=${TEMP_PATH}/camino-node.tar.gz
    if [[ ${GOOS} == "darwin" ]]; then
      DOWNLOAD_URL=https://github.com/chain4travel/camino-node/releases/download/v${VERSION_1}/camino-node-macos-v${VERSION_1}.zip
      DOWNLOAD_PATH=${TEMP_PATH}/camino-node.zip
    fi

    rm -rf ${TEMP_PATH}/camino-node-v${VERSION_1}
    rm -rf ${TEMP_PATH}/camino-node-build
    rm -f ${DOWNLOAD_PATH}

    echo "downloading camino-node ${VERSION_1} at ${DOWNLOAD_URL}"
    curl -L ${DOWNLOAD_URL} -o ${DOWNLOAD_PATH}

    echo "extracting downloaded camino-node"
    if [[ ${GOOS} == "linux" ]]; then
      tar xzvf ${DOWNLOAD_PATH} -C ${TEMP_PATH}
    elif [[ ${GOOS} == "darwin" ]]; then
      unzip ${DOWNLOAD_PATH} -d ${TEMP_PATH}/camino-node-build
      mv ${TEMP_PATH}/camino-node-build/build ${TEMP_PATH}/camino-node-v${VERSION_1}
    fi
fi

if [ ! -f ${TEMP_PATH}/camino-node-v${VERSION_2}/camino-node ]
then
    ############################
    # download camino-node
    # https://github.com/chain4travel/camino-node/releases
    GOARCH=$(go env GOARCH)
    GOOS=$(go env GOOS)
    DOWNLOAD_URL=https://github.com/chain4travel/camino-node/releases/download/v${VERSION_2}/camino-node-linux-${GOARCH}-v${VERSION_2}.tar.gz
    DOWNLOAD_PATH=${TEMP_PATH}/camino-node.tar.gz
    if [[ ${GOOS} == "darwin" ]]; then
      DOWNLOAD_URL=https://github.com/chain4travel/camino-node/releases/download/v${VERSION_2}/camino-node-macos-v${VERSION_2}.zip
      DOWNLOAD_PATH=${TEMP_PATH}/camino-node.zip
    fi

    rm -rf ${TEMP_PATH}/camino-node-v${VERSION_2}
    rm -rf ${TEMP_PATH}/camino-node-build
    rm -f ${DOWNLOAD_PATH}

    echo "downloading camino-node ${VERSION_2} at ${DOWNLOAD_URL}"
    curl -L ${DOWNLOAD_URL} -o ${DOWNLOAD_PATH}

    echo "extracting downloaded camino-node"
    if [[ ${GOOS} == "linux" ]]; then
      tar xzvf ${DOWNLOAD_PATH} -C ${TEMP_PATH}
    elif [[ ${GOOS} == "darwin" ]]; then
      unzip ${DOWNLOAD_PATH} -d ${TEMP_PATH}/camino-node-build
      mv ${TEMP_PATH}/camino-node-build/build ${TEMP_PATH}/camino-node-v${VERSION_2}
    fi
fi

############################
echo "building runner"
./scripts/build.sh

echo "building e2e.test"
# to install the ginkgo binary (required for test build and run)
go install -v github.com/onsi/ginkgo/v2/ginkgo@v2.1.3
ACK_GINKGO_RC=true ginkgo build ./tests/e2e
./tests/e2e/e2e.test --help

snapshots_dir=${TEMP_PATH}/camino-network-runner-snapshots-e2e/
rm -rf $snapshots_dir

killall camino-network-runner || true

echo "launch local test cluster in the background"
bin/camino-network-runner \
server \
--log-level debug \
--port=":8080" \
--snapshots-dir=$snapshots_dir \
--grpc-gateway-port=":8081" &
PID=${!}

function cleanup()
{
  echo "shutting down network runner"
  kill ${PID}
}
trap cleanup EXIT

echo "running e2e tests"
./tests/e2e/e2e.test \
--ginkgo.v \
--ginkgo.fail-fast \
--log-level debug \
--grpc-endpoint="0.0.0.0:8080" \
--grpc-gateway-endpoint="0.0.0.0:8081" \
--camino-node-path-1=${TEMP_PATH}/camino-node-v${VERSION_1}/camino-node \
--camino-node-path-2=${TEMP_PATH}/camino-node-v${VERSION_2}/camino-node

kill ${PID}
echo "ALL SUCCESS!"
