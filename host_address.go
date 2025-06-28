package dnsmill

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"slices"
	"strings"
)

// HostAddresses represents a list of IP addresses and hostnames.
//
// When parsing, it may either parse a single host address or a list of host
// addresses.
type HostAddresses []HostAddress

func (a *HostAddresses) UnmarshalJSON(data []byte) error {
	var items []HostAddress
	if bytes.HasPrefix(data, []byte{'['}) {
		if err := json.Unmarshal(data, &items); err != nil {
			return fmt.Errorf("failed to parse HostAddresses array: %w", err)
		}
	} else {
		var item HostAddress
		if err := json.Unmarshal(data, &item); err != nil {
			return fmt.Errorf("failed to parse HostAddresses string: %w", err)
		}
		items = []HostAddress{item}
	}

	*a = items
	return nil
}

// ResolveIPs resolves the address strings into [net.IPAddr]s.
// This function is not cached. [net.DefaultResolver] is used to
// resolve the hostnames.
func (as HostAddresses) ResolveIPs(ctx context.Context) ([]net.IPAddr, error) {
	addrs := make([]net.IPAddr, 0, len(as))
	for _, a := range as {
		addrs1, err := a.ResolveIPs(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve address %q: %w", a.String(), err)
		}
		addrs = append(addrs, addrs1...)
	}
	return addrs, nil
}

// HostAddress is a string that has the host address plus optional flags
// in the format `[flag1[,flag2[...]]!]<address>`.
type HostAddress struct {
	Address string           `json:"address"`
	Flags   HostAddressFlags `json:"flags,omitempty"`
}

// ParseHostAddress parses a host address string into a HostAddress struct.
func ParseHostAddress(str string) (*HostAddress, error) {
	flagPart, address, ok := strings.Cut(str, "!")
	if !ok {
		return &HostAddress{
			Address: str,
		}, nil
	}

	flags := make(HostAddressFlags, 0, strings.Count(flagPart, ",")+1)
	for _, flag := range strings.Split(flagPart, ",") {
		flags = append(flags, HostAddressFlag(flag))
	}
	if err := flags.Validate(); err != nil {
		return nil, err
	}

	return &HostAddress{
		Address: address,
		Flags:   flags,
	}, nil
}

func (a *HostAddress) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("cannot parse host address: %w", io.ErrUnexpectedEOF)
	}

	switch data[0] {
	case '"':
		var str string
		if err := json.Unmarshal(data, &str); err != nil {
			return fmt.Errorf("failed to parse HostAddress string: %w", err)
		}

		return a.UnmarshalText([]byte(str))

	case '{':
		var raw HostAddress
		if err := json.Unmarshal(data, &raw); err != nil {
			return fmt.Errorf("failed to parse HostAddress object: %w", err)
		}

		if err := raw.Validate(); err != nil {
			return fmt.Errorf("invalid HostAddress %q: %w", raw.String(), err)
		}

		*a = raw
		return nil

	default:
		return fmt.Errorf("invalid JSON for HostAddress: expected string or object, got %s", data)
	}
}

func (a *HostAddress) UnmarshalText(data []byte) error {
	var parsed HostAddress

	if flagPart, address, ok := strings.Cut(string(data), "!"); ok {
		parsed.Address = address
		parsed.Flags = make(HostAddressFlags, 0, strings.Count(flagPart, ",")+1)
		for _, flag := range strings.Split(flagPart, ",") {
			parsed.Flags = append(parsed.Flags, HostAddressFlag(flag))
		}
	} else {
		parsed.Address = string(data)
	}

	if err := parsed.Validate(); err != nil {
		return fmt.Errorf("invalid HostAddress %q: %w", parsed.String(), err)
	}

	*a = parsed
	return nil
}

func (a HostAddress) Validate() error {
	if err := a.Flags.Validate(); err != nil {
		return fmt.Errorf("invalid host address flags: %w", err)
	}
	return nil
}

func (a HostAddress) String() string {
	return fmt.Sprintf("%s!%s", a.Address, a.Flags.String())
}

func (a HostAddress) ResolveIPs(ctx context.Context) ([]net.IPAddr, error) {
	switch {
	case a.Flags.Has(HostAddressIP):
		return a.resolveAsIP(ctx)
	case a.Flags.Has(HostAddressInterface):
		return a.resolveAsInterface(ctx)
	case a.Flags.Has(HostAddressExternal):
		return a.resolveAsExternal(ctx)
	case a.Flags.Has(HostAddressHostname):
		return a.resolveAsHostname(ctx)
	default:
		for _, flag := range []HostAddressFlag{
			HostAddressIP,
			HostAddressInterface,
			HostAddressHostname,
		} {
			aa := a
			aa.Flags = append(slices.Clone(aa.Flags), flag)

			if addrs, err := aa.ResolveIPs(ctx); err == nil {
				return addrs, nil
			}
		}

		return nil, fmt.Errorf(
			"cannot guess address for %q, consider explicitly using a flag",
			a.String(),
		)
	}
}

