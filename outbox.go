package core

import (
	"context"
	"sync"
	"time"

	"github.com/hood-chat/core/entity"
	"github.com/libp2p/go-libp2p/core/peer"
)

const Timeout = 60 * 5

type Data map[peer.ID][]*entity.Envelop

type outbox struct {
	mux     sync.Mutex
	data    Data
	failed  chan *entity.Envelop
	bctx    context.Context
	bcancel context.CancelFunc
}

func newOutBox() *outbox {
	return &outbox{
		mux:     sync.Mutex{},
		data:    make(Data),
		failed:  make(chan *entity.Envelop),
		bctx:    nil,
		bcancel: nil,
	}
}

func (o *outbox) put(key peer.ID, val *entity.Envelop) {
	o.mux.Lock()
	defer o.mux.Unlock()
	o.data[key] = append(o.data[key], val)
	o.mayStart()
}

func (o *outbox) pop(key peer.ID) []*entity.Envelop {
	o.mux.Lock()
	msgs, ok := o.data[key]
	if ok {
		delete(o.data, key)
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
			tmp := make(map[peer.ID][]*entity.Envelop)
			for k, v := range o.data {
				for _, m := range v {
					if m.Message.CreatedAt+(Timeout) <= t.UTC().Unix() {
						o.failed <- m
					} else {
						tmp[k] = append(tmp[k], m)
					}
				}
			}
			o.data = tmp
			o.mux.Unlock()
			o.mayStop()
		case <-ctx.Done():
			log.Debug("context error broke sender")
			return
		}

	}
}
