package core

import (
	"context"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	bf "github.com/libp2p/go-libp2p/p2p/discovery/backoff"
	ma "github.com/multiformats/go-multiaddr"
)

type Connector interface {
	Need(proc string, p peer.AddrInfo)
	Done(proc string, p peer.ID)
}

func NewConnector(h host.Host) Connector {
	return newConnector(h)
}

var bfk = bf.NewFixedBackoff(250 * time.Millisecond)

type connCacheData struct {
	nextTry time.Time
	strat   bf.BackoffStrategy
}

type Info struct {
	process  map[string]int
	done     bool
	working  bool
	cache    connCacheData
	peerInfo peer.AddrInfo
}

type PeerSet struct {
	set map[peer.ID]Info
	mux sync.Mutex
}

func (p *PeerSet) addRefCount(m map[string]int, proc string) {
	pc, ok := m[proc]
	if ok {
		m[proc] = pc + 1
	} else {
		m[proc] = 1
	}
}
func (p *PeerSet) subRefCount(m map[string]int, proc string) {
	pc, ok := m[proc]
	if ok {
		if pc > 1 {
			m[proc] = pc - 1
			return
		}
		delete(m, proc)
	}
}

func (p *PeerSet) Add(proc string, pa peer.AddrInfo) {
	p.mux.Lock()
	defer p.mux.Unlock()
	info, ok := p.set[pa.ID]
	if ok {
		p.addRefCount(info.process, proc)
	} else {
		set := make(map[string]int)
		set[proc] = 1
		strat := bfk()
		p.set[pa.ID] = Info{process: set, cache: connCacheData{strat: strat, nextTry: time.Now()}, peerInfo: pa, done: false}
	}
}

func (p *PeerSet) Remove(proc string, pid peer.ID) {
	p.mux.Lock()
	defer p.mux.Unlock()
	info, ok := p.set[pid]
	if ok {

		p.subRefCount(info.process, proc)
	}
	_, ok = info.process[proc]
	if !ok {
		delete(p.set, pid)
	}
}

func (p *PeerSet) turn(t time.Time) []peer.AddrInfo {
	p.mux.Lock()
	defer p.mux.Unlock()
	res := make([]peer.AddrInfo, 0)
	for _, val := range p.set {
		if val.cache.nextTry.Before(t) && !val.done && !val.working {
			res = append(res, val.peerInfo)
			val.working = true
		}
	}
	return res
}

func (p *PeerSet) done(id peer.ID) {
	p.mux.Lock()
	defer p.mux.Unlock()
	info, ok := p.set[id]
	if ok {
		info.done = true
		info.working = false
		info.cache.strat.Reset()
	}
}

func (p *PeerSet) fail(id peer.ID) {
	p.mux.Lock()
	defer p.mux.Unlock()
	info, ok := p.set[id]
	if ok {
		info.done = false
		info.working = false
		info.cache.nextTry = time.Now().Add(info.cache.strat.Delay())
	}
}

var _ Connector = (*connector)(nil)

type connector struct {
	h      host.Host
	bfk    bf.BackoffFactory
	needed PeerSet
}

func newConnector(h host.Host) *connector {
	c := connector{}
	c.h = h
	c.needed = PeerSet{set: make(map[peer.ID]Info), mux: sync.Mutex{}}
	c.bfk = bf.NewFixedBackoff(250 * time.Millisecond)
	c.h.Network().Notify((*connectorNotifiee)(&c))
	go c.background(context.Background())
	return &c
}

func (c *connector) background(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Microsecond)
	for {
		select {
		case t := <-ticker.C:
			ticker.Reset(2 * time.Second)
			peers := c.needed.turn(t)
			for _, val := range peers {
				if c.h.Network().Connectedness(val.ID) != network.Connected {
					go func(pi peer.AddrInfo) {
						err := c.h.Connect(context.Background(), pi)
						if err != nil {
							c.needed.fail(pi.ID)
							return
						}
						c.needed.done(pi.ID)
					}(val)
				}
			}

		case <-ctx.Done():
			return
		}
	}

}

func (c *connector) Need(proc string, p peer.AddrInfo) {
	c.h.ConnManager().Protect(p.ID,proc)
	c.needed.Add(proc, p)
}
func (c *connector) Done(proc string, p peer.ID) {
	c.h.ConnManager().Unprotect(p,proc)
	c.needed.Remove(proc, p)
}

type connectorNotifiee connector

func (cn *connectorNotifiee) connector() *connector {
	return (*connector)(cn)
}

func (cn *connectorNotifiee) Listen(network.Network, ma.Multiaddr)      {}
func (cn *connectorNotifiee) ListenClose(network.Network, ma.Multiaddr) {}
func (cn *connectorNotifiee) Connected(n network.Network, c network.Conn) {
	cn.connector().needed.done(c.RemotePeer())
}
func (cn *connectorNotifiee) Disconnected(n network.Network, c network.Conn) {
	cn.connector().needed.fail(c.RemotePeer())
}
