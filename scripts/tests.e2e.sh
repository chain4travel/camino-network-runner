#!/usr/bin/env bash
set -e

export RUN_E2E="true"
# e.g.,
# ./scripts/tests.e2e.sh /tmp/caminogo
if ! [[ "$0" =~ scripts/tests.e2e.sh ]]; then
  echo "must be run from repository root"
  exit 255
fi

CAMINO_NETWORK_RUNNER_PATH=$( cd "$( dirname "${BASH_SOURCE[0]}" )"; cd .. && pwd )

CAMINO_NODE_PATH="${1-}"
if [[ -z "${CAMINO_NODE_PATH}" ]]; then
  echo "Missing CAMINO_NODE_PATH argument!"
  echo "Usage: ${0} [CAMINO_NODE_PATH]" >> /dev/stderr
  exit 255
fi

echo "Running e2e tests:"

TEMP_PATH=/tmp

mkdir $TEMP_PATH/camino-node-1 -p
cp -r $CAMINO_NODE_PATH $TEMP_PATH/camino-node-1/camino-node

DEFAULT_SUBNET_EVM_VERSION=0.5.1
SUBNET_EVM_VERSION=$DEFAULT_SUBNET_EVM_VERSION

if [ ! -f ${TEMP_PATH}/subnet-evm-v${SUBNET_EVM_VERSION}/subnet-evm ]
then
    ############################
    # download subnet-evm 
    # https://github.com/ava-labs/subnet-evm/releases
    GOARCH=$(go env GOARCH)
    GOOS=$(go env GOOS)
    DOWNLOAD_URL=https://github.com/ava-labs/subnet-evm/releases/download/v${SUBNET_EVM_VERSION}/subnet-evm_${SUBNET_EVM_VERSION}_linux_${GOARCH}.tar.gz
    DOWNLOAD_PATH=${TEMP_PATH}/subnet-evm.tar.gz
    if [[ ${GOOS} == "darwin" ]]; then
      DOWNLOAD_URL=https://github.com/ava-labs/subnet-evm/releases/download/v${SUBNET_EVM_VERSION}/subnet-evm_${SUBNET_EVM_VERSION}_darwin_${GOARCH}.tar.gz
    fi

    rm -rf ${TEMP_PATH}/subnet-evm-v${SUBNET_EVM_VERSION}
    rm -f ${DOWNLOAD_PATH}

    echo "downloading subnet-evm ${SUBNET_EVM_VERSION} at ${DOWNLOAD_URL}"
    curl -L ${DOWNLOAD_URL} -o ${DOWNLOAD_PATH}

    echo "extracting downloaded subnet-evm"
    mkdir ${TEMP_PATH}/subnet-evm-v${SUBNET_EVM_VERSION}
    tar xzvf ${DOWNLOAD_PATH} -C ${TEMP_PATH}/subnet-evm-v${SUBNET_EVM_VERSION}
    # NOTE: We are copying the subnet-evm binary here to a plugin hardcoded as srEXiWaHuhNyGwPUi444Tu47ZEDwxTWrbQiuD7FmgSAQ6X7Dy which corresponds to the VM name `subnetevm` used as such in the test
    mkdir -p ${TEMP_PATH}/camino-node-1/plugins/
    cp ${TEMP_PATH}/subnet-evm-v${SUBNET_EVM_VERSION}/subnet-evm ${TEMP_PATH}/camino-node-1/plugins/srEXiWaHuhNyGwPUi444Tu47ZEDwxTWrbQiuD7FmgSAQ6X7Dy
    find ${TEMP_PATH}/subnet-evm-v${SUBNET_EVM_VERSION}/subnet-evm
fi

############################
echo "building runner"
"$CAMINO_NETWORK_RUNNER_PATH/scripts/build.sh"

echo "building e2e.test"
# to install the ginkgo binary (required for test build and run)
go install -v github.com/onsi/ginkgo/v2/ginkgo@v2.1.3
ACK_GINKGO_RC=true ginkgo build $CAMINO_NETWORK_RUNNER_PATH/tests/e2e
# $CAMINO_NETWORK_RUNNER_PATH/tests/e2e/e2e.test --help

snapshots_dir=${TEMP_PATH}/camino-network-runner-snapshots-e2e/
rm -rf $snapshots_dir

killall camino-network-runner || true

echo "launch local test cluster in the background"
"$CAMINO_NETWORK_RUNNER_PATH/bin/camino-network-runner" \
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
"$CAMINO_NETWORK_RUNNER_PATH/tests/e2e/e2e.test" \
--ginkgo.v \
--ginkgo.fail-fast \
--log-level debug \
--grpc-endpoint="0.0.0.0:8080" \
--grpc-gateway-endpoint="0.0.0.0:8081" \
--camino-node-path-1=$TEMP_PATH/camino-node-1/camino-node \
--subnet-evm-path=${TEMP_PATH}/subnet-evm-v${SUBNET_EVM_VERSION}/subnet-evm
# --camino-node-path-2=$TEMP_PATH/camino-node-2/camino-node \
# camino-node-path-2 arg can be used to specify last compatible version to test its compatibility // TODO @evlekht verify it

kill ${PID}
echo "ALL SUCCESS!"
