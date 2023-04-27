package dispatch

import (
	"context"
	"sync"

	"github.com/miekg/dns"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/request"

	"github.com/pcmid/dispatch/proxy"
)

type Dispatch struct {
	Next     plugin.Handler
	Matchers []*Matcher
}

func (d *Dispatch) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	if len(r.Question) == 0 {
		return plugin.NextOrFailure(d.Name(), d.Next, ctx, w, r)
	}

	var (
		respChan = make(chan *dns.Msg)
		once     = sync.Once{}
	)
	defer close(respChan)

	for _, m := range d.Matchers {
		if m.dt.Has(r.Question[0].Name) {
			for _, p := range m.proxies {
				if p.Down(m.maxfails) {
					log.Debugf("skip %s for failed count over %d", p.Addr(), m.maxfails)
					continue
				}

				go func(p *proxy.Proxy) {
					msg := r.Copy()
					state := request.Request{
						Req: msg,
						W:   w,
					}
					resp, err := p.Connect(ctx, state, proxy.Options{})
					if err != nil {
						log.Errorf("failed to query %s: %s", msg.Question[0].Name, err)
						return
					}
					once.Do(func() {
						log.Debugf("get first resp from %s: %d, %s", p.Addr(), resp.MsgHdr.Id, resp.Question[0].Name)
						respChan <- resp
					})
				}(p)
			}

			select {
			case resp := <-respChan:
				if resp.MsgHdr.Id != r.MsgHdr.Id {
					resp.MsgHdr.Id = r.MsgHdr.Id
				}
				err := w.WriteMsg(resp)
				if err != nil {
					return dns.RcodeServerFailure, err
				}
				return dns.RcodeSuccess, nil
			case <-ctx.Done():
				return dns.RcodeServerFailure, ctx.Err()
			}
		}
	}

	return plugin.NextOrFailure(d.Name(), d.Next, ctx, w, r)
}

func (d *Dispatch) Name() string {
	return _PluginName_
}

func (d *Dispatch) OnStartup() error {
	for _, m := range d.Matchers {
		for _, p := range m.proxies {
			p.Start(m.healthcheck)
			p.Healthcheck()
		}
	}
	return nil
}

func (d *Dispatch) OnShutdown() error {
	for _, m := range d.Matchers {
		for _, p := range m.proxies {
			p.Stop()
		}
	}
	return nil
}
