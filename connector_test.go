package core

import (
	"context"
	"testing"
	"time"

	"github.com/hood-chat/core/entity"
	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	bhost "github.com/libp2p/go-libp2p/p2p/host/blank"
	swarmt "github.com/libp2p/go-libp2p/p2p/net/swarm/testing"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/require"
)

type DisconnectAbleHost interface {
	host.Host
	ON(t *testing.T)
	OFF()
}

func NewDisconnectAbleHost(t *testing.T) DisconnectAbleHost {
	identity, err := entity.CreateIdentity("")
	if err != nil {
		panic("")
	}
	sk, err := identity.DecodePrivateKey("passphrase todo!")
	if err != nil {
		panic("")
	}
	opt := swarmt.OptPeerPrivateKey(sk)

	netw := swarmt.GenSwarm(t, opt)
	h := bhost.NewBlankHost(netw)
	h.Addrs()
	return &disconnectAbleHost{h, identity, h.Addrs()}
}

type disconnectAbleHost struct {
	host.Host
	id     entity.Identity
	adders []ma.Multiaddr
}

func (d *disconnectAbleHost) ON(t *testing.T) {
	d.Close()
	sk, err := d.id.DecodePrivateKey("passphrase todo!")
	if err != nil {
		panic("")
	}
	opt := swarmt.OptPeerPrivateKey(sk)

	netw := swarmt.GenSwarm(t, opt)
	netw.ListenClose(netw.ListenAddresses()...)
	netw.Listen(d.adders...)
	h := bhost.NewBlankHost(netw)
	d = &disconnectAbleHost{h, d.id, d.adders}
}
func (d *disconnectAbleHost) OFF() {
	err := d.Network().Close()
	if err != nil {
		panic("can not off")
	}
	err = d.Close()
	if err != nil {
		panic("can not off")
	}
}
func getNetHosts(t *testing.T, n int) []DisconnectAbleHost {
	var out []DisconnectAbleHost

	for i := 0; i < n; i++ {
		h := NewDisconnectAbleHost(t)
		t.Cleanup(func() { h.Close() })
		out = append(out, h)
	}

	return out
}

func TestConnector(t *testing.T) {
	t.Log("start test")
	err := logging.SetLogLevel("msgr-core", "DEBUG")
	require.NoError(t, err)
	// err = logging.SetLogLevel("*", "DEBUG")
	require.NoError(t, err)
	hosts := getNetHosts(t, 5)
	primary := hosts[0]
	disconnectFor(t, 1*time.Second, hosts[2])
	disconnectFor(t, 1*time.Second, hosts[1])
	// primary.Close()
	connector := NewConnector(primary)
	connector.Need("test", hosts[1].Peerstore().PeerInfo(hosts[1].ID()))
	connector.Need("test", hosts[1].Peerstore().PeerInfo(hosts[1].ID()))
	connector.Need("test", hosts[2].Peerstore().PeerInfo(hosts[2].ID()))
	disconnectFor(t, 2*time.Second, hosts[2])
	disconnectFor(t, 2*time.Second, hosts[1])
	require.Eventually(t, func() bool { return len(primary.Network().Peers()) == 2 }, 5*time.Second, 1*time.Second)
	// time.Sleep(5 * time.Second)
	connector.Done("test", hosts[1].ID())
	connector.Done("test", hosts[1].ID())
	connector.Done("test", hosts[2].ID())
	disconnectFor(t, 30*time.Second, hosts[2])
	disconnectFor(t, 30*time.Second, hosts[1])
	// for _, v := range primary.Network().Conns() {
	// 	v.Close()
	// }
	go require.Eventually(t, func() bool { log.Debug(len(primary.Network().Peers())); return len(primary.Network().Peers()) != 2 }, 5*time.Second, 1*time.Second)


}

func TestWithoutConnector(t *testing.T) {
	t.Log("start test")
	err := logging.SetLogLevel("msgr-core", "DEBUG")
	require.NoError(t, err)
	// err = logging.SetLogLevel("*", "DEBUG")
	require.NoError(t, err)
	hosts := getNetHosts(t, 5)
	primary := hosts[0]

	// primary.Close()
	primary.Connect(context.Background(), hosts[1].Peerstore().PeerInfo(hosts[1].ID()))
	primary.Connect(context.Background(), hosts[2].Peerstore().PeerInfo(hosts[2].ID()))

	disconnectFor(t, 30*time.Second, hosts[2])
	disconnectFor(t, 30*time.Second, hosts[1])
	// for _, v := range primary.Network().Conns() {
	// 	v.Close()
	// }
	require.Eventually(t, func() bool { return !isConnected2(primary, hosts[1].ID(), hosts[2].ID()) }, 5*time.Second, 1*time.Second)
	require.Eventually(t, func() bool { return len(primary.Network().Peers()) != 2 }, 5*time.Second, 1*time.Second)
	// require.Eventually(t, func() bool { return !isConnected(primary) }, 5*time.Second, 1*time.Second)

}

func disconnectFor(t *testing.T, d time.Duration, h DisconnectAbleHost) {
	h.OFF()
	go time.AfterFunc(d, func() {
		h.ON(t)
	})

}

func isConnected(h host.Host) bool {
	var f bool = true
	for _, con := range h.Network().Conns() {
		_, err := con.NewStream(context.Background())
		f = f && err == nil
	}
	return f
}

func isConnected2(h host.Host, ps ...peer.ID) bool {
	var f bool = true
	for _, p := range ps {
		f = f && h.Network().Connectedness(p) == network.Connected
	}
	return f
}
