package network

import (
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"
)

// PeerAddr holds the address of a peer node.
type PeerAddr struct {
	IP       net.IP
	Port     uint16
	LastSeen time.Time
	Services uint64 // bitmask of supported services
}

func (p *PeerAddr) String() string {
	return fmt.Sprintf("%s:%d", p.IP.String(), p.Port)
}

// ServiceFull indicates a full node.
const ServiceFull = uint64(1)

// Discovery manages peer discovery via DNS seeds, hardcoded bootstraps, and DHT.
type Discovery struct {
	mu           sync.RWMutex
	knownPeers   map[string]*PeerAddr // addr → peer
	connectedPeers map[string]*PeerAddr
	dnsSeeds     []string
	bootstraps   []string
	maxPeers     int
}

// MainnetDNSSeeds lists the official GAMINIUM DNS seeds.
var MainnetDNSSeeds = []string{
	"seed1.gaminium.io",
	"seed2.gaminium.io",
	"seed3.gaminium.io",
	"seed.gaminium.org",
}

// MainnetBootstraps are hardcoded bootstrap node addresses (fallback).
var MainnetBootstraps = []string{
	"boot1.gaminium.io:8333",
	"boot2.gaminium.io:8333",
}

// NewDiscovery creates a discovery manager.
func NewDiscovery(maxPeers int, dnsSeeds, bootstraps []string) *Discovery {
	return &Discovery{
		knownPeers:     make(map[string]*PeerAddr),
		connectedPeers: make(map[string]*PeerAddr),
		dnsSeeds:       dnsSeeds,
		bootstraps:     bootstraps,
		maxPeers:       maxPeers,
	}
}

// Discover attempts to find peers via DNS seeds and bootstraps.
func (d *Discovery) Discover() []*PeerAddr {
	var found []*PeerAddr

	// 1. DNS seed lookup
	for _, seed := range d.dnsSeeds {
		addrs, err := net.LookupHost(seed)
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ip := net.ParseIP(addr)
			if ip == nil {
				continue
			}
			peer := &PeerAddr{
				IP:       ip,
				Port:     8333,
				LastSeen: time.Now(),
				Services: ServiceFull,
			}
			found = append(found, peer)
			d.AddPeer(peer)
		}
	}

	// 2. Hardcoded bootstrap fallback
	if len(found) == 0 {
		for _, addrStr := range d.bootstraps {
			host, portStr, err := net.SplitHostPort(addrStr)
			if err != nil {
				continue
			}
			ip := net.ParseIP(host)
			if ip == nil {
				ips, err := net.LookupHost(host)
				if err != nil || len(ips) == 0 {
					continue
				}
				ip = net.ParseIP(ips[0])
			}
			_ = portStr
			peer := &PeerAddr{
				IP:       ip,
				Port:     8333,
				LastSeen: time.Now(),
				Services: ServiceFull,
			}
			found = append(found, peer)
			d.AddPeer(peer)
		}
	}

	return found
}

// AddPeer registers a known peer address.
func (d *Discovery) AddPeer(peer *PeerAddr) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.knownPeers[peer.String()] = peer
}

// RandomPeers returns up to n random known peers.
func (d *Discovery) RandomPeers(n int) []*PeerAddr {
	d.mu.RLock()
	defer d.mu.RUnlock()

	all := make([]*PeerAddr, 0, len(d.knownPeers))
	for _, p := range d.knownPeers {
		all = append(all, p)
	}

	rand.Shuffle(len(all), func(i, j int) {
		all[i], all[j] = all[j], all[i]
	})

	if n > len(all) {
		n = len(all)
	}
	return all[:n]
}

// KnownCount returns the number of known peers.
func (d *Discovery) KnownCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.knownPeers)
}
