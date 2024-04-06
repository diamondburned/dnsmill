// Package main is the main dnsmill command. It bundles all built-in providers
// in this repository.
package main

import (
	"libdb.so/dnsmill/cmd/dnsmill/dnsmillcmd"
	_ "libdb.so/dnsmill/providers"
)

func main() {
	dnsmillcmd.Main()
}
