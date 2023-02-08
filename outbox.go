package core

import (
	"context"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

type Data map[peer.ID]map[string]Expiry

func (d Data) Add(key peer.ID, val Expiry) {
	if mp, ok := d[key]; !ok { 
		m := make(map[string]Expiry)
		m[val.id()] = val
		d[key] = m
	} else {
		mp[val.id()] = val
	}
}

type Config struct {
	// Keep the envelop after timeout
	Keep bool

	// How long it take to fail an envelop
	Timeout time.Duration

	// duration of Ticker
	Interval time.Duration
}

type outbox struct {
	conf    Config
	mux     sync.Mutex
	active  Data
	passive Data
	failed  chan Expiry
	ticker  *time.Ticker
	paused  bool
}

func NewOutBox(ctx context.Context, conf Config) OutBox {
	o := &outbox{
		conf:    conf,
		mux:     sync.Mutex{},
		active:  make(Data),
		passive: make(Data),
		failed:  make(chan Expiry),
		paused:  true,
		ticker:  time.NewTicker(1 * time.Second),
	}
	o.ticker.Stop()
	go o.background(ctx)
	return o
}

func (o *outbox) Put(key peer.ID, val Expiry) {
	o.mux.Lock()
	defer o.mux.Unlock()
	o.active.Add(key, val)
	o.adjustTicker()
}

func (o *outbox) Pop(key peer.ID) []Expiry {
	o.mux.Lock()
	var msgs []Expiry
	for _, v := range []Data{o.active, o.passive} {
		da, ok := v[key]
		if ok {
			delete(v, key)
		}
		for _, v := range da {
			msgs = append(msgs, v)
		}
	}

	o.mux.Unlock()
	o.adjustTicker()
	return msgs
}

func (o *outbox) C() chan Expiry {
	return o.failed
}

func (o *outbox) adjustTicker() {
	if o.paused && len(o.active) > 0 {
		o.paused = false
		o.ticker.Reset(o.conf.Interval)
		return
	}
	if !o.paused && len(o.active) == 0 {
		o.paused = true
		o.ticker.Stop()
	}
}

func (o *outbox) background(ctx context.Context) {
	for {
		select {
		case t := <-o.ticker.C:
			o.mux.Lock()
			for k, v := range o.active {
				for sk, sm := range v {
					if t.After(sm.createdAt().Add(o.conf.Timeout)) {
						delete(v, sk)
						o.failed <- sm
						o.passive.Add(k, sm)
					}
				}
			}
			o.mux.Unlock()
			o.adjustTicker()
		case e := <-ctx.Done():
			log.Error("context error broke sender", e)
			return
		}

	}
}
