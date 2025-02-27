#!/bin/sh

reflex -r '^cmd/backend/(.*?).*|.env|pkg/(.*?).*$' -s -- sh -c 'cd cmd/backend && go run main.go'
