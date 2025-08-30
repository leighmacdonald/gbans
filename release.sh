#!/bin/env bash
set -e

if [ -z "$1" ]; then
  echo "Please set the new version as the first parameter"
  exit 1
fi

NEXT_VERSION=$1

CURRENT_VERSION=`sed -n 's/^VERSION=v\(.*\)/\1/p' < Makefile`

if [ -z "$CURRENT_VERSION" ]; then
  echo "Failed to find current version"
  exit 1
fi

echo "Setting version from $CURRENT_VERSION -> $NEXT_VERSION"

sed -i "s/"${CURRENT_VERSION}"/"${NEXT_VERSION}"/g" sourcemod/scripting/gbans/globals.sp
sed -i "s/v${CURRENT_VERSION}"/v${NEXT_VERSION}"/g" frontend/package.json
sed -i "s/v${CURRENT_VERSION}/v${NEXT_VERSION}/g" Makefile

exit 0
