# dnsmill

Declaratively set your DNS records with [dnsmill], powered by [libdns].

[dnsmill]: https://github.com/diamondburned/dnsmill
[libdns]: https://github.com/libdns/libdns

## Use Cases

dnsmill is useful if you already have declarative infrastructure, such as with
[NixOS]. Using this tool, you can ensure that your DNS records match up with your
web server of choice without needing to manually update your DNS provider.

[NixOS]: https://nixos.org/

## Installation

It's recommended that you build this from source using Go 1.21 or later. To do
so, simply run:

```sh
go install libdb.so/dnsmill/cmd/dnsmill@latest
```

### Nix Flake

The repository contains a Nix flake that you can use to build and run dnsmill.
To do so, clone the repository and run the following command:

```sh
nix run .#
```

## Usage

First, declare a YAML or JSON file for your DNS profile. Below is a very
minimal example for Cloudflare:

```yml
config:
  duplicatePolicy: overwrite

providers:
  cloudflare: [libdb.so]

dnsmill_test.libdb.so: localhost
```

This profile sets the `dnsmill_test.libdb.so` record to `127.0.0.1` and `::1`.

Then, you'll need to set your `CLOUDFLARE_API_TOKEN` environment variable to a
scoped API token in your Cloudflare account with the `Zone-Zone-Read` and
`Zone-DNS-Edit` permissions:

```sh
export CLOUDFLARE_API_TOKEN=your_token_here
```

To apply this profile, put it in a file called `profile.yml` and run the
following command:

```sh
dnsmill profile.yml
```

And that's it! Your DNS should now be set:

```
Apr  6 03:36:03.701 INF applying fresh libdns record profile=profile.yml domain=libdb.so provider=cloudflare record.type=AAAA record.name=dnsmill_test record.value=::1
Apr  6 03:36:03.702 INF applying fresh libdns record profile=profile.yml domain=libdb.so provider=cloudflare record.type=A record.name=dnsmill_test record.value=127.0.0.1
Apr  6 03:36:05.557 INF applied libdns record profile=profile.yml domain=libdb.so provider=cloudflare record.id=37e833f8a80fabc9747adf6e94aae894 record.type=AAAA record.name=dnsmill_test record.value=::1
Apr  6 03:36:05.557 INF applied libdns record profile=profile.yml domain=libdb.so provider=cloudflare record.id=47c54fd81e07ee8bde61ef0761838f01 record.type=A record.name=dnsmill_test record.value=127.0.0.1
```

### Host Address Types

In the above YAML example, our `localhost` is a "host address". This address is
resolved to IP addresses by dnsmill and then applied to the DNS provider.

dnsmill supports 4 types of host addresses:

- IP address, which is used verbatim (IP must be valid, e.g. `127.0.0.1`)
- Interface name, which is resolved to the interface's IP address (e.g. `eth0`)
- Hostname, which is resolved to the host's IP address (e.g. `localhost` or
  `example.com`) using the system's DNS resolver
- External IP address via `external!`, which is resolved to the external IP
  address of the network using `ifconfig.me`.

It is possible to explicitly specify the type of host address by prefixing
the host address string with a type followed by a `!`:

- `ipv4!127.0.0.1` for an IPv4 address
- `ipv6!::1` for an IPv6 address
- `interface!eth0` for an interface name, optionally with:
  - `interface,ipv4!eth0` for the IPv4 addresses of the interface,
  - `interface,ipv6!eth0` for the IPv6 addresses of the interface, and
  - `interface,external!eth0` to resolve all internal IP addresses of the
    interface to external IP addresses
    - You can combine the `ipv4` and `ipv6` flags here, too
- `hostname!example.com` for a hostname
- `external!` for the external IP address, optionally with:
  - `external,ipv4!` for the external IPv4 addresses only, and
  - `external,ipv6!` for the external IPv6 addresses only
  - There must be no host address after the `!` delimiter

## Extending

You can extend dnsmill with more DNS providers that libdns supports. To do so,
you must first create a new Go module that contains a new package main that
runs [dnsmillcmd]. For example:

```sh
go mod init localhost/dnsmill-custom
cat<<EOF > main.go
package main

import (
    "libdb.so/dnsmill/cmd/dnsmillcmd"
    _ "libdns.so/dnsmill/providers"
    _ "localhost/dnsmill-custom/vercel" # add a custom Vercel provider
)

func main() {
    dnsmillcmd.Main()
}
EOF
```

Then, you'll need to write a small package that wraps around libdns's provider.
To help with that, you can refer to the [Vercel
provider](./providers/vercel/vercel.go) in this repository. Here, the package
is put in the `vercel` directory:

```md
- dnsmill-custom/ # module localhost/dnsmill-custom
  - main.go
  - go.mod
  - go.sum
  - vercel/
    - vercel.go
```

Then, you can build and run the package as usual.

[dnsmillcmd]: https://pkg.go.dev/libdb.so/dnsmill/cmd/dnsmillcmd

## Name Origin

The name is a combination of DNS and milling (like mill or milling machine).
It's pronounced as "dee-en-es-mill".
