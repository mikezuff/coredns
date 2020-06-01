package llnwdebug

import (
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/caddyserver/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/reuseport"
)

func init() { plugin.Register("llnwdebug", setup) }

func setup(c *caddy.Controller) error {
	a4, a6, err := getAnswers()
	if err != nil {
		return err
	}

	ld := &LLNWDebug{
		dnsRequests: make(map[string]log),
		answers4:    a4,
		answers6:    a6,
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

func getAnswers() (a4, a6 []net.IP, err error) {
	hostname, err := os.Hostname()
	if err != nil {
		return
	}

	addrs, err := net.LookupHost(hostname)
	if err != nil {
		return
	}

	if len(addrs) == 0 {
		err = fmt.Errorf("host %s has no addresses in the DNS", hostname)
	}

	for _, addr := range addrs {
		ip := net.ParseIP(addr)
		if ip == nil {
			fmt.Fprintf(os.Stderr, "Ignoring invalid %s address %s\n", hostname, addr)
			continue
		}
		if ip4 := ip.To4(); ip4 != nil {
			a4 = append(a4, ip4)
		} else {
			a6 = append(a6, ip)
		}
	}

	return
}
