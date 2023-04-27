package dispatch

import (
	"crypto/tls"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/plugin/pkg/parse"
	"github.com/coredns/coredns/plugin/pkg/transport"

	"github.com/pcmid/dispatch/proxy"
)

type Matcher struct {
	dt          *DomainTree
	proxies     []*proxy.Proxy
	maxfails    uint32
	healthcheck time.Duration
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
	forms := c.RemainingArgs()
	n := len(forms)
	if n == 0 {
		return c.ArgErr()
	}

	if m.dt == nil {
		m.dt = NewDomainTree()
	}
	for _, from := range forms {
		log.Infof("found from: %s", from)
		var dt *DomainTree
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

		m.dt.Merge(dt)
	}

	return nil
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

	seconds, err := strconv.Atoi(args[0])
	if err != nil {
		return err
	}
	if seconds <= 0 {
		return Error(fmt.Errorf("healthcheck must be great than 0: %s", c.ArgErr()))
	}

	m.healthcheck = time.Duration(seconds) * time.Second
	return nil
}
