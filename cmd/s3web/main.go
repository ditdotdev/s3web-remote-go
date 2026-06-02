// Package main provides the s3web remote server executable.
package main

import "github.com/ditdotdev/remote-sdk-go/remote"

func main() {
	remote.Serve("s3web")
}
