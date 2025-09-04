#!/bin/bash
CURDIR=$(cd $(dirname $0); pwd)
BinaryName=innospark.core_api
echo "$CURDIR/bin/${BinaryName}"
exec $CURDIR/bin/${BinaryName}