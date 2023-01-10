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
	"/dns/2ir.hoodchat.info/tcp/4001/p2p/12D3KooWL8o7oc961jtnEkEPsDkpoqVSV1FKmfH6q4am2jnfexmX",
	"/dns/2ir.hoodchat.info/udp/4001/quic/p2p/12D3KooWL8o7oc961jtnEkEPsDkpoqVSV1FKmfH6q4am2jnfexmX",
	"/dns/ir.hoodchat.info/tcp/4001/p2p/12D3KooWA5VK6oL1vJXpuHiBCufoeua9iRwoWH84UwkXAzGRi1qZ",
	"/dns/ir.hoodchat.info/udp/4001/quic/p2p/12D3KooWA5VK6oL1vJXpuHiBCufoeua9iRwoWH84UwkXAzGRi1qZ",
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
	bts, err := ParseBootstrapPeers(BootstrapNodes)
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
		libp2p.EnableAutoRelay(autorelay.WithStaticRelays(bts)),
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
	// sw,err := swarm.NewSwarm(basicHost.ID(),basicHost.Peerstore(),swarm.WithDialTimeout(1*time.Minute))
	// if err != nil {
	// 	return nil, err
	// }

	basicHost.Network()
	// Construct a datastore (needed by the DHT). This is just a simple, in-memory thread-safe datastore.
	dstore := dsync.MutexWrap(ds.NewMapDatastore())

	// Make the DHT
	kDht := dht.NewDHT(context.Background(), basicHost, dstore)

	bts, err := ParseBootstrapPeers(BootstrapNodes)
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
