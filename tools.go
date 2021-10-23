//go:build tools
// +build tools

package main

import (
	_ "github.com/deepmap/oapi-codegen/cmd/oapi-codegen"
	_ "github.com/rakyll/statik"
	_ "golang.org/x/lint/golint"
)
