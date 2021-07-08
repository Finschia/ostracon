#!/usr/bin/env sh

##
## Input parameters
##
BINARY=/ostracon/${BINARY:-ostracon}
ID=${ID:-0}
LOG=${LOG:-ostracon.log}

##
## Assert linux binary
##
if ! [ -f "${BINARY}" ]; then
	echo "The binary $(basename "${BINARY}") cannot be found. Please add the binary to the shared folder. Please use the BINARY environment variable if the name of the binary is not 'ostracon' E.g.: -e BINARY=ostracon_my_test_version"
	exit 1
elif ! [ -x "${BINARY}" ]; then
	echo "The binary $(basename "${BINARY}") is not executable."
	exit 1
fi
BINARY_CHECK="$(file "$BINARY" | grep 'ELF 64-bit LSB executable, x86-64')"
if [ -z "${BINARY_CHECK}" ]; then
	echo "Binary needs to be OS linux, ARCH amd64"
	exit 1
fi

##
## Run binary with all parameters
##
export OCHOME="/ostracon/node${ID}"

if [ -d "`dirname ${OCHOME}/${LOG}`" ]; then
  "$BINARY" "$@" | tee "${OCHOME}/${LOG}"
else
  "$BINARY" "$@"
fi

chmod 777 -R /ostracon