func (a HostAddress) resolveAsIP(_ context.Context) ([]net.IPAddr, error) {
	ip := net.ParseIP(a.Address)
	if ip == nil {
		return nil, fmt.Errorf("invalid IP address %q", a.Address)
	}
	return []net.IPAddr{{IP: ip}}, nil
}

func (a HostAddress) resolveAsInterface(ctx context.Context) ([]net.IPAddr, error) {
	iface, err := net.InterfaceByName(a.Address)
	if err != nil {
		return nil, fmt.Errorf("failed to get interface %q: %w", a.Address, err)
	}

	ifaceAddrs, err := iface.Addrs()
	if err != nil {
		return nil, fmt.Errorf("failed to get addresses for interface %q: %w", a.Address, err)
	}

	ifaceIPAddrs := make([]net.IPAddr, 0, len(ifaceAddrs))
	for _, addr := range ifaceAddrs {
		switch addr := addr.(type) {
		case *net.IPAddr:
			ifaceIPAddrs = append(ifaceIPAddrs, *addr)
		case *net.IPNet:
			ifaceIPAddrs = append(ifaceIPAddrs, net.IPAddr{IP: addr.IP})
		default:
			return nil, fmt.Errorf(
				"unsupported address type %T for interface %q",
				addr, a.Address)
		}
	}

	addrs := ifaceIPAddrs

	switch {
	case a.Flags.Has(HostAddressIPv4Only):
		addrs = slices.DeleteFunc(addrs, func(addr net.IPAddr) bool {
			return addr.IP.To4() == nil // delete what's not v4
		})
	case a.Flags.Has(HostAddressIPv6Only):
		addrs = slices.DeleteFunc(addrs, func(addr net.IPAddr) bool {
			return addr.IP.To4() != nil // delete what is v4
		})
	case a.Flags.Has(HostAddressExternal):
		externalIPs, err := ResolveExternalIPsForLocalAddrs(ctx, ifaceIPAddrs)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to resolve external IPs for interface %q using %q: %w",
				a.Address, ExternalIPProvider, err)
		}
		addrs = externalIPs
	}

	return addrs, nil
}

func (a HostAddress) resolveAsExternal(ctx context.Context) ([]net.IPAddr, error) {
	if a.Address != "" {
		return nil, fmt.Errorf("external address cannot have an actual address (needs to be empty), has: %q", a.Address)
	}

	externalIPs, err := ResolveExternalIPs(ctx, a.Flags)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve external IPs: %w", err)
	}

	return externalIPs, nil
}

func (a HostAddress) resolveAsHostname(ctx context.Context) ([]net.IPAddr, error) {
	resolved, err := net.DefaultResolver.LookupIPAddr(ctx, a.Address)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve hostname %q: %w", a.Address, err)
	}
	return resolved, nil
}

// HostAddressFlags represents a list of host address flags.
type HostAddressFlags []HostAddressFlag

func (f HostAddressFlags) Has(anyOf ...HostAddressFlag) bool {
	for _, flag := range f {
		if slices.Contains(anyOf, flag) {
			return true
		}
	}
	return false
}

func (f HostAddressFlags) String() string {
	strs := make([]string, len(f))
	for i, flag := range f {
		strs[i] = string(flag)
	}
	return strings.Join(strs, ",")
}

func (fs HostAddressFlags) Validate() error {
	for _, f := range fs {
		if err := f.Validate(); err != nil {
			return fmt.Errorf("invalid host address flag %q: %w", f, err)
		}

		if ftype := validHostAddressFlags[f]; ftype != "" {
			oindex := slices.IndexFunc(fs, func(other HostAddressFlag) bool {
				otype := validHostAddressFlags[other]
				return other != f && otype != "" && otype == ftype
			})

			if oindex != -1 {
				other := fs[oindex]
				return fmt.Errorf(
					"mutually exclusive host address flags %q and %q (both have type %s)",
					f, other, ftype,
				)
			}
		}
	}

	return nil
}

// HostAddressFlag is the type for a possible flag for a host address.
type HostAddressFlag string

// Flags for casting host address types.
const (
	HostAddressIP        HostAddressFlag = "ip"
	HostAddressInterface HostAddressFlag = "interface"
	HostAddressExternal  HostAddressFlag = "external" // also modifier for Interface
	HostAddressHostname  HostAddressFlag = "hostname"
)

// Modifier flags for host addresses.
const (
	HostAddressIPv4Only HostAddressFlag = "ipv4"
	HostAddressIPv6Only HostAddressFlag = "ipv6"
)

// map of valid host address flags to flag type, which is used for mutual
// exclusivity.
var validHostAddressFlags = map[HostAddressFlag]string{
	HostAddressIP:        "type",
	HostAddressInterface: "type",
	HostAddressExternal:  "type",
	HostAddressHostname:  "type",
	HostAddressIPv4Only:  "ip-version",
	HostAddressIPv6Only:  "ip-version",
}

func (f HostAddressFlag) Validate() error {
	_, ok := validHostAddressFlags[f]
	if !ok {
		return fmt.Errorf("unknown host address flag %q", f)
	}
	return nil
}
