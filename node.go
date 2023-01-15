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
	"/ip4/194.5.178.130/tcp/4002/p2p/12D3KooWK5ok6gr6L5SVuaAtme3HfUWW4YYm4AAsqUYfZeonKM1C",
	"/ip6/2a01:4f8:160:33c5:250:56ff:fe94:dedc/tcp/4002/p2p/12D3KooWK5ok6gr6L5SVuaAtme3HfUWW4YYm4AAsqUYfZeonKM1C",
	"/ip4/194.5.178.130/udp/4002/quic/p2p/12D3KooWK5ok6gr6L5SVuaAtme3HfUWW4YYm4AAsqUYfZeonKM1C",
	"/ip6/2a01:4f8:160:33c5:250:56ff:fe94:dedc/udp/4002/quic/p2p/12D3KooWK5ok6gr6L5SVuaAtme3HfUWW4YYm4AAsqUYfZeonKM1C",
	"/ip4/194.5.178.130/udp/4002/quic-v1/p2p/12D3KooWK5ok6gr6L5SVuaAtme3HfUWW4YYm4AAsqUYfZeonKM1C",
	"/ip6/2a01:4f8:160:33c5:250:56ff:fe94:dedc/udp/4002/quic-v1/p2p/12D3KooWK5ok6gr6L5SVuaAtme3HfUWW4YYm4AAsqUYfZeonKM1C",
}

var StaticRelays = []string{
	"/ip6/2a01:4f8:160:33c5:250:56ff:fe94:dedc/udp/4001/quic/p2p/12D3KooWBFpA7pCMBySBqtduBVkakVQ3bmmaeagB83WHoruBN9s9",
	"/ip4/194.5.178.130/tcp/4001/p2p/12D3KooWBFpA7pCMBySBqtduBVkakVQ3bmmaeagB83WHoruBN9s9",
	"/ip6/2a01:4f8:160:33c5:250:56ff:fe94:dedc/tcp/4001/p2p/12D3KooWBFpA7pCMBySBqtduBVkakVQ3bmmaeagB83WHoruBN9s9",
	"/ip4/194.5.178.130/udp/4001/quic/p2p/12D3KooWBFpA7pCMBySBqtduBVkakVQ3bmmaeagB83WHoruBN9s9",
}

type Builder interface {
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
	bts, err := ParseBootstrapPeers(StaticRelays)
	if err != nil {
		panic(err)
	}
	con, err := connmgr.NewConnManager(10, 100)
	if err != nil {
		panic(err)
	}

	opt := []libp2p.Option{
		libp2p.DefaultTransports,
		libp2p.DefaultSecurity,
		libp2p.DefaultListenAddrs,
		libp2p.ConnectionManager(con),
		libp2p.EnableAutoRelay(autorelay.WithCircuitV1Support(),autorelay.WithStaticRelays(bts)),
		libp2p.EnableNATService(),
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

	bts, err := ParseBootstrapPeers(append(BootstrapNodes, StaticRelays...))
	if err != nil {
		return nil, err
	}
	btconf := bootstrap.BootstrapConfigWithPeers(bts)
	btconf.MinPeerThreshold = 1

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
