#!/usr/bin/env bash
set -e

DEST_DIR="/usr/local/bin"


if [ ! -d ${DEST_DIR} ]; then
	mkdir ${DEST_DIR}
fi

go build -v -o "${DEST_DIR}"/l2sm-init ./cmd/l2sm-init 
go build -v -o "${DEST_DIR}"/l2sm-vxlans ./cmd/l2sm-vxlans 
go build -v -o "${DEST_DIR}"/l2sm-add-port ./cmd/l2sm-add-port 
