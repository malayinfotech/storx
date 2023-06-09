#!/usr/bin/env bash

set -ueo pipefail
set +x

exitcode=0

for component in "satellite" "storagenode" "multinode"
do
	if grep -q "eslint-plugin-storx@github:storx/eslint-storx" "./web/$component/package-lock.json"; then
		echo "$component/package-lock.json import for eslint-storx should not be changed."
		exitcode=-1
	fi
done

exit $exitcode