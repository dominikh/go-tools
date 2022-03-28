#!/bin/sh
set -e
rm -rf ./public
go run ./cmd/generate_checks/generate_checks.go >data/checks.json
go run ./cmd/generate_config/generate_config.go >content/docs/configuration/default_config/index.md
hugo --minify
