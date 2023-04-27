package dispatch

import (
	"github.com/coredns/caddy"

	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
)

func init() { plugin.Register(_PluginName_, setup) }

func setup(c *caddy.Controller) error {
	log.Infof("Initializing, version %v, HEAD %v", version, commit)

	ms, err := NewMatchers(c)
	if err != nil {
		return Error(err)
	}

	d := &Dispatch{Matchers: ms}
	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		d.Next = next
		return d
	})

	c.OnStartup(func() error {
		return d.OnStartup()
	})

	c.OnShutdown(func() error {
		return d.OnShutdown()
	})

	return nil
}
