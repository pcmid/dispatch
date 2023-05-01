package dispatch

import (
	"crypto/tls"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/plugin/pkg/parse"
	"github.com/coredns/coredns/plugin/pkg/transport"

	"github.com/pcmid/dispatch/proxy"
)

type Matcher struct {
	froms       []string
	dt          atomic.Pointer[DomainTree]
	proxies     []*proxy.Proxy
	maxfails    uint32
	healthcheck time.Duration

	reload time.Duration
	stop   chan struct{}
}

func NewMatchers(c *caddy.Controller) ([]*Matcher, error) {
	var matchers []*Matcher

	for c.Next() {
		u, err := newMatch(c)
		if err != nil {
			return nil, err
		}
		matchers = append(matchers, u)
	}

	if matchers == nil {
		panic("nil upstreams")
	}
	return matchers, nil
}

func newMatch(c *caddy.Controller) (*Matcher, error) {
	m := &Matcher{
		healthcheck: 10 * time.Second,
		stop:        make(chan struct{}),
	}

	if err := m.parseFrom(c); err != nil {
		return nil, err
	}

	for c.NextBlock() {
		if err := m.parseBlock(c); err != nil {
			return nil, err
		}
	}

	return m, nil
}

func (m *Matcher) parseFrom(c *caddy.Controller) (err error) {
	m.froms = c.RemainingArgs()
	n := len(m.froms)
	if n == 0 {
		return c.ArgErr()
	}

	m.rebuildDt()
	return
}

func (m *Matcher) rebuildDt() {
	global := NewDomainTree()
	for _, from := range m.froms {
		log.Infof("found from: %s", from)
		var (
			dt  *DomainTree
			err error
		)
		if strings.HasPrefix(from, "http://") || strings.HasPrefix(from, "https://") {
			dt, err = NewTreeFromUrl(from)
			if err != nil {
				log.Errorf("failed to parse from url: %s: %s", from, err)
				continue
			}
		} else {
			dt, err = NewTreeFromFile(from)
			if err != nil {
				log.Errorf("failed to parse from file: %s: %s", from, err)
				continue
			}
		}

		global.Merge(dt)
	}

	m.dt.Store(global)
}

func (m *Matcher) parseBlock(c *caddy.Controller) error {
	switch dir := c.Val(); dir {
	case "to":
		if err := m.parseTo(c); err != nil {
			return err
		}
	case "maxfails":
		if err := m.parseMaxfails(c); err != nil {
			return err
		}
	case "healthcheck":
		if err := m.parseHealthcheck(c); err != nil {
			return err
		}
	case "reload":
		if err := m.parseReload(c); err != nil {
			return err
		}
	default:
		if len(c.RemainingArgs()) != 0 {
			log.Errorf("unknown arg: %q", dir)
		}
	}

	return nil
}

func (m *Matcher) parseTo(c *caddy.Controller) error {
	args := c.RemainingArgs()
	if len(args) == 0 {
		return c.ArgErr()
	}

	m.proxies = make([]*proxy.Proxy, 0, len(args))

	for _, arg := range args {
		server, name, found := strings.Cut(arg, "@")

		ss, err := parse.HostPortOrFile(server)
		if err != nil {
			return err
		}
		trans, addr := parse.Transport(ss[0])

		upstream := proxy.NewProxy(addr, trans)
		if found && (trans == transport.TLS || trans == transport.GRPC || trans == transport.HTTPS) {
			upstream.SetTLSConfig(&tls.Config{
				ServerName: name,
			})
		}

		m.proxies = append(m.proxies, upstream)
	}

	return nil
}

func (m *Matcher) parseMaxfails(c *caddy.Controller) error {
	args := c.RemainingArgs()
	if len(args) != 1 {
		return c.ArgErr()
	}

	maxfails, err := strconv.Atoi(args[0])
	if err != nil {
		return err
	}

	m.maxfails = uint32(maxfails)
	return nil
}

func (m *Matcher) parseHealthcheck(c *caddy.Controller) error {
	args := c.RemainingArgs()
	if len(args) != 1 {
		return c.ArgErr()
	}

	duration, err := time.ParseDuration(args[0])
	if err != nil {
		return err
	}
	if duration <= 0 {
		return Error(fmt.Errorf("healthcheck interval must be great than 0: %s", c.ArgErr()))
	}

	m.healthcheck = duration
	return nil
}

func (m *Matcher) parseReload(c *caddy.Controller) error {
	args := c.RemainingArgs()
	if len(args) != 1 {
		return c.ArgErr()
	}

	duration, err := time.ParseDuration(args[0])
	if err != nil {
		return err
	}
	if duration <= 0 {
		return Error(fmt.Errorf("reload interval must be great than 0: %s", c.ArgErr()))
	}

	m.reload = duration
	return nil
}

func (m *Matcher) Start() {
	for _, p := range m.proxies {
		p.Start(m.healthcheck)
		p.Healthcheck()
	}

	if m.reload > 0 {
		go func() {
			reload := time.NewTicker(m.reload)

			for {
				select {
				case <-reload.C:
					m.rebuildDt()
				case <-m.stop:
					log.Infof("match stop watch: %s", m.froms)
					return
				}
			}
		}()
	}
}

func (m *Matcher) Stop() {
	for _, p := range m.proxies {
		p.Stop()
	}

	close(m.stop)
}
