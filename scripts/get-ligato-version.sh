#!/usr/bin/env bash

get_ligato_version() {
	LIGATO_AGENT_DEP="go.cdnf.io/cnf-nsm go.ligato.io/vpp-agent"
	go mod graph | grep "$LIGATO_AGENT_DEP" | awk 'BEGIN { FS = "@" } ; { print $2 }'
}
