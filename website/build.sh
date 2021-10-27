#!/bin/sh
set -e
rm -rf ./public
go run ./generate_checks.go >data/checks.json
hugo --minify
