package core

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

type Connector interface {
	Need(proc string, p peer.AddrInfo)
	Done(proc string, p peer.ID)
}

func NewConnector(h host.Host) Connector {
	return newConnector(h)
}

var _ Connector = (*connector)(nil)

type connector struct {
	h      host.Host
	needed *PeerSet
	bctx   context.Context
	cancel context.CancelFunc
}

func newConnector(h host.Host) *connector {
	c := connector{}
	c.h = h
	c.needed = NewPeerSet()
	c.h.Network().Notify((*connectorNotifiee)(&c))
	c.bctx = nil
	c.cancel = nil

	return &c
}

func (c *connector) background(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	for {
		select {
		case t := <-ticker.C:
			peers := c.needed.Turn(t)
			for _, peer := range peers {
				c.connect(peer)
			}

		case <-ctx.Done():
			ticker.Stop()
			return
		}
	}

}

func (c *connector) mayStart() {
	if c.bctx == nil {
		c.bctx, c.cancel = context.WithCancel(context.Background())
		go c.background(c.bctx)
	}

}

func (c *connector) mayStop() {
	if c.needed.Empty() && c.bctx != nil {
		c.cancel()
		c.bctx = nil
		c.cancel = nil
	}
}

func (c *connector) connect(p peer.AddrInfo) {
	if c.h.Network().Connectedness(p.ID) != network.Connected {
		go func(pi peer.AddrInfo) {
			err := c.h.Connect(context.Background(), pi)
			if err != nil {
				c.needed.Failed(pi.ID)
				return
			}
		}(p)
	}
}

func (c *connector) Need(proc string, p peer.AddrInfo) {
	c.h.ConnManager().Protect(p.ID, proc)
	c.needed.Add(proc, p)
	c.needed.Force(p.ID)
	c.mayStart()
	c.connect(p)
}
func (c *connector) Done(proc string, p peer.ID) {
	c.h.ConnManager().Unprotect(p, proc)
	c.needed.Remove(proc, p)
	c.mayStop()
}

type connectorNotifiee connector

func (cn *connectorNotifiee) connector() *connector {
	return (*connector)(cn)
}

func (cn *connectorNotifiee) Listen(network.Network, ma.Multiaddr)      {}
func (cn *connectorNotifiee) ListenClose(network.Network, ma.Multiaddr) {}
func (cn *connectorNotifiee) Connected(n network.Network, c network.Conn) {
	log.Debug("peer connected")
	cn.connector().needed.Done(c.RemotePeer())
}
func (cn *connectorNotifiee) Disconnected(n network.Network, c network.Conn) {
	log.Debugf("node %v peer %v disconnected ", cn.h.ID(), c.RemotePeer())
	cn.connector().needed.Failed(c.RemotePeer())
}
