#!/bin/sh
set -e
rm -rf ./public
go run ./cmd/generate_checks/generate_checks.go >data/checks.json
go run ./cmd/generate_config/generate_config.go >content/docs/configuration/default_config/index.md

(
	cd themes/docsy
	# --omit=dev so we don't try to install Hugo as an NPM module
	npm install --omit=dev
)

go run github.com/gohugoio/hugo@v0.110.0 --minify
