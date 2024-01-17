#!/bin/env bash
set -e

trap 'echo "Received non-zero exit code, bailing"' EXIT

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

make fmt
make check
make docker_test

echo "Setting version from $CURRENT_VERSION -> $NEXT_VERSION"

sed -i "s/"${CURRENT_VERSION}"/"${NEXT_VERSION}"/g" sourcemod/scripting/gbans/globals.sp
sed -i "s/"${CURRENT_VERSION}"/"${NEXT_VERSION}"/g" frontend/package.json
sed -i "s/"v${CURRENT_VERSION}"/v"${NEXT_VERSION}"/g" Dockerfile
sed -i "s/v${CURRENT_VERSION}/v${NEXT_VERSION}/g" Makefile

git commit -a -m "[Release] Bump version to ${NEXT_VERSION}"
git push

git tag -a "v${NEXT_VERSION}" -m "v${NEXT_VERSION} Release"
git push --tags

goreleaser release --clean