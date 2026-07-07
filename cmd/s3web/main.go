// Copyright Dit 2026
// SPDX-License-Identifier: BUSL-1.1

// Package main provides the s3web remote server executable.
package main

import "github.com/ditdotdev/remote-sdk-go/remote"

func main() {
	remote.Serve("s3web")
}
