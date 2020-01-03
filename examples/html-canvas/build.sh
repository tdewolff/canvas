#!/usr/bin/sh

GOARCH=wasm GOOS=js go build -o lib.wasm main.go
