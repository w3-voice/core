package core

import (
	"math/rand"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	bf "github.com/libp2p/go-libp2p/p2p/discovery/backoff"
)

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
	set map[peer.ID]*Info
	mux sync.Mutex
	bfk bf.BackoffFactory
}

func NewPeerSet() *PeerSet {
	ps := &PeerSet{}
	ps.bfk = bf.NewPolynomialBackoff(time.Second, time.Minute*2, bf.NoJitter, time.Second, []float64{0.5, 2, 2.5}, rand.NewSource(0))
	ps.set = make(map[peer.ID]*Info)
	ps.mux = sync.Mutex{}
	return ps
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
		strat := p.bfk()
		p.set[pa.ID] = &Info{process: set, cache: connCacheData{strat: strat, nextTry: time.Now()}, peerInfo: pa, done: false}
	}
}

func (p *PeerSet) Remove(proc string, pid peer.ID) {
	p.mux.Lock()
	defer p.mux.Unlock()
	info, ok := p.set[pid]
	if ok {

		p.subRefCount(info.process, proc)
		_, ok = info.process[proc]
		if !ok {
			delete(p.set, pid)
		}
		return
	}
	// panic("not exist")

}

func (p *PeerSet) Turn(t time.Time) []peer.AddrInfo {
	p.mux.Lock()
	defer p.mux.Unlock()
	res := make([]peer.AddrInfo, 0)
	for key, val := range p.set {
		if val.cache.nextTry.Before(t) && !val.done && !val.working {
			res = append(res, val.peerInfo)
			val.working = true
			p.set[key] = val
		}
	}
	return res
}

func (p *PeerSet) Done(id peer.ID) {
	p.mux.Lock()
	defer p.mux.Unlock()
	info, ok := p.set[id]
	if ok {
		info.done = true
		info.working = false
		info.cache.strat.Reset()
	}
}

func (p *PeerSet) Failed(id peer.ID) {
	p.mux.Lock()
	defer p.mux.Unlock()
	info, ok := p.set[id]
	if ok {
		info.done = false
		info.working = false
		info.cache.nextTry = time.Now().Add(info.cache.strat.Delay())
	}
}

func (p *PeerSet) Force(id peer.ID) {
	p.mux.Lock()
	defer p.mux.Unlock()
	info, ok := p.set[id]
	if ok {
		info.done = false
		info.working = true
	}
}

func (p *PeerSet) Empty() bool {
	p.mux.Lock()
	defer p.mux.Unlock()
	return len(p.set) == 0
}
