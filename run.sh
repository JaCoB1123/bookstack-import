#!/bin/sh

export BOOKSTACK_URL=http://srv-dev01:6875
export BOOKSTACK_TOKEN_ID=J3Y5uMtlF0ueHgOHJrEvcnnqBU24f8hN
export BOOKSTACK_TOKEN_SECRET=NU55YgNcqCy8EtlEm1pYscGWiWcaNddv
export BOOKSTACK_IMPORT_PATH=C:\\temp\\notes\\Knowledge-Base-UB-S

#go build . &&
    ./bookstack-import
