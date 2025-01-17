#!/bin/sh

reflex -r '^cmd/websocket/(.*?).*|.env|pkg/(.*?).*$' -s -- sh -c 'cd cmd/websocket && go run main.go'
