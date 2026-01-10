// Copyright 2024 Universidad Carlos III de Madrid
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"encoding/binary"
	"fmt"
	"net"
)

func ipv4ToUint32(ip net.IP) (uint32, error) {
	v4 := ip.To4()
	if v4 == nil {
		return 0, fmt.Errorf("not an IPv4 address: %v", ip)
	}
	return binary.BigEndian.Uint32(v4), nil
}

func uint32ToIPv4(n uint32) net.IP {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, n)
	return net.IP(b)
}

// allocateIPv4s returns count IPs starting at host offset ipStart within cidr.
// It enforces: IPv4, ipStart>=1, and avoids network/broadcast addresses for typical subnets.
func AllocateIPv4s(cidr string, ipStart int, count int) ([]string, error) {
	if count <= 0 {
		return nil, fmt.Errorf("count must be > 0")
	}
	if ipStart < 1 {
		return nil, fmt.Errorf("ipStart must be >= 1 (got %d)", ipStart)
	}

	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, fmt.Errorf("invalid CIDR %q: %w", cidr, err)
	}

	base := ip.To4()
	if base == nil {
		return nil, fmt.Errorf("CIDR %q is not IPv4 (IPv6 not supported)", cidr)
	}

	ones, bits := ipnet.Mask.Size()
	if bits != 32 {
		return nil, fmt.Errorf("unexpected mask size for %q", cidr)
	}

	// total addresses in subnet
	total := 1 << uint32(32-ones)
	if total < 4 {
		// /31, /32 have special semantics; keep it simple unless you want to support them explicitly
		return nil, fmt.Errorf("CIDR %q too small / not supported (size=%d)", cidr, total)
	}

	// Compute usable host range: [1, total-2] to avoid network (0) and broadcast (total-1)
	minHost := 1
	maxHost := total - 2

	lastHost := ipStart + count - 1
	if ipStart < minHost || lastHost > maxHost {
		return nil, fmt.Errorf(
			"CIDR %q cannot allocate %d IPs from start %d (usable host range %d..%d)",
			cidr, count, ipStart, minHost, maxHost,
		)
	}

	baseU, err := ipv4ToUint32(base)
	if err != nil {
		return nil, err
	}

	ips := make([]string, 0, count)
	for i := 0; i < count; i++ {
		hostOffset := uint32(ipStart + i)
		ips = append(ips, uint32ToIPv4(baseU+hostOffset).String())
	}
	return ips, nil
}
