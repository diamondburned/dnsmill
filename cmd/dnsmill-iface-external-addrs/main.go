package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"libdb.so/dnsmill"
)

func main() {
	flag.Parse()

	hostAddr := flag.Arg(0)
	if hostAddr == "" {
		hostAddr = "external"
	}

	addrs, err := (&dnsmill.HostAddresses{hostAddr}).ResolveIPs(context.Background())
	for _, addr := range addrs {
		fmt.Println(addr.String())
	}
	if err != nil {
		log.Fatalf("failed to resolve external IPs for interface %q: %v", hostAddr, err)
	}
}
