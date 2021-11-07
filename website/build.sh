#!/bin/sh
set -e
rm -rf ./public
go run ./generate_checks.go >data/checks.json
go run ./generate_config.go >content/docs/configuration/default_config/index.md
hugo --minify
