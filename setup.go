package fallbacklocal

import (
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	clog "github.com/coredns/coredns/plugin/pkg/log"
	"github.com/caddyserver/caddy"
)

var log = clog.NewWithPlugin("fallback_local")

func init() {
	caddy.RegisterPlugin("fallback_local", caddy.Plugin{
		ServerType: "dns",
		Action:     setup,
	})
}

func setup(c *caddy.Controller) error {
	handler := FallbackLocal{}
	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		handler.Next = next
		return handler
	})
	return nil
}
