package core

import (
	"context"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

const Timeout = 60 * 5

type DataItem struct{
	nvp *Envelop
	failed bool
}
type Data map[peer.ID][]DataItem

type outbox struct {
	mux     sync.Mutex
	data    Data
	failed  chan *Envelop
	bctx    context.Context
	bcancel context.CancelFunc
}

func newOutBox() *outbox {
	return &outbox{
		mux:     sync.Mutex{},
		data:    make(Data),
		failed:  make(chan *Envelop),
		bctx:    nil,
		bcancel: nil,
	}
}

func (o *outbox) put(key peer.ID, val *Envelop) {
	o.mux.Lock()
	defer o.mux.Unlock()
	o.data[key] = append(o.data[key], DataItem{val, false})
	o.mayStart()
}

func (o *outbox) pop(key peer.ID) []*Envelop {
	o.mux.Lock()
	var msgs []*Envelop
	da, ok := o.data[key]
	if ok {
		delete(o.data, key)
	}
	for _,v := range da {
		msgs = append(msgs, v.nvp)
	}
	o.mux.Unlock()

	o.mayStop()
	return msgs
}

func (o *outbox) mayStart() {
	if o.bctx == nil {
		o.bctx, o.bcancel = context.WithCancel(context.Background())
		go o.background(o.bctx)
	}

}

func (o *outbox) mayStop() {
	o.mux.Lock()
	defer o.mux.Unlock()
	if len(o.data) == 0 && o.bctx != nil {
		o.bcancel()
		o.bctx = nil
		o.bcancel = nil
	}
}

func (o *outbox) background(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	for {
		select {
		case t := <-ticker.C:
			o.mux.Lock()
			tmp := make(Data)
			for k, v := range o.data {
				for _, m := range v {
					if m.nvp.CreatedAt+(Timeout) <= t.UTC().Unix() && !m.failed {
						o.failed <- m.nvp
					}
					tmp[k] = append(tmp[k], m)
				}
			}
			o.data = tmp
			o.mux.Unlock()
			o.mayStop()
		case e:=<-ctx.Done():
			log.Error("context error broke sender",e)
			return
		}

	}
}
