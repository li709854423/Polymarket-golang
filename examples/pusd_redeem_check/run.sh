#!/bin/bash

set -e

cd "$(dirname "$0")/../.."
export GOCACHE="${GOCACHE:-/tmp/polymarket-client-go-build}"
go run ./examples/pusd_redeem_check
