package core

import (
	"context"

	"github.com/hood-chat/core/entity"
	ds "github.com/ipfs/go-datastore"
	dsync "github.com/ipfs/go-datastore/sync"
	libp2p "github.com/libp2p/go-libp2p"
	host "github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/p2p/host/autorelay"
	rh "github.com/libp2p/go-libp2p/p2p/host/routed"

	"github.com/ipfs/kubo/core/bootstrap"
	ma "github.com/multiformats/go-multiaddr"
)

var BootstrapNodes = []string{
	"/ip4/45.90.74.114/tcp/4001/p2p/12D3KooWPjwysxEgUrHWxFdPc2rBCogQ6Hdm1hDysaCG7KQi1QvF",
	"/ip4/45.90.74.114/udp/4001/quic/p2p/12D3KooWPjwysxEgUrHWxFdPc2rBCogQ6Hdm1hDysaCG7KQi1QvF",
}

type HostBuilder interface {
	Create(opt Option) (host.Host, error)
}

// func(Option)

type Option struct {
	LpOpt []libp2p.Option
	ID    peer.ID
}

func (opt *Option) SetIdentity(identity *entity.Identity) error {
	sk, err := identity.DecodePrivateKey("passphrase todo!")
	if err != nil {
		return err
	}
	opt.LpOpt = append(opt.LpOpt, libp2p.Identity(sk))
	opt.ID = peer.ID(identity.ID)
	return nil
}

func DefaultOption() Option {
	// Now, normally you do not just want a simple host, you want
	// that is fully configured to best support your p2p application.
	// Let's create a second host setting some more options.
	// Set your own keypair

	con, err := connmgr.NewConnManager(10, 100)
	if err != nil {
		panic(err)
	}

	opt := []libp2p.Option{
		libp2p.DefaultTransports,
		libp2p.DefaultSecurity,
		// Use the keypair we generated
		// Multiple listen addresses
		libp2p.DefaultListenAddrs,
		// Let's prevent our peer from having too many
		// connections by attaching a connection manager.
		libp2p.ConnectionManager(con),
		// libp2p.DefaultMuxers,
		// Let this host use relays and advertise itself on relays if
		// it finds it is behind NAT. Use libp2p.Relay(options...) to
		// enable active relays and more.
		// libp2p.EnableAutoRelay(),
		libp2p.EnableAutoRelay(autorelay.WithDefaultStaticRelays()),
		// If you want to help other peers to figure out if they are behind
		// NATs, you can launch the server-side of AutoNAT too (AutoRelay
		// already runs the client)
		//
		// This service is highly rate-limited and should not cause any
		// performance issues.
		libp2p.EnableNATService(),
		libp2p.EnableHolePunching(),
	}
	return Option{
		LpOpt: opt,
		ID:    "",
	}
}

type DefaultRoutedHost struct {
}

func (b DefaultRoutedHost) Create(opt Option) (host.Host, error) {
	basicHost, err := libp2p.New(opt.LpOpt...)
	if err != nil {
		return nil, err
	}

	// Construct a datastore (needed by the DHT). This is just a simple, in-memory thread-safe datastore.
	dstore := dsync.MutexWrap(ds.NewMapDatastore())

	// Make the DHT
	kDht := dht.NewDHT(context.Background(), basicHost, dstore)

	bts, err := ParseBootstrapPeers(BootstrapNodes)
	if err != nil {
		return nil, err
	}
	btconf := bootstrap.BootstrapConfigWithPeers(bts)
	btconf.MinPeerThreshold = 2

	// connect to the chosen ipfs nodes
	_, err = bootstrap.Bootstrap(ID, basicHost, kDht, btconf)
	if err != nil {
		log.Error("bootstrap failed. ", err)
		return nil, err
	}
	// Make the routed host
	routedHost := rh.Wrap(basicHost, kDht)

	log.Infof("core bootstrapped and ready on:", routedHost.Addrs())
	return routedHost, nil
}

func ParseBootstrapPeers(addrs []string) ([]peer.AddrInfo, error) {
	maddrs := make([]ma.Multiaddr, len(addrs))
	for i, addr := range addrs {
		var err error
		maddrs[i], err = ma.NewMultiaddr(addr)
		if err != nil {
			return nil, err
		}
	}
	return peer.AddrInfosFromP2pAddrs(maddrs...)
}
