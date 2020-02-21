#!/usr/bin/env bash

# directories to exclude
dirs=(
   proto/
   plugins/crdplugin/pkg/client/
   plugins/nsmplugin/descriptor/adapter/
)
excluded=$(echo ${dirs[@]} | sed 'y/ /,/')

# max time to analyze
timeout=3m0s

golangci-lint run --timeout=$timeout --skip-dirs $excluded