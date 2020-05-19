package llnwdebug

import (
	"encoding/hex"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/caddyserver/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/pkg/reuseport"
)

func init() { plugin.Register("llnwdebug", setup) }

/*
TODO:
type shares data between dns and http
request for main page hits resolverinfo endpoint w/ unique dns name
unique dns request comes in, resolver info is saved under key (for a time)
http request comes in, resolver info is retrieved using Request.Host
*/
func setup(c *caddy.Controller) error {
	ld := &LLNWDebug{
		dnsRequests: make(map[string]string),
	}

	c.OnStartup(func() error {
		ln, err := reuseport.Listen("tcp", ":80")
		if err != nil {
			return err
		}

		mux := http.NewServeMux()
		mux.HandleFunc("/redirect", func(w http.ResponseWriter, r *http.Request) {
			b := make([]byte, 4)
			rand.Read(b)
			http.Redirect(w, r, fmt.Sprintf("http://ri-%d-%s.ri.zuffs.net/resolverinfo",
				time.Now().Unix(), hex.EncodeToString(b)), http.StatusFound)
		})
		mux.HandleFunc("/resolverinfo", func(w http.ResponseWriter, r *http.Request) {
			ld.lock.Lock()
			defer ld.lock.Unlock()

			if ri, ok := ld.dnsRequests[r.Host+"."]; ok {
				fmt.Fprintf(w, "HI %s %s ResolverInfo: %#v\n", r.RemoteAddr, r.Host, ri)
				fmt.Printf("HI %s %s ResolverInfo: %#v\n", r.RemoteAddr, r.Host, ri)
			} else {
				fmt.Fprintf(w, "HI %s %s ResolverInfo not found\n", r.RemoteAddr, r.Host)
				fmt.Printf("HI %s %s ResolverInfo not found\n", r.RemoteAddr, r.Host)
			}
		})
		go func() { http.Serve(ln, mux) }()
		return nil
	})

	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		return ld
	})
	return nil
}
