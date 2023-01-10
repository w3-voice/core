package core

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/hood-chat/core/entity"
	logging "github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	bf "github.com/libp2p/go-libp2p/p2p/discovery/backoff"
	bhost "github.com/libp2p/go-libp2p/p2p/host/blank"
	"github.com/libp2p/go-libp2p/p2p/net/swarm"
	swarmt "github.com/libp2p/go-libp2p/p2p/net/swarm/testing"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/require"
)

// TODO: need to fix udp port not close as i ask
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
	return &disconnectAbleHost{h, identity, h.Addrs(), netw}
}

type disconnectAbleHost struct {
	host.Host
	id     entity.Identity
	adders []ma.Multiaddr
	swarm  *swarm.Swarm
}

func (d *disconnectAbleHost) ON(t *testing.T) {
	sk, err := d.id.DecodePrivateKey("passphrase todo!")
	if err != nil {
		panic("")
	}
	opt := swarmt.OptPeerPrivateKey(sk)

	netw := swarmt.GenSwarm(t, opt)
	netw.ListenClose(netw.ListenAddresses()...)
	netw.Listen(d.adders...)
	h := bhost.NewBlankHost(netw)
	d = &disconnectAbleHost{h, d.id, d.adders,netw}
}
func (d *disconnectAbleHost) OFF() {
	d.swarm.Close()
	d.Host.Close()
	
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
	err = logging.SetLogLevel("*", "DEBUG")
	require.NoError(t, err)
	hosts := getNetHosts(t, 5)
	primary := hosts[0]

	connector := NewConnector(primary)
	connector.Need("test", hosts[1].Peerstore().PeerInfo(hosts[1].ID()))
	connector.Need("test", hosts[1].Peerstore().PeerInfo(hosts[1].ID()))
	connector.Need("test", hosts[2].Peerstore().PeerInfo(hosts[2].ID()))

	require.Eventually(t, func() bool { return len(primary.Network().Peers()) == 2 }, 10*time.Second, 1*time.Second)

	connector.Done("test", hosts[1].ID())
	connector.Done("test", hosts[1].ID())
	connector.Done("test", hosts[2].ID())
	disconnectFor(t, 30*time.Second, hosts[2])
	disconnectFor(t, 30*time.Second, hosts[1])
	go require.Eventually(t, func() bool { log.Debug(len(primary.Network().Peers())); return len(primary.Network().Peers()) != 2 }, 5*time.Second, 1*time.Second)

}

func TestWithoutConnector(t *testing.T) {
	t.Log("start test")
	err := logging.SetLogLevel("msgr-core", "DEBUG")
	require.NoError(t, err)
	err = logging.SetLogLevel("*", "DEBUG")
	require.NoError(t, err)
	hosts := getNetHosts(t, 5)
	primary := hosts[0]
	disconnectFor(t, 30*time.Second, hosts[2])
	disconnectFor(t, 30*time.Second, hosts[1])

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

func TestPolynomialBackoff(t *testing.T) {
	bkf := bf.NewPolynomialBackoff(time.Second, time.Minute*2, bf.NoJitter, time.Second, []float64{0.5, 2, 2.5}, rand.NewSource(0))
	b1 := bkf()
	b2 := bkf()

	if b1.Delay() != time.Second || b2.Delay() != time.Second {
		t.Fatal("incorrect delay time")
	}
	sum := time.Second * 0
	for i := 0; i < 10; i++ {
		delay := b1.Delay()
		sum += delay
		t.Logf("delay: %s, sum: %s", delay, sum)
	}
}
