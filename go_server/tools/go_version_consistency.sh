#!/usr/bin/env bash
# This script is meant to ensure all the various tooling in this repo uses the same
# version of Go (eg. CI, Docker, builds, etc).
set -uxo pipefail

GO_VER="$(mktemp)"

grep '^go ' go.mod | cut -d ' ' -f 2 >> ${GO_VER}

echo $(find ./sysops -name '*Dockerfile')

for dockerfile in $(find ./sysops -name '*Dockerfile'); do
    grep -E -o 'FROM golang:[0-9.]+' $dockerfile

    if [ $? == '1' ]; then
        echo "$dockerfile is not a go image, skipping."
    else
        new_ver=$(grep -E -o 'FROM golang:[0-9.]+' $dockerfile | cut -d ':' -f 2)
        echo $dockerfile
        echo $new_ver >> $GO_VER
    fi

done

if [ "$(uniq "$GO_VER" | wc -l | xargs)" != '1' ]; then
    echo "Inconsistent Go versions in use: $(cat $GO_VER | sort -u | tr "\n" ' ')"
    echo $dockerfile
    echo ${GO_VER}
    exit 1
fi
echo "All Go versions match."
