#!/bin/sh
set -e

mkdir -p /tmp/splitter

exec /usr/local/bin/splitter run "$@"
