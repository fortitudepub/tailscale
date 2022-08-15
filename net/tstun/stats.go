package tstun

import (
	"fmt"
	"net/netip"
	"time"

	"tailscale.com/types/ipproto"
)

// NetworkConnection is a tuple of source IP:port and destination IP:port.
// It is an approximation of the concept of a "connection".
type NetworkConnection struct {
	Protocol    ipproto.Proto
	Source      netip.AddrPort
	Destination netip.AddrPort
}

func (c NetworkConnection) MarshalText() ([]byte, error) {
	var src string
	if c.Source.Port() == 0 {
		src = c.Source.Addr().String()
	} else {
		src = c.Source.String()
	}

	var dst string
	if c.Destination.Port() == 0 {
		dst = c.Destination.Addr().String()
	} else {
		dst = c.Destination.String()
	}

	if c.Protocol == ipproto.Unknown {
		return []byte(fmt.Sprintf("%s ⇔ %s", src, dst)), nil
	}
	return []byte(fmt.Sprintf("%s: %s ⇔ %s", c.Protocol, src, dst)), nil
}

// NetworkTraffic is statistics about a connection.
type NetworkTraffic struct {
	TxPackets uint64 // number of packets sent
	TxBytes   uint64 // number of bytes sent
	RxPackets uint64 // number of packets received
	RxBytes   uint64 // number of bytes received
}

// NetworkTrafficStats is the network traffic statistics for
// a particular span of time.
// It represents a directed cyclic graph.
// It can represent the traffic from the perspective of either
// a particular tailnode or for the entire tailnet.
//
// When representing an entire tailnet, the physical address may
// be ambiguous if it refers to a private IP address,
// which resolves differently depending on which peer is trying to reach it.
type NetworkTrafficStats struct {
	StartTime time.Time
	EndTime   time.Time

	// VirtualTraffic is traffic between two tailnodes.
	// For traffic for a particular tailnode,
	// the source is always the address of the current tailnode
	// even if the connection was initiated by the destination.
	// For traffic for an entire tailnet,
	// the source address is lexicographically smaller address
	// between the two peers.
	VirtualTraffic map[NetworkConnection]*NetworkTraffic
}
