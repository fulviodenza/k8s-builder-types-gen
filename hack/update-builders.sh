#!/usr/bin/env bash

echo "Running builder generator"
go run ./hack/custom-builder-gen/custom-builder-gen.go "$@"
