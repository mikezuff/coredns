package llnwdebug

import (
	"net/http"

	"github.com/caddyserver/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/reuseport"
)

func init() { plugin.Register("llnwdebug", setup) }

func setup(c *caddy.Controller) error {
	ld := &LLNWDebug{
		dnsRequests: make(map[string]log),
	}

	c.OnStartup(func() error {
		ln, err := reuseport.Listen("tcp", ":80")
		if err != nil {
			return err
		}

		mux := http.NewServeMux()
		mux.HandleFunc("/redirect", handleRedirect)
		mux.Handle("/resolverinfo", ld)

		go func() { http.Serve(ln, mux) }()
		return nil
	})

	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		return ld
	})
	return nil
}
